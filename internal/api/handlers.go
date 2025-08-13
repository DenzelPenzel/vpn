package api

import (
	"fmt"
	"regexp"

	"github.com/denzelpenzel/vpn/internal/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// registerHandler handles user registration
func (s *Server) registerHandler(ctx *fasthttp.RequestCtx) {
	var req models.UserRegistration
	if err := s.parseJSONBody(ctx, &req); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Validate input
	if err := s.validateRegistration(&req); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	// Check if email already exists
	exists, err := s.userService.EmailExists(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to check email existence", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Internal server error")
		return
	}

	if exists {
		s.sendErrorResponse(ctx, fasthttp.StatusConflict, "Email already registered")
		return
	}

	// Hash password
	passwordHash, err := s.authService.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Internal server error")
		return
	}

	// Create user
	user, err := s.userService.CreateUser(ctx, req.Email, passwordHash)
	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate JWT token
	token, err := s.authService.GenerateToken(user.ID, user.Email)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Internal server error")
		return
	}

	// Return user data and token
	response := map[string]interface{}{
		"user":  s.userService.ToUserResponse(user),
		"token": token,
	}

	s.sendSuccessResponse(ctx, response)
}

// loginHandler handles user login
func (s *Server) loginHandler(ctx *fasthttp.RequestCtx) {
	var req models.UserLogin
	if err := s.parseJSONBody(ctx, &req); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Validate input
	if err := s.validateLogin(&req); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	// Get user by email
	user, err := s.userService.GetUserByEmail(ctx, req.Email)
	if err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Verify password
	if err := s.authService.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate JWT token
	token, err := s.authService.GenerateToken(user.ID, user.Email)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Internal server error")
		return
	}

	// Return user data and token
	response := map[string]interface{}{
		"user":  s.userService.ToUserResponse(user),
		"token": token,
	}

	s.sendSuccessResponse(ctx, response)
}

// getConfigHandler handles WireGuard config generation
func (s *Server) getConfigHandler(ctx *fasthttp.RequestCtx) {
	// Get user ID from context (set by auth middleware)
	userID, ok := ctx.UserValue("user_id").(uuid.UUID)
	if !ok {
		s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Invalid user context")
		return
	}

	// Parse request body for config request
	var req models.ConfigRequest
	if err := s.parseJSONBody(ctx, &req); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	// Validate public key
	if err := s.wireguardService.ValidatePublicKey(req.PublicKey); err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Invalid public key: %v", err))
		return
	}

	// Parse server ID
	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		s.sendErrorResponse(ctx, fasthttp.StatusBadRequest, "Invalid server ID")
		return
	}

	// Add user key to server
	userKey, err := s.wireguardService.AddUserKey(ctx, userID, serverID, req.PublicKey)
	if err != nil {
		s.logger.Error("Failed to add user key", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Failed to configure VPN")
		return
	}

	// Get server information for response
	server, err := s.serverService.GetServerByID(ctx, serverID)
	if err != nil {
		s.logger.Error("Failed to get server", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusNotFound, "Server not found")
		return
	}

	// Create config response
	config := models.WireGuardConfig{
		Interface: models.WireGuardInterface{
			PrivateKey: "[CLIENT_PRIVATE_KEY]", // Client should replace this
			Address:    userKey.AllowedIPs,
			DNS:        "1.1.1.1, 8.8.8.8",
		},
		Peer: models.WireGuardPeer{
			PublicKey:  server.PublicKey,
			Endpoint:   fmt.Sprintf("%s:%d", server.Endpoint, server.Port),
			AllowedIPs: "0.0.0.0/0, ::/0",
		},
	}

	s.sendSuccessResponse(ctx, config)
}

// getServersHandler handles server locations listing
func (s *Server) getServersHandler(ctx *fasthttp.RequestCtx) {
	// Get active servers
	servers, err := s.serverService.GetActiveServers(ctx)
	if err != nil {
		s.logger.Error("Failed to get servers", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Failed to get servers")
		return
	}

	s.sendSuccessResponse(ctx, servers)
}

// validateRegistration validates user registration input
func (s *Server) validateRegistration(req *models.UserRegistration) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	if !s.isValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	// Additional password strength validation
	if !s.isStrongPassword(req.Password) {
		return fmt.Errorf("password must contain at least one uppercase letter, one lowercase letter, and one number")
	}

	return nil
}

// validateLogin validates user login input
func (s *Server) validateLogin(req *models.UserLogin) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	if !s.isValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	return nil
}

// isValidEmail validates email format
func (s *Server) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isStrongPassword validates password strength
func (s *Server) isStrongPassword(password string) bool {
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	
	return hasUpper && hasLower && hasNumber
}

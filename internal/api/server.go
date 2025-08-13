package api

import (
	"context"
	"time"

	"github.com/denzelpenzel/vpn/internal/config"
	"github.com/denzelpenzel/vpn/internal/services"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// Server represents the API server
type Server struct {
	config           *config.Config
	logger           *zap.Logger
	userService      *services.UserService
	authService      *services.AuthService
	wireguardService *services.WireguardService
	serverService    *services.ServerService
	router           *router.Router
	server           *fasthttp.Server
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	logger *zap.Logger,
	userService *services.UserService,
	authService *services.AuthService,
	wireguardService *services.WireguardService,
	serverService *services.ServerService,
) *Server {
	s := &Server{
		config:           cfg,
		logger:           logger,
		userService:      userService,
		authService:      authService,
		wireguardService: wireguardService,
		serverService:    serverService,
		router:           router.New(),
	}

	s.setupRoutes()
	s.setupServer()

	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Security middleware for all routes
	s.router.GlobalOPTIONS = s.corsHandler

	// Public routes (no authentication required)
	s.router.POST("/api/users/register", s.withMiddleware(s.registerHandler))
	s.router.POST("/api/users/login", s.withMiddleware(s.loginHandler))

	// Protected routes (authentication required)
	s.router.GET("/api/client/config", s.withMiddleware(s.authMiddleware(s.getConfigHandler)))
	s.router.GET("/api/servers/locations", s.withMiddleware(s.authMiddleware(s.getServersHandler)))

	// Health check endpoint
	s.router.GET("/api/health", s.withMiddleware(s.healthHandler))
}

// setupServer configures the FastHTTP server
func (s *Server) setupServer() {
	s.server = &fasthttp.Server{
		Handler:                       s.router.Handler,
		Name:                          "VPN-Service",
		ReadTimeout:                   10 * time.Second,
		WriteTimeout:                  10 * time.Second,
		IdleTimeout:                   60 * time.Second,
		MaxRequestBodySize:            1024 * 1024, // 1MB
		DisableHeaderNamesNormalizing: true,
		NoDefaultServerHeader:         true,
		NoDefaultDate:                 true,
		NoDefaultContentType:          true,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	s.logger.Info("Starting API server",
		zap.String("address", s.config.Server.Address),
		zap.String("environment", s.config.Server.Environment))

	return s.server.ListenAndServe(s.config.Server.Address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down API server")
	return s.server.ShutdownWithContext(ctx)
}

// withMiddleware wraps handlers with common middleware
func (s *Server) withMiddleware(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	return s.loggingMiddleware(
		s.securityMiddleware(
			s.rateLimitMiddleware(handler),
		),
	)
}

// corsHandler handles CORS preflight requests
func (s *Server) corsHandler(ctx *fasthttp.RequestCtx) {
	s.setCORSHeaders(ctx)
	ctx.SetStatusCode(fasthttp.StatusOK)
}

// setCORSHeaders sets CORS headers for security
func (s *Server) setCORSHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	ctx.Response.Header.Set("Access-Control-Max-Age", "86400")
}

// healthHandler handles health check requests
func (s *Server) healthHandler(ctx *fasthttp.RequestCtx) {
	s.setCORSHeaders(ctx)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	response := `{"status":"healthy","service":"vpn-api","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`
	ctx.SetBodyString(response)
}

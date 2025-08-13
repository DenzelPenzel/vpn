package api

import (
	"encoding/json"
	"testing"

	"github.com/denzelpenzel/vpn/internal/config"
	"github.com/denzelpenzel/vpn/internal/models"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// MockUserService for testing
type MockUserService struct{}

func (m *MockUserService) CreateUser(ctx *fasthttp.RequestCtx, email, passwordHash string) (*models.User, error) {
	return &models.User{Email: email}, nil
}

func (m *MockUserService) GetUserByEmail(ctx *fasthttp.RequestCtx, email string) (*models.User, error) {
	return &models.User{Email: email, PasswordHash: "$2a$12$test"}, nil
}

func (m *MockUserService) EmailExists(ctx *fasthttp.RequestCtx, email string) (bool, error) {
	return false, nil
}

func (m *MockUserService) ToUserResponse(user *models.User) *models.UserResponse {
	return &models.UserResponse{Email: user.Email}
}

// MockAuthService for testing
type MockAuthService struct{}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	return "$2a$12$test", nil
}

func (m *MockAuthService) VerifyPassword(password, hash string) error {
	return nil
}

func (m *MockAuthService) GenerateToken(userID, email string) (string, error) {
	return "test-jwt-token", nil
}

func TestHealthHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{}
	
	server := &Server{
		config: cfg,
		logger: logger,
	}

	ctx := &fasthttp.RequestCtx{}
	server.healthHandler(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(ctx.Response.Body(), &response); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestRegisterHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &config.Config{}
	
	server := &Server{
		config:      cfg,
		logger:      logger,
		userService: &MockUserService{},
		authService: &MockAuthService{},
	}

	// Test valid registration
	reqBody := models.UserRegistration{
		Email:    "test@example.com",
		Password: "SecurePass123",
	}
	
	jsonBody, _ := json.Marshal(reqBody)
	
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBody(jsonBody)
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.SetMethod("POST")

	server.registerHandler(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Errorf("Expected status 200, got %d", ctx.Response.StatusCode())
	}
}

func TestValidateRegistration(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name    string
		req     *models.UserRegistration
		wantErr bool
	}{
		{
			name: "valid registration",
			req: &models.UserRegistration{
				Email:    "test@example.com",
				Password: "SecurePass123",
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			req: &models.UserRegistration{
				Email:    "invalid-email",
				Password: "SecurePass123",
			},
			wantErr: true,
		},
		{
			name: "weak password",
			req: &models.UserRegistration{
				Email:    "test@example.com",
				Password: "weak",
			},
			wantErr: true,
		},
		{
			name: "password without uppercase",
			req: &models.UserRegistration{
				Email:    "test@example.com",
				Password: "securepass123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.validateRegistration(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegistration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

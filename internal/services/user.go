package services

import (
	"context"
	"fmt"

	"github.com/denzelpenzel/vpn/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// UserService handles user-related operations
type UserService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewUserService creates a new user service
func NewUserService(db *pgxpool.Pool, logger *zap.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{}

	query := `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at, is_active
	`

	err := s.db.QueryRow(ctx, query, email, passwordHash).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		s.logger.Error("Failed to create user", zap.Error(err), zap.String("email", email))
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User created successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email))

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE email = $1 AND is_active = true
	`

	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		s.logger.Warn("User not found", zap.String("email", email))
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, email, password_hash, created_at, updated_at, is_active
		FROM users
		WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
	)

	if err != nil {
		s.logger.Warn("User not found", zap.String("user_id", userID.String()))
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// EmailExists checks if an email already exists
func (s *UserService) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool

	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	err := s.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		s.logger.Error("Failed to check email existence", zap.Error(err))
		return false, fmt.Errorf("failed to check email: %w", err)
	}

	return exists, nil
}

// ToUserResponse converts User to UserResponse (removes sensitive data)
func (s *UserService) ToUserResponse(user *models.User) *models.UserResponse {
	return &models.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		IsActive:  user.IsActive,
	}
}

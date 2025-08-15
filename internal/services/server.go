package services

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/denzelpenzel/vpn/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ServerService handles server-related operations
type ServerService struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewServerService creates a new server service
func NewServerService(db *pgxpool.Pool, logger *zap.Logger) *ServerService {
	return &ServerService{
		db:     db,
		logger: logger,
	}
}

// GetActiveServers retrieves all active VPN servers
func (s *ServerService) GetActiveServers(ctx context.Context) ([]*models.ServerResponse, error) {
	query := `
		SELECT id, name, location, endpoint, public_key, port
		FROM servers
		WHERE is_active = true
		ORDER BY location, name
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		s.logger.Error("Failed to query servers", zap.Error(err))
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}
	defer rows.Close()

	var servers []*models.ServerResponse
	for rows.Next() {
		server := &models.ServerResponse{}
		err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Location,
			&server.Endpoint,
			&server.PublicKey,
			&server.Port,
		)
		if err != nil {
			s.logger.Error("Failed to scan server row", zap.Error(err))
			continue
		}
		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating server rows", zap.Error(err))
		return nil, fmt.Errorf("failed to iterate servers: %w", err)
	}

	s.logger.Info("Retrieved active servers", zap.Int("count", len(servers)))
	return servers, nil
}

// GetServerByID retrieves a server by ID
func (s *ServerService) GetServerByID(ctx context.Context, serverID uuid.UUID) (*models.Server, error) {
	server := &models.Server{}
	query := `
		SELECT id, name, location, endpoint, public_key, port, is_active, created_at, updated_at
		FROM servers
		WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(ctx, query, serverID).Scan(
		&server.ID,
		&server.Name,
		&server.Location,
		&server.Endpoint,
		&server.PublicKey,
		&server.Port,
		&server.IsActive,
		&server.CreatedAt,
		&server.UpdatedAt,
	)

	if err != nil {
		s.logger.Warn("Server not found", zap.String("server_id", serverID.String()))
		return nil, fmt.Errorf("server not found")
	}

	return server, nil
}

// CreateServer creates a new VPN server (admin function)
func (s *ServerService) CreateServer(ctx context.Context, name, location, endpoint, publicKey string, port int) (*models.Server, error) {
	server := &models.Server{}
	query := `
		INSERT INTO servers (name, location, endpoint, public_key, port)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, location, endpoint, public_key, port, is_active, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query, name, location, endpoint, publicKey, port).Scan(
		&server.ID,
		&server.Name,
		&server.Location,
		&server.Endpoint,
		&server.PublicKey,
		&server.Port,
		&server.IsActive,
		&server.CreatedAt,
		&server.UpdatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create server", zap.Error(err))
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	s.logger.Info("Server created successfully",
		zap.String("server_id", server.ID.String()),
		zap.String("name", name),
		zap.String("location", location))

	return server, nil
}

// InitializeDefaultServers creates default servers if none exist
// SyncServerPublicKey reads the server's public key from a file and updates the database.
func (s *ServerService) SyncServerPublicKey(ctx context.Context, keyFilePath string, serverID uuid.UUID) error {
	keyBytes, err := os.ReadFile(keyFilePath)
	if err != nil {
		s.logger.Warn("Could not read public key file", zap.String("path", keyFilePath), zap.Error(err))
		return fmt.Errorf("could not read public key file: %w", err)
	}
	publicKey := strings.TrimSpace(string(keyBytes))

	if publicKey == "" {
		s.logger.Warn("Public key file is empty", zap.String("path", keyFilePath))
		return fmt.Errorf("public key file is empty")
	}

	query := `UPDATE servers SET public_key = $1, updated_at = NOW() WHERE id = $2 AND (public_key IS NULL OR public_key != $1)`
	result, err := s.db.Exec(ctx, query, publicKey, serverID)
	if err != nil {
		s.logger.Error("Failed to update server public key in database", zap.Error(err))
		return fmt.Errorf("failed to update server public key: %w", err)
	}

	if result.RowsAffected() > 0 {
		s.logger.Info("Successfully synchronized server public key with database", zap.String("server_id", serverID.String()))
	} else {
		s.logger.Info("Server public key is already up-to-date in the database", zap.String("server_id", serverID.String()))
	}

	return nil
}

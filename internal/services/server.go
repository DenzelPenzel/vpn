package services

import (
	"context"
	"fmt"

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
		SELECT id, name, location, endpoint, public_key, private_key, port, is_active, created_at, updated_at
		FROM servers
		WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(ctx, query, serverID).Scan(
		&server.ID,
		&server.Name,
		&server.Location,
		&server.Endpoint,
		&server.PublicKey,
		&server.PrivateKey,
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
func (s *ServerService) CreateServer(ctx context.Context, name, location, endpoint, publicKey, privateKey string, port int) (*models.Server, error) {
	server := &models.Server{}
	query := `
		INSERT INTO servers (name, location, endpoint, public_key, private_key, port)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, location, endpoint, public_key, private_key, port, is_active, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query, name, location, endpoint, publicKey, privateKey, port).Scan(
		&server.ID,
		&server.Name,
		&server.Location,
		&server.Endpoint,
		&server.PublicKey,
		&server.PrivateKey,
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
func (s *ServerService) InitializeDefaultServers(ctx context.Context, wgService *WireguardService) error {
	var count int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check server count: %w", err)
	}

	if count > 0 {
		s.logger.Info("Servers already exist, skipping initialization")
		return nil
	}

	// Create default servers
	defaultServers := []struct {
		name     string
		location string
		endpoint string
	}{
		{"US-East-1", "New York, USA", "vpn-us-east.example.com"},
		{"EU-West-1", "London, UK", "vpn-eu-west.example.com"},
		{"Asia-1", "Singapore", "vpn-asia.example.com"},
	}

	for _, srv := range defaultServers {
		privateKey, publicKey, err := wgService.GenerateKeyPair()
		if err != nil {
			s.logger.Error("Failed to generate keys for server", zap.String("name", srv.name), zap.Error(err))
			continue
		}

		_, err = s.CreateServer(ctx, srv.name, srv.location, srv.endpoint, publicKey, privateKey, 51820)
		if err != nil {
			s.logger.Error("Failed to create default server", zap.String("name", srv.name), zap.Error(err))
			continue
		}
	}

	s.logger.Info("Default servers initialized successfully")
	return nil
}

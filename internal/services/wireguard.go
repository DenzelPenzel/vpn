package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/denzelpenzel/vpn/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// WireguardService handles WireGuard-related operations
type WireguardService struct {
	db         *pgxpool.Pool
	logger     *zap.Logger
	wgClient   *wgctrl.Client
	deviceName string // WireGuard interface name (e.g., "wg0")
}

// NewWireguardService creates a new WireGuard service
func NewWireguardService(logger *zap.Logger) (*WireguardService, error) {
	wgClient, err := wgctrl.New()
	if err != nil {
		logger.Error("Failed to create WireGuard client", zap.Error(err))
		return nil, err
	}

	return &WireguardService{
		logger:     logger,
		wgClient:   wgClient,
		deviceName: "wg0", // Default WireGuard interface name
	}, nil
}

// SetDB sets the database connection (called after initialization)
func (s *WireguardService) SetDB(db *pgxpool.Pool) {
	s.db = db
}

// GenerateKeyPair generates a WireGuard key pair
func (s *WireguardService) GenerateKeyPair() (privateKey, publicKey string, err error) {
	// Generate private key (32 random bytes)
	var privKey [32]byte
	if _, err := rand.Read(privKey[:]); err != nil {
		s.logger.Error("Failed to generate private key", zap.Error(err))
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Clamp private key for Curve25519
	privKey[0] &= 248
	privKey[31] &= 127
	privKey[31] |= 64

	// Generate public key from private key
	var pubKey [32]byte
	curve25519.ScalarBaseMult(&pubKey, &privKey)

	// Encode keys to base64
	privateKey = base64.StdEncoding.EncodeToString(privKey[:])
	publicKey = base64.StdEncoding.EncodeToString(pubKey[:])

	s.logger.Info("WireGuard key pair generated successfully")
	return privateKey, publicKey, nil
}

// ValidatePublicKey validates a WireGuard public key format
func (s *WireguardService) ValidatePublicKey(publicKey string) error {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("invalid base64 encoding: %w", err)
	}

	if len(decoded) != 32 {
		return fmt.Errorf("invalid key length: expected 32 bytes, got %d", len(decoded))
	}

	return nil
}

// AddUserKey adds a user's public key to a server and authorizes them in WireGuard
func (s *WireguardService) AddUserKey(ctx context.Context, userID, serverID uuid.UUID, publicKey string) (*models.UserKey, error) {
	// Validate public key
	if err := s.ValidatePublicKey(publicKey); err != nil {
		s.logger.Warn("Invalid public key provided", zap.Error(err))
		return nil, fmt.Errorf("invalid public key: %w", err)
	}

	// Generate IP address for user (simple allocation)
	allowedIPs, err := s.allocateUserIP(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	if err := s.authorizeUserInWireGuard(publicKey, allowedIPs); err != nil {
		s.logger.Error("Failed to authorize user in WireGuard engine",
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.String("public_key", publicKey))
		return nil, fmt.Errorf("failed to authorize user in WireGuard: %w", err)
	}

	userKey := &models.UserKey{}
	query := `
		INSERT INTO user_keys (user_id, server_id, public_key, allowed_ips)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, server_id) 
		DO UPDATE SET 
			public_key = EXCLUDED.public_key,
			allowed_ips = EXCLUDED.allowed_ips,
			updated_at = NOW(),
			is_active = true
		RETURNING id, user_id, server_id, public_key, allowed_ips, created_at, updated_at, is_active
	`

	err = s.db.QueryRow(ctx, query, userID, serverID, publicKey, allowedIPs).Scan(
		&userKey.ID,
		&userKey.UserID,
		&userKey.ServerID,
		&userKey.PublicKey,
		&userKey.AllowedIPs,
		&userKey.CreatedAt,
		&userKey.UpdatedAt,
		&userKey.IsActive,
	)

	if err != nil {
		// If database insert fails, remove the peer from WireGuard
		s.removeUserFromWireGuard(publicKey)
		s.logger.Error("Failed to add user key to database", zap.Error(err))
		return nil, fmt.Errorf("failed to add user key: %w", err)
	}

	s.logger.Info("User authorized in WireGuard and database",
		zap.String("user_id", userID.String()),
		zap.String("server_id", serverID.String()),
		zap.String("allowed_ips", allowedIPs),
		zap.String("public_key", publicKey[:16]+"..."))

	return userKey, nil
}

// GetUserKey retrieves a user's key for a specific server
func (s *WireguardService) GetUserKey(ctx context.Context, userID, serverID uuid.UUID) (*models.UserKey, error) {
	userKey := &models.UserKey{}
	query := `
		SELECT id, user_id, server_id, public_key, allowed_ips, created_at, updated_at, is_active
		FROM user_keys
		WHERE user_id = $1 AND server_id = $2 AND is_active = true
	`

	err := s.db.QueryRow(ctx, query, userID, serverID).Scan(
		&userKey.ID,
		&userKey.UserID,
		&userKey.ServerID,
		&userKey.PublicKey,
		&userKey.AllowedIPs,
		&userKey.CreatedAt,
		&userKey.UpdatedAt,
		&userKey.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("user key not found")
	}

	return userKey, nil
}

// GenerateConfig generates a WireGuard configuration for a user
func (s *WireguardService) GenerateConfig(ctx context.Context, userID, serverID uuid.UUID, clientPrivateKey string) (string, error) {
	// Get server information
	server := &models.Server{}
	serverQuery := `
		SELECT id, name, location, endpoint, public_key, port
		FROM servers
		WHERE id = $1 AND is_active = true
	`

	err := s.db.QueryRow(ctx, serverQuery, serverID).Scan(
		&server.ID,
		&server.Name,
		&server.Location,
		&server.Endpoint,
		&server.PublicKey,
		&server.Port,
	)

	if err != nil {
		s.logger.Error("Server not found", zap.String("server_id", serverID.String()))
		return "", fmt.Errorf("server not found")
	}

	// Get user key information
	userKey, err := s.GetUserKey(ctx, userID, serverID)
	if err != nil {
		return "", fmt.Errorf("user key not found: %w", err)
	}

	// Generate WireGuard configuration
	config := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = %s
Endpoint = %s:%d
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`,
		clientPrivateKey,
		userKey.AllowedIPs,
		server.PublicKey,
		server.Endpoint,
		server.Port,
	)

	s.logger.Info("WireGuard config generated",
		zap.String("user_id", userID.String()),
		zap.String("server_id", serverID.String()))

	return config, nil
}

// allocateUserIP allocates an IP address for a user on a server
func (s *WireguardService) allocateUserIP(ctx context.Context, serverID uuid.UUID) (string, error) {
	var count int
	countQuery := `SELECT COUNT(*) FROM user_keys WHERE server_id = $1 AND is_active = true`

	err := s.db.QueryRow(ctx, countQuery, serverID).Scan(&count)
	if err != nil {
		return "", fmt.Errorf("failed to count existing users: %w", err)
	}

	// Allocate IP in 10.0.0.0/24 range (10.0.0.2 onwards, .1 is server)
	if count >= 253 {
		return "", fmt.Errorf("no available IP addresses")
	}

	ip := fmt.Sprintf("10.0.0.%d/32", count+2)
	return ip, nil
}

// IsValidIPAddress validates if a string is a valid IP address
func (s *WireguardService) IsValidIPAddress(ip string) bool {
	// Remove CIDR notation if present
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}
	return net.ParseIP(ip) != nil
}

// authorizeUserInWireGuard adds a user's public key to the WireGuard interface as an allowed peer
func (s *WireguardService) authorizeUserInWireGuard(publicKey, allowedIPs string) error {
	if s.wgClient == nil {
		s.logger.Warn("WireGuard client not available - skipping peer authorization")
		return fmt.Errorf("WireGuard client not available")
	}

	pubKey, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Parse allowed IPs
	_, allowedIPNet, err := net.ParseCIDR(allowedIPs)
	if err != nil {
		return fmt.Errorf("failed to parse allowed IPs: %w", err)
	}

	// Create peer configuration
	peerConfig := wgtypes.PeerConfig{
		PublicKey:                   pubKey,
		AllowedIPs:                  []net.IPNet{*allowedIPNet},
		ReplaceAllowedIPs:           true,
		PersistentKeepaliveInterval: &[]time.Duration{25 * time.Second}[0],
	}

	// Configure the WireGuard device to add this peer
	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peerConfig},
	}

	err = s.wgClient.ConfigureDevice(s.deviceName, config)
	if err != nil {
		return fmt.Errorf("failed to configure WireGuard device: %w", err)
	}

	s.logger.Info("User authorized in WireGuard engine",
		zap.String("device", s.deviceName),
		zap.String("public_key", publicKey[:16]+"..."),
		zap.String("allowed_ips", allowedIPs))

	return nil
}

// removeUserFromWireGuard removes a user's public key from the WireGuard interface
func (s *WireguardService) removeUserFromWireGuard(publicKey string) error {
	if s.wgClient == nil {
		s.logger.Warn("WireGuard client not available - skipping peer removal")
		return nil // Allow operation to continue for development
	}

	// Parse the public key
	pubKey, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Create peer configuration for removal
	peerConfig := wgtypes.PeerConfig{
		PublicKey: pubKey,
		Remove:    true,
	}

	// Configure the WireGuard device to remove this peer
	config := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{peerConfig},
	}

	// Apply configuration to WireGuard interface
	err = s.wgClient.ConfigureDevice(s.deviceName, config)
	if err != nil {
		return fmt.Errorf("failed to remove peer from WireGuard device: %w", err)
	}

	s.logger.Info("User removed from WireGuard engine",
		zap.String("device", s.deviceName),
		zap.String("public_key", publicKey[:16]+"..."))

	return nil
}

// RemoveUserKey removes a user's key from both database and WireGuard engine
func (s *WireguardService) RemoveUserKey(ctx context.Context, userID, serverID uuid.UUID) error {
	// Get user key first to get public key for WireGuard removal
	userKey, err := s.GetUserKey(ctx, userID, serverID)
	if err != nil {
		return fmt.Errorf("user key not found: %w", err)
	}

	// Remove from WireGuard engine first
	if err := s.removeUserFromWireGuard(userKey.PublicKey); err != nil {
		s.logger.Error("Failed to remove user from WireGuard engine", zap.Error(err))
		// Continue with database removal even if WireGuard removal fails
	}

	// Remove from database
	query := `UPDATE user_keys SET is_active = false, updated_at = NOW() WHERE user_id = $1 AND server_id = $2`
	_, err = s.db.Exec(ctx, query, userID, serverID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user key: %w", err)
	}

	s.logger.Info("User key removed from WireGuard and database",
		zap.String("user_id", userID.String()),
		zap.String("server_id", serverID.String()))

	return nil
}

// ListAuthorizedPeers lists all currently authorized peers in the WireGuard interface
func (s *WireguardService) ListAuthorizedPeers() ([]wgtypes.Peer, error) {
	if s.wgClient == nil {
		return nil, fmt.Errorf("WireGuard client not available")
	}

	device, err := s.wgClient.Device(s.deviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get WireGuard device info: %w", err)
	}

	s.logger.Info("Retrieved WireGuard peers",
		zap.String("device", s.deviceName),
		zap.Int("peer_count", len(device.Peers)))

	return device.Peers, nil
}

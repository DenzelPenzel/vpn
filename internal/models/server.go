package models

import (
	"time"

	"github.com/google/uuid"
)

// Server represents a VPN server
type Server struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Location   string    `json:"location" db:"location"`
	Endpoint   string    `json:"endpoint" db:"endpoint"`
	PublicKey  string    `json:"public_key" db:"public_key"`
	PrivateKey string    `json:"-" db:"private_key"` // Never expose private key in JSON
	Port       int       `json:"port" db:"port"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// ServerResponse represents server response for clients (without private key)
type ServerResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Endpoint  string    `json:"endpoint"`
	PublicKey string    `json:"public_key"`
	Port      int       `json:"port"`
}

// UserKey represents a user's WireGuard key pair association with a server
type UserKey struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	ServerID   uuid.UUID `json:"server_id" db:"server_id"`
	PublicKey  string    `json:"public_key" db:"public_key"`
	AllowedIPs string    `json:"allowed_ips" db:"allowed_ips"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
	IsActive   bool      `json:"is_active" db:"is_active"`
}

// WireGuardConfig represents a complete WireGuard configuration
type WireGuardConfig struct {
	Interface WireGuardInterface `json:"interface"`
	Peer      WireGuardPeer      `json:"peer"`
}

// WireGuardInterface represents the [Interface] section of WireGuard config
type WireGuardInterface struct {
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
	DNS        string `json:"dns"`
}

// WireGuardPeer represents the [Peer] section of WireGuard config
type WireGuardPeer struct {
	PublicKey  string `json:"public_key"`
	Endpoint   string `json:"endpoint"`
	AllowedIPs string `json:"allowed_ips"`
}

// ConfigRequest represents a client config request
type ConfigRequest struct {
	PublicKey string `json:"public_key" validate:"required"`
	ServerID  string `json:"server_id" validate:"required,uuid"`
}

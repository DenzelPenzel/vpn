package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Security SecurityConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Address     string
	Port        int
	Environment string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DSN string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret string
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	BCryptCost int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Address:     getEnv("SERVER_ADDRESS", "0.0.0.0:8080"),
			Port:        getEnvAsInt("SERVER_PORT", 8080),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
		Database: DatabaseConfig{
			DSN: os.Getenv("DATABASE_DSN"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
		},
		Security: SecurityConfig{
			BCryptCost: getEnvAsInt("BCRYPT_COST", 12),
		},
	}

	if cfg.Database.DSN == "" {
		return nil, fmt.Errorf("DSN is required")
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/denzelpenzel/vpn/assets"
	"github.com/denzelpenzel/vpn/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// NewConnection creates a new database connection pool
func NewConnection(cfg config.DatabaseConfig, automigrate bool, logger *zap.Logger) (*pgxpool.Pool, error) {
	// Create connection pool configuration
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Security: Set connection pool limits
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5

	// Create connection pool with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run automigrations if enabled
	if automigrate {
		logger.Info("starting SQL migrations for VPN service...")
		iofsDriver, err := iofs.New(assets.EmbeddedFiles, "migrations")
		if err != nil {
			pool.Close()
			return nil, err
		}

		migrator, err := migrate.NewWithSourceInstance("iofs", iofsDriver, cfg.DSN)
		if err != nil {
			pool.Close()
			return nil, err
		}

		err = migrator.Up()
		switch {
		case errors.Is(err, migrate.ErrNoChange):
			break
		case err != nil:
			pool.Close()
			return nil, err
		}

		version, isDirty, err := migrator.Version()
		if err != nil {
			pool.Close()
			return nil, err
		}

		logger.Info("SQL migrations completed",
			zap.Uint("version", version),
			zap.Bool("dirty_state", isDirty))
	}

	return pool, nil
}

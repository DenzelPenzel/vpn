package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/denzelpenzel/vpn/internal/api"
	"github.com/denzelpenzel/vpn/internal/config"
	"github.com/denzelpenzel/vpn/internal/database"
	"github.com/denzelpenzel/vpn/internal/logger"
	"github.com/denzelpenzel/vpn/internal/services"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func synchronizeKeys(serverService *services.ServerService, logger *zap.Logger) {
	const keyFilePath = "/etc/wireguard/publickey"
	const serverIDStr = "a7f4c3d6-1b3c-4e8b-9f0e-1d2c3b4a5e6f"

	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		logger.Fatal("Failed to parse static server ID", zap.Error(err))
	}

	// Retry logic to wait for the key file to be created by the wireguard container
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		err := serverService.SyncServerPublicKey(context.Background(), keyFilePath, serverID)
		if err == nil {
			logger.Info("Successfully synchronized WireGuard public key.")
			return
		}
		logger.Warn("Failed to sync WireGuard public key, retrying in 5 seconds...", zap.Error(err), zap.Int("attempt", i+1))
		time.Sleep(5 * time.Second)
	}
	logger.Fatal("Failed to synchronize WireGuard public key after multiple retries. Please check the WireGuard container logs.")
}

func main() {

	// Initialize logger
	zapLogger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		zapLogger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize database with automigrations enabled
	db, err := database.NewConnection(cfg.Database, true, zapLogger)
	if err != nil {
		zapLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize services
	userService := services.NewUserService(db, zapLogger)
	authService := services.NewAuthService(cfg.JWT.Secret, zapLogger)
	wireguardService, err := services.NewWireguardService(zapLogger)
	if err != nil {
		zapLogger.Fatal("Failed to initialize WireGuard service", zap.Error(err))
	}
	wireguardService.SetDB(db) // Set database connection
	serverService := services.NewServerService(db, zapLogger)

	// Synchronize WireGuard public key with the database
	// This is done in a retry loop to handle cases where the API starts before the key is generated
	synchronizeKeys(serverService, zapLogger)

	// Initialize API server
	server := api.NewServer(cfg, zapLogger, userService, authService, wireguardService, serverService)

	// Start server in goroutine
	go func() {
		zapLogger.Info("Starting VPN API server", zap.String("address", cfg.Server.Address))

		if err := server.Start(); err != nil {
			zapLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := server.Shutdown(ctx); err != nil {
		zapLogger.Error("Server forced to shutdown", zap.Error(err))
	}

	zapLogger.Info("Server exited")
}

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
	"go.uber.org/zap"
)

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

	// Initialize default servers if needed
	if err := serverService.InitializeDefaultServers(context.Background(), wireguardService); err != nil {
		zapLogger.Warn("Failed to initialize default servers", zap.Error(err))
	}

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

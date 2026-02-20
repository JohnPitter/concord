package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/pkg/version"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.json")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	loggerCfg := observability.LoggerConfig{
		Level:        cfg.GetLogLevel(),
		Format:       cfg.Logging.Format,
		OutputPath:   cfg.Logging.OutputPath,
		ErrorPath:    cfg.Logging.ErrorPath,
		EnableCaller: cfg.Logging.EnableCaller,
		EnableStack:  cfg.Logging.EnableStack,
		Service:      "concord-server",
		Version:      version.Version,
	}
	logger := observability.NewLogger(loggerCfg)

	logger.Info().
		Str("version", version.Version).
		Str("git_commit", version.GitCommit).
		Str("platform", version.Platform).
		Msg("starting Concord server")

	// Initialize metrics
	metrics := observability.NewMetrics()
	logger.Info().Msg("metrics initialized")

	// Initialize health checker
	health := observability.NewHealthChecker(logger, version.Version)
	logger.Info().Msg("health checker initialized")

	// TODO: Initialize database (PostgreSQL)
	// TODO: Initialize Redis
	// TODO: Initialize HTTP server
	// TODO: Initialize WebSocket signaling server
	// TODO: Initialize P2P relay server

	logger.Info().
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Msg("server started successfully")

	// Placeholder server implementation
	logger.Info().Msg("server is running (stub implementation)")
	logger.Info().Msg("press Ctrl+C to stop")

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-sigChan
	logger.Info().Msg("shutdown signal received")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Perform graceful shutdown
	logger.Info().Msg("shutting down server gracefully")

	// TODO: Close database connections
	// TODO: Close Redis connections
	// TODO: Stop HTTP server
	// TODO: Stop WebSocket server
	// TODO: Stop P2P relay

	// Wait for shutdown to complete or timeout
	<-shutdownCtx.Done()

	// Suppress unused variable warnings for now
	_ = ctx
	_ = metrics
	_ = health

	logger.Info().Msg("server shut down successfully")
}

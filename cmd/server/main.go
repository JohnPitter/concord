package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/concord-chat/concord/internal/api"
	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/store/postgres"
	"github.com/concord-chat/concord/internal/store/redis"
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
		Msg("starting Concord central server")

	// Initialize metrics
	metrics := observability.NewMetrics()
	_ = metrics // Will be wired into middleware in future iteration

	// Initialize health checker
	health := observability.NewHealthChecker(logger, version.Version)

	// --- Infrastructure: PostgreSQL ---
	pgDB, err := postgres.New(cfg.Database.Postgres, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("postgresql unavailable — server will start without database")
		pgDB = nil
	} else {
		// Run migrations
		migrator := postgres.NewMigrator(pgDB, logger)
		if err := migrator.Run(context.Background()); err != nil {
			logger.Error().Err(err).Msg("failed to run postgresql migrations")
		}

		// Register PG health check
		health.RegisterCheck("postgresql", observability.DatabaseHealthCheck(pgDB.Ping))

		logger.Info().Msg("postgresql initialized and migrations applied")
	}

	// --- Infrastructure: Redis ---
	var redisClient *redis.Client
	if cfg.Cache.Redis.Enabled {
		redisClient, err = redis.New(cfg.Cache.Redis, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("redis unavailable — server will start without cache")
			redisClient = nil
		} else {
			health.RegisterCheck("redis", observability.RedisHealthCheck(redisClient.Ping))
			logger.Info().Msg("redis initialized")
		}
	}

	// --- JWT Manager ---
	jwtManager, err := auth.NewJWTManager(cfg.Security.JWTSecret)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// --- API Server ---
	// Note: Full service integration with PG-backed repos will come in a future iteration.
	// For now the API server is initialized without services for routes that need PG-backed repos.
	// Auth, server, and chat services are nil; handlers will return 500 if called without them.
	apiServer := api.New(
		cfg.Server,
		nil, // auth service — requires PG-backed repo
		nil, // server service — requires PG-backed repo
		nil, // chat service — requires PG-backed repo
		jwtManager,
		health,
		logger,
	)

	// Start HTTP server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := apiServer.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	logger.Info().
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Msg("concord central server started")

	// --- Graceful shutdown ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case err := <-errCh:
		logger.Error().Err(err).Msg("server error, initiating shutdown")
	}

	// Create shutdown context with configured timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	// Close Redis
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			logger.Error().Err(err).Msg("redis close error")
		}
	}

	// Close PostgreSQL
	if pgDB != nil {
		pgDB.Close()
	}

	logger.Info().Msg("concord central server shut down successfully")
}

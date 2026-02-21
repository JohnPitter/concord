package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/concord-chat/concord/internal/api"
	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/cache"
	"github.com/concord-chat/concord/internal/chat"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/security"
	"github.com/concord-chat/concord/internal/server"
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

	// --- Services ---
	var (
		authSvc   *auth.Service
		serverSvc *server.Service
		chatSvc   *chat.Service
	)

	if pgDB != nil {
		// Bridge pgx pool to database/sql interface
		stdlibDB := pgDB.StdlibDB()
		pgAdapter := postgres.NewAdapter(stdlibDB)

		// Auth service
		githubOAuth := auth.NewGitHubOAuth(cfg.Auth.GitHubClientID, logger)
		authRepo := auth.NewRepository(pgAdapter, logger)
		cryptoMgr := security.NewCryptoManager()
		encryptKey := sha256Key(cfg.Security.JWTSecret)
		authSvc = auth.NewService(githubOAuth, jwtManager, authRepo, cryptoMgr, encryptKey, logger)

		// Server service
		serverRepo := server.NewRepository(pgAdapter, logger)
		serverCache := cache.NewLRU(1000)
		serverSvc = server.NewService(serverRepo, serverCache, logger)

		// Chat service
		chatRepo := chat.NewRepository(pgAdapter, logger)
		chatSvc = chat.NewService(chatRepo, logger)

		logger.Info().Msg("all services initialized with postgresql backend")
	}

	// --- API Server ---
	apiServer := api.New(
		cfg.Server,
		authSvc,
		serverSvc,
		chatSvc,
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

// sha256Key derives a 32-byte key from the JWT secret for AES-256 encryption.
func sha256Key(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

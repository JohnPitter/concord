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
	"time"

	"github.com/concord-chat/concord/internal/api"
	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/cache"
	"github.com/concord-chat/concord/internal/chat"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/friends"
	"github.com/concord-chat/concord/internal/network/signaling"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/presence"
	"github.com/concord-chat/concord/internal/security"
	"github.com/concord-chat/concord/internal/server"
	"github.com/concord-chat/concord/internal/store/postgres"
	"github.com/concord-chat/concord/internal/store/redis"
	"github.com/concord-chat/concord/internal/voice"
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

	// Initialize health checker
	health := observability.NewHealthChecker(logger, version.Version)

	// --- Infrastructure: PostgreSQL (with retry) ---
	var pgDB *postgres.DB
	const maxRetries = 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		pgDB, err = postgres.New(cfg.Database.Postgres, logger)
		if err == nil {
			break
		}
		if attempt == maxRetries {
			logger.Fatal().Err(err).Int("attempts", maxRetries).Msg("postgresql unavailable after retries — cannot start without database")
		}
		wait := time.Duration(attempt) * 2 * time.Second
		logger.Warn().Err(err).Int("attempt", attempt).Dur("retry_in", wait).Msg("postgresql unavailable — retrying")
		time.Sleep(wait)
	}
	{
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

	// --- Services (pgDB is guaranteed non-nil due to retry+fatal above) ---
	stdlibDB := pgDB.StdlibDB()
	pgAdapter := postgres.NewAdapter(stdlibDB)

	// Auth service
	githubOAuth := auth.NewGitHubOAuth(cfg.Auth.GitHubClientID, logger)
	authRepo := auth.NewRepository(pgAdapter, logger)
	cryptoMgr := security.NewCryptoManager()
	encryptKey := sha256Key(cfg.Security.JWTSecret)
	authSvc := auth.NewService(githubOAuth, jwtManager, authRepo, cryptoMgr, encryptKey, logger)

	// Server service
	serverRepo := server.NewRepository(pgAdapter, logger)
	serverCache := cache.NewLRU(1000)
	serverSvc := server.NewService(serverRepo, serverCache, logger)

	// Chat service
	chatRepo := chat.NewRepository(pgAdapter, logger)
	chatSvc := chat.NewService(chatRepo, logger)

	// Friends service — wrap transactions with pgAdapter-style placeholder translation
	friendTx := friends.NewStdlibTransactorWithWrapper(stdlibDB, func(q friends.Querier) friends.Querier {
		return postgres.NewQuerierAdapter(q)
	})
	friendRepo := friends.NewRepository(pgAdapter, friendTx, logger)
	// Keep online status responsive: clients send frequent authenticated polls.
	// 15s avoids stale "online" while still tolerating short jitter.
	presenceTracker := presence.NewTracker(15 * time.Second)
	friendsSvc := friends.NewService(friendRepo, presenceTracker, logger)

	logger.Info().Msg("all services initialized with postgresql backend")

	// --- Signaling Server (voice WebRTC coordination) ---
	sigServer := signaling.NewServer(logger)
	logger.Info().Msg("signaling server initialized")

	// --- API Server ---
	apiServer := api.New(
		cfg.Server,
		authSvc,
		serverSvc,
		chatSvc,
		friendsSvc,
		sigServer,
		jwtManager,
		presenceTracker,
		health,
		metrics,
		logger,
	)

	iceProvider := voice.NewICECredentialsProvider(
		cfg.Voice.TURNHost,
		cfg.Voice.TURNPort,
		cfg.Voice.TURNTLSPort,
		cfg.Voice.TURNSecret,
		cfg.Voice.TURNCredentialTTL,
	)
	if cfg.Voice.TURNEnabled && iceProvider.Enabled() {
		apiServer.SetVoiceICEProvider(iceProvider)
		logger.Info().
			Str("turn_host", cfg.Voice.TURNHost).
			Int("turn_port", cfg.Voice.TURNPort).
			Int("turn_tls_port", cfg.Voice.TURNTLSPort).
			Msg("voice TURN credentials endpoint enabled")
	}

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

	logger.Info().Dur("timeout", cfg.Server.ShutdownTimeout).Msg("starting graceful shutdown — draining in-flight requests")

	// Create shutdown context with configured timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// 1. Stop accepting new connections and drain in-flight requests
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error — some requests may not have completed")
	} else {
		logger.Info().Msg("HTTP server drained and stopped")
	}

	// 2. Close Redis (after HTTP to allow in-flight requests to finish)
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			logger.Error().Err(err).Msg("redis close error")
		} else {
			logger.Info().Msg("redis connection closed")
		}
	}

	// 3. Close PostgreSQL (last, since other services depend on it)
	if pgDB != nil {
		pgDB.Close()
		logger.Info().Msg("postgresql connection closed")
	}

	logger.Info().Msg("concord central server shut down successfully")
}

// sha256Key derives a 32-byte key from the JWT secret for AES-256 encryption.
func sha256Key(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

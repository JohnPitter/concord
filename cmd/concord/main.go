package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/security"
	"github.com/concord-chat/concord/internal/store/sqlite"
	"github.com/concord-chat/concord/pkg/version"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// NOTE: Frontend assets will be embedded when the frontend is built in Phase 1.2
// For now, we use Wails dev mode which serves the frontend directly
// In production, uncomment and properly configure the embed directive:

// App struct holds the application state
type App struct {
	ctx         context.Context
	cfg         *config.Config
	db          *sqlite.DB
	logger      zerolog.Logger
	metrics     *observability.Metrics
	health      *observability.HealthChecker
	authService *auth.Service
}

// NewApp creates a new application instance
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	configPath := filepath.Join(config.Default().App.ConfigDir, "config.json")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	a.cfg = cfg

	// Initialize logger
	loggerCfg := observability.LoggerConfig{
		Level:        cfg.GetLogLevel(),
		Format:       cfg.Logging.Format,
		OutputPath:   cfg.Logging.OutputPath,
		ErrorPath:    cfg.Logging.ErrorPath,
		EnableCaller: cfg.Logging.EnableCaller,
		EnableStack:  cfg.Logging.EnableStack,
		Service:      "concord-desktop",
		Version:      version.Version,
	}
	a.logger = observability.NewLogger(loggerCfg)

	a.logger.Info().
		Str("version", version.Version).
		Str("git_commit", version.GitCommit).
		Str("platform", version.Platform).
		Msg("starting Concord")

	// Initialize metrics
	a.metrics = observability.NewMetrics()
	a.logger.Info().Msg("metrics initialized")

	// Initialize health checker
	a.health = observability.NewHealthChecker(a.logger, version.Version)
	a.logger.Info().Msg("health checker initialized")

	// Initialize database
	dbCfg := sqlite.Config{
		Path:            cfg.Database.SQLite.Path,
		MaxOpenConns:    cfg.Database.SQLite.MaxOpenConns,
		MaxIdleConns:    cfg.Database.SQLite.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.SQLite.ConnMaxLifetime,
		WALMode:         cfg.Database.SQLite.WALMode,
		ForeignKeys:     cfg.Database.SQLite.ForeignKeys,
		BusyTimeout:     cfg.Database.SQLite.BusyTimeout,
	}

	db, err := sqlite.New(dbCfg, a.logger)
	if err != nil {
		a.logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	a.db = db
	a.logger.Info().Str("path", cfg.Database.SQLite.Path).Msg("database initialized")

	// Register database health check
	a.health.RegisterCheck("sqlite", observability.DatabaseHealthCheck(a.db.Ping))

	// Run migrations
	migrator := sqlite.NewMigrator(a.db, a.logger)
	if err := migrator.Migrate(context.Background()); err != nil {
		a.logger.Fatal().Err(err).Msg("failed to run migrations")
	}

	// Initialize auth service
	githubOAuth := auth.NewGitHubOAuth(cfg.Auth.GitHubClientID, a.logger)

	jwtManager, err := auth.NewJWTManager(cfg.Security.JWTSecret)
	if err != nil {
		a.logger.Fatal().Err(err).Msg("failed to initialize JWT manager")
	}

	authRepo := auth.NewRepository(a.db, a.logger)
	cryptoManager := security.NewCryptoManager()

	// Derive 32-byte encryption key from JWT secret for encrypting refresh tokens at rest
	encryptKey := sha256.Sum256([]byte(cfg.Security.JWTSecret))

	a.authService = auth.NewService(githubOAuth, jwtManager, authRepo, cryptoManager, encryptKey[:], a.logger)
	a.logger.Info().Msg("auth service initialized")

	a.logger.Info().Msg("Concord started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.logger.Info().Msg("shutting down Concord")

	// Close database
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error().Err(err).Msg("failed to close database")
		} else {
			a.logger.Info().Msg("database closed")
		}
	}

	a.logger.Info().Msg("Concord shut down successfully")
}

// Greet is a simple test method exposed to the frontend
func (a *App) Greet(name string) string {
	a.logger.Info().Str("name", name).Msg("greet called")
	return fmt.Sprintf("Hello %s! Welcome to Concord v%s", name, version.Version)
}

// GetVersion returns version information
func (a *App) GetVersion() version.Info {
	return version.Get()
}

// GetHealth returns the health status of the application
func (a *App) GetHealth() *observability.Health {
	return a.health.Check(a.ctx)
}

// StartLogin initiates the GitHub Device Flow.
// Returns the device code response for the frontend to display.
func (a *App) StartLogin() (*auth.DeviceCodeResponse, error) {
	return a.authService.StartLogin(a.ctx)
}

// CompleteLogin polls for the GitHub token and creates local user + session.
func (a *App) CompleteLogin(deviceCode string, interval int) (*auth.AuthState, error) {
	return a.authService.CompleteLogin(a.ctx, deviceCode, interval)
}

// RestoreSession attempts to restore an existing session for a user.
func (a *App) RestoreSession(userID string) (*auth.AuthState, error) {
	return a.authService.RestoreSession(a.ctx, userID)
}

// Logout removes all sessions for the current user.
func (a *App) Logout(userID string) error {
	return a.authService.Logout(a.ctx, userID)
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Concord",
		Width:            1200,
		Height:           800,
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 15, A: 255}, // Void theme background
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		// AssetServer will be configured when frontend is built
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

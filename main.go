package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/cache"
	"github.com/concord-chat/concord/internal/chat"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/files"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/security"
	"github.com/concord-chat/concord/internal/server"
	"github.com/concord-chat/concord/internal/store/sqlite"
	"github.com/concord-chat/concord/internal/translation"
	"github.com/concord-chat/concord/internal/voice"
	"github.com/concord-chat/concord/pkg/version"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// App struct holds the application state
type App struct {
	ctx           context.Context
	cfg           *config.Config
	db            *sqlite.DB
	logger        zerolog.Logger
	metrics       *observability.Metrics
	health        *observability.HealthChecker
	authService   *auth.Service
	serverService *server.Service
	chatService   *chat.Service
	voiceEngine   *voice.Engine
	fileService   *files.Service
	translationService *translation.Service
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

	// Initialize LRU cache
	srvCache := cache.NewLRU(cfg.Cache.LRU.MaxEntries)
	a.logger.Info().Int("max_entries", cfg.Cache.LRU.MaxEntries).Msg("LRU cache initialized")

	// Initialize server service
	serverRepo := server.NewRepository(a.db, a.logger)
	a.serverService = server.NewService(serverRepo, srvCache, a.logger)
	a.logger.Info().Msg("server service initialized")

	// Initialize chat service
	chatRepo := chat.NewRepository(a.db, a.logger)
	a.chatService = chat.NewService(chatRepo, a.logger)
	a.logger.Info().Msg("chat service initialized")

	// Initialize file service
	storageDir := filepath.Join(filepath.Dir(cfg.Database.SQLite.Path), "files")
	fileStorage, err := files.NewLocalStorage(storageDir, a.logger)
	if err != nil {
		a.logger.Fatal().Err(err).Msg("failed to initialize file storage")
	}
	fileRepo := files.NewRepository(a.db, a.logger)
	a.fileService = files.NewService(fileRepo, fileStorage, a.logger)
	a.logger.Info().Str("storage_dir", storageDir).Msg("file service initialized")

	// Initialize voice engine
	a.voiceEngine = voice.NewEngine(voice.DefaultEngineConfig(), a.logger)
	a.logger.Info().Msg("voice engine initialized")

	// Initialize translation service
	a.translationService = translation.NewService(cfg.Translation, a.logger)
	a.logger.Info().
		Bool("enabled", cfg.Translation.Enabled).
		Str("default_lang", cfg.Translation.DefaultLang).
		Msg("translation service initialized")

	a.logger.Info().Msg("Concord started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.logger.Info().Msg("shutting down Concord")

	// Stop translation service if active
	if a.translationService != nil {
		a.translationService.StopPipeline()
		a.logger.Info().Msg("translation service stopped")
	}

	// Leave voice channel if connected
	if a.voiceEngine != nil {
		if err := a.voiceEngine.LeaveChannel(); err != nil {
			a.logger.Error().Err(err).Msg("failed to leave voice channel")
		} else {
			a.logger.Info().Msg("voice engine stopped")
		}
	}

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

// --- Server Management Bindings ---

// CreateServer creates a new server. Returns the created server.
func (a *App) CreateServer(name, ownerID string) (*server.Server, error) {
	return a.serverService.CreateServer(a.ctx, name, ownerID)
}

// GetServer retrieves a server by ID.
func (a *App) GetServer(serverID string) (*server.Server, error) {
	return a.serverService.GetServer(a.ctx, serverID)
}

// ListUserServers returns all servers the user belongs to.
func (a *App) ListUserServers(userID string) ([]*server.Server, error) {
	return a.serverService.ListUserServers(a.ctx, userID)
}

// UpdateServer updates a server's name and icon.
func (a *App) UpdateServer(serverID, userID, name, iconURL string) error {
	return a.serverService.UpdateServer(a.ctx, serverID, userID, name, iconURL)
}

// DeleteServer removes a server. Only the owner can delete.
func (a *App) DeleteServer(serverID, userID string) error {
	return a.serverService.DeleteServer(a.ctx, serverID, userID)
}

// CreateChannel creates a new channel within a server.
func (a *App) CreateChannel(serverID, userID, name, chType string) (*server.Channel, error) {
	return a.serverService.CreateChannel(a.ctx, serverID, userID, name, chType)
}

// ListChannels returns all channels for a server.
func (a *App) ListChannels(serverID string) ([]*server.Channel, error) {
	return a.serverService.ListChannels(a.ctx, serverID)
}

// DeleteChannel removes a channel from a server.
func (a *App) DeleteChannel(serverID, userID, channelID string) error {
	return a.serverService.DeleteChannel(a.ctx, serverID, userID, channelID)
}

// ListMembers returns all members of a server.
func (a *App) ListMembers(serverID string) ([]*server.Member, error) {
	return a.serverService.ListMembers(a.ctx, serverID)
}

// KickMember removes a member from a server.
func (a *App) KickMember(serverID, actorID, targetID string) error {
	return a.serverService.KickMember(a.ctx, serverID, actorID, targetID)
}

// UpdateMemberRole changes a member's role in a server.
func (a *App) UpdateMemberRole(serverID, actorID, targetID string, role string) error {
	return a.serverService.UpdateMemberRole(a.ctx, serverID, actorID, targetID, server.Role(role))
}

// GenerateInvite creates a new invite code for a server.
func (a *App) GenerateInvite(serverID, userID string) (string, error) {
	return a.serverService.GenerateInvite(a.ctx, serverID, userID)
}

// RedeemInvite joins a server using an invite code.
func (a *App) RedeemInvite(code, userID string) (*server.Server, error) {
	return a.serverService.RedeemInvite(a.ctx, code, userID)
}

// GetInviteInfo returns info about a server from an invite code.
func (a *App) GetInviteInfo(code string) (*server.InviteInfo, error) {
	return a.serverService.GetInviteInfo(a.ctx, code)
}

// --- Chat Bindings ---

// SendMessage sends a text message to a channel.
func (a *App) SendMessage(channelID, authorID, content string) (*chat.Message, error) {
	return a.chatService.SendMessage(a.ctx, channelID, authorID, content)
}

// GetMessages retrieves messages for a channel with cursor-based pagination.
func (a *App) GetMessages(channelID string, before string, after string, limit int) ([]*chat.Message, error) {
	return a.chatService.GetMessages(a.ctx, channelID, chat.PaginationOpts{
		Before: before,
		After:  after,
		Limit:  limit,
	})
}

// EditMessage updates the content of a message.
func (a *App) EditMessage(messageID, authorID, content string) (*chat.Message, error) {
	return a.chatService.EditMessage(a.ctx, messageID, authorID, content)
}

// DeleteMessage removes a message.
func (a *App) DeleteMessage(messageID, actorID string, isManager bool) error {
	return a.chatService.DeleteMessage(a.ctx, messageID, actorID, isManager)
}

// SearchMessages performs full-text search in a channel.
func (a *App) SearchMessages(channelID, query string, limit int) ([]*chat.SearchResult, error) {
	return a.chatService.SearchMessages(a.ctx, channelID, query, limit)
}

// --- Voice Bindings ---

// JoinVoice joins a voice channel.
func (a *App) JoinVoice(channelID string) error {
	return a.voiceEngine.JoinChannel(a.ctx, channelID)
}

// LeaveVoice leaves the current voice channel.
func (a *App) LeaveVoice() error {
	return a.voiceEngine.LeaveChannel()
}

// ToggleMute toggles the microphone mute state.
func (a *App) ToggleMute() bool {
	a.voiceEngine.Mute()
	return a.voiceEngine.IsMuted()
}

// ToggleDeafen toggles the audio output deafen state.
func (a *App) ToggleDeafen() bool {
	a.voiceEngine.Deafen()
	return a.voiceEngine.IsDeafened()
}

// GetVoiceStatus returns the current voice status.
func (a *App) GetVoiceStatus() voice.VoiceStatus {
	return a.voiceEngine.GetStatus()
}

// --- File Sharing Bindings ---

// UploadFile validates and stores a file attached to a message.
func (a *App) UploadFile(messageID, filename string, data []byte) (*files.Attachment, error) {
	return a.fileService.Upload(a.ctx, messageID, filename, data)
}

// DownloadFile retrieves file data for an attachment.
func (a *App) DownloadFile(attachmentID string) ([]byte, error) {
	data, _, err := a.fileService.Download(a.ctx, attachmentID)
	return data, err
}

// GetAttachments returns all attachments for a message.
func (a *App) GetAttachments(messageID string) ([]*files.Attachment, error) {
	return a.fileService.GetAttachments(a.ctx, messageID)
}

// DeleteAttachment removes an attachment.
func (a *App) DeleteAttachment(attachmentID string) error {
	return a.fileService.DeleteAttachment(a.ctx, attachmentID)
}

// --- Translation Bindings ---

// EnableTranslation activates voice translation between two languages.
func (a *App) EnableTranslation(sourceLang, targetLang string) error {
	return a.translationService.Enable(sourceLang, targetLang)
}

// DisableTranslation deactivates voice translation and stops the pipeline.
func (a *App) DisableTranslation() error {
	return a.translationService.Disable()
}

// GetTranslationStatus returns the current translation service status.
func (a *App) GetTranslationStatus() translation.Status {
	return a.translationService.GetStatus()
}

// SelectAvatarFile abre diálogo de seleção de arquivo de imagem e retorna
// o conteúdo como data URL base64 para armazenamento local.
// Complexity: O(n) onde n é o tamanho do arquivo de imagem.
func (a *App) SelectAvatarFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Escolha seu avatar",
		Filters: []runtime.FileFilter{
			{DisplayName: "Imagens", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.webp"},
		},
	})
	if err != nil || path == "" {
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read avatar file: %w", err)
	}

	mimeType := http.DetectContentType(data)
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded), nil
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
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

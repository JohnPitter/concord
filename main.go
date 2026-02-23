package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"errors"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/cache"
	"github.com/concord-chat/concord/internal/chat"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/files"
	"github.com/concord-chat/concord/internal/friends"
	"github.com/concord-chat/concord/internal/network/p2p"
	"github.com/concord-chat/concord/internal/network/signaling"
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
	friendService      *friends.Service
	voiceEngine        *voice.Engine
	voiceOrch          *voice.Orchestrator
	voiceTranslator    *voice.VoiceTranslator
	sigServer          *signaling.Server
	sigListener        net.Listener
	fileService        *files.Service
	translationService *translation.Service
	p2pHost            *p2p.Host
	p2pRepo            *sqlite.P2PRepo
	p2pPeerNames       sync.Map // peerID(string) → p2p.ProfilePayload
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

	// Initialize friends service
	friendRepo := friends.NewRepository(a.db, friends.NewStdlibTransactor(a.db.Conn()), a.logger)
	a.friendService = friends.NewService(friendRepo, a.logger)
	a.logger.Info().Msg("friends service initialized")

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

	// Initialize local signaling server for voice WebRTC coordination
	a.sigServer = signaling.NewServer(a.logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/signaling", a.sigServer.Handler())

	sigListener, err := net.Listen("tcp", "127.0.0.1:0") // random free port
	if err != nil {
		a.logger.Fatal().Err(err).Msg("failed to start signaling listener")
	}
	a.sigListener = sigListener
	go func() {
		if err := http.Serve(sigListener, mux); err != nil && !errors.Is(err, net.ErrClosed) {
			a.logger.Error().Err(err).Msg("signaling HTTP server error")
		}
	}()
	sigAddr := sigListener.Addr().String()
	a.logger.Info().Str("addr", sigAddr).Msg("local signaling server started")

	// Initialize voice engine + orchestrator
	a.voiceEngine = voice.NewEngine(voice.DefaultEngineConfig(), a.logger)
	a.voiceOrch = voice.NewOrchestrator(a.voiceEngine, a.logger)
	a.logger.Info().Msg("voice engine initialized")

	// Initialize translation service
	a.translationService = translation.NewService(cfg.Translation, a.logger)
	a.logger.Info().
		Bool("enabled", cfg.Translation.Enabled).
		Str("default_lang", cfg.Translation.DefaultLang).
		Msg("translation service initialized")

	// Initialize voice translator (STT + TTS pipeline)
	vtCfg := cfg.Voice.VoiceTranslation
	sttClient := voice.NewSTTClient(voice.STTConfig{
		APIURL:  vtCfg.STTURL,
		APIKey:  vtCfg.STTAPIKey,
		Model:   vtCfg.STTModel,
		Timeout: vtCfg.Timeout,
	}, a.logger)
	ttsClient := voice.NewTTSClient(voice.TTSConfig{
		APIURL:  vtCfg.TTSURL,
		APIKey:  vtCfg.TTSAPIKey,
		Voice:   vtCfg.TTSVoice,
		Format:  vtCfg.TTSFormat,
		Timeout: vtCfg.Timeout,
	}, a.logger)
	a.voiceTranslator = voice.NewVoiceTranslator(
		sttClient,
		ttsClient,
		a.translationService,
		func(eventName string, data ...interface{}) {
			runtime.EventsEmit(a.ctx, eventName, data...)
		},
		vtCfg.SegmentLength,
		a.logger,
	)
	a.voiceEngine.SetTranslator(a.voiceTranslator)
	a.logger.Info().Msg("voice translator initialized")

	a.logger.Info().Msg("Concord started successfully")
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	a.logger.Info().Msg("shutting down Concord")

	// Disable voice translator if active
	if a.voiceTranslator != nil {
		_ = a.voiceTranslator.Disable()
		a.logger.Info().Msg("voice translator stopped")
	}

	// Disable translation service if active
	if a.translationService != nil {
		_ = a.translationService.Disable()
		a.logger.Info().Msg("translation service stopped")
	}

	// Leave voice channel if connected
	if a.voiceOrch != nil {
		if err := a.voiceOrch.Leave(); err != nil {
			a.logger.Error().Err(err).Msg("failed to leave voice channel")
		} else {
			a.logger.Info().Msg("voice engine stopped")
		}
	}

	// Stop local signaling server
	if a.sigListener != nil {
		_ = a.sigListener.Close()
		a.logger.Info().Msg("signaling server stopped")
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

// GetPublicURL returns the configured public URL (Cloudflare tunnel or local).
func (a *App) GetPublicURL() string {
	return a.cfg.GetPublicURL()
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

// --- Friend Bindings ---

// SendFriendRequest sends a friend request to a user by username.
func (a *App) SendFriendRequest(senderID, username string) error {
	return a.friendService.SendRequest(a.ctx, senderID, username)
}

// GetPendingRequests returns all pending friend requests for a user.
func (a *App) GetPendingRequests(userID string) ([]friends.FriendRequestView, error) {
	return a.friendService.GetPendingRequests(a.ctx, userID)
}

// AcceptFriendRequest accepts a friend request.
func (a *App) AcceptFriendRequest(requestID, userID string) error {
	return a.friendService.AcceptRequest(a.ctx, requestID, userID)
}

// RejectFriendRequest rejects or cancels a friend request.
func (a *App) RejectFriendRequest(requestID, userID string) error {
	return a.friendService.RejectRequest(a.ctx, requestID, userID)
}

// GetFriends returns all friends for a user.
func (a *App) GetFriends(userID string) ([]friends.FriendView, error) {
	return a.friendService.GetFriends(a.ctx, userID)
}

// RemoveFriend removes a friendship.
func (a *App) RemoveFriend(userID, friendID string) error {
	return a.friendService.RemoveFriend(a.ctx, userID, friendID)
}

// BlockUser blocks a target user.
func (a *App) BlockUser(userID, targetID string) error {
	return a.friendService.BlockUser(a.ctx, userID, targetID)
}

// UnblockUser unblocks a user by username.
func (a *App) UnblockUser(userID, username string) error {
	return a.friendService.UnblockUser(a.ctx, userID, username)
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

// JoinVoice joins a voice channel via signaling.
// serverID identifies which server the channel belongs to.
func (a *App) JoinVoice(serverID, channelID, userID, username, avatarURL string) error {
	// Use the local embedded signaling server
	wsURL := fmt.Sprintf("http://%s", a.sigListener.Addr().String())
	return a.voiceOrch.Join(a.ctx, wsURL, serverID, channelID, userID, username, avatarURL)
}

// LeaveVoice leaves the current voice channel.
func (a *App) LeaveVoice() error {
	return a.voiceOrch.Leave()
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

// GetVoiceParticipants returns the list of peers currently in a voice channel.
// This works for any user browsing the server, not just those connected.
func (a *App) GetVoiceParticipants(serverID, channelID string) []signaling.PeerEntry {
	if a.sigServer == nil {
		return []signaling.PeerEntry{}
	}
	peers := a.sigServer.GetChannelPeers(serverID, channelID)
	if peers == nil {
		return []signaling.PeerEntry{}
	}
	return peers
}

// --- Voice Translation Bindings ---

// EnableVoiceTranslation activates real-time voice translation.
func (a *App) EnableVoiceTranslation(sourceLang, targetLang string) error {
	return a.voiceTranslator.Enable(sourceLang, targetLang)
}

// DisableVoiceTranslation deactivates real-time voice translation.
func (a *App) DisableVoiceTranslation() error {
	return a.voiceTranslator.Disable()
}

// GetVoiceTranslationStatus returns the current voice translation status.
func (a *App) GetVoiceTranslationStatus() voice.VoiceTranslationStatus {
	return a.voiceTranslator.GetStatus()
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

// EnableTranslation activates text translation between two languages.
func (a *App) EnableTranslation(sourceLang, targetLang string) error {
	return a.translationService.Enable(sourceLang, targetLang)
}

// DisableTranslation deactivates text translation.
func (a *App) DisableTranslation() error {
	return a.translationService.Disable()
}

// GetTranslationStatus returns the current translation service status.
func (a *App) GetTranslationStatus() translation.Status {
	return a.translationService.GetStatus()
}

// TranslateText translates text using the configured translation service.
// Unlike EnableTranslation (which gates the service), this always works
// as long as the LibreTranslate backend is reachable.
func (a *App) TranslateText(text, sourceLang, targetLang string) (string, error) {
	return a.translationService.TranslateTextDirect(a.ctx, text, sourceLang, targetLang)
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

// --- P2P Bindings ---

// GetP2PRoomCode retorna o código de sala do peer local.
// Retorna string vazia se o host P2P não estiver inicializado.
func (a *App) GetP2PRoomCode() string {
	if a.p2pHost == nil {
		return ""
	}
	return a.p2pHost.RoomCode()
}

// JoinP2PRoom conecta ao rendezvous DHT de uma sala pelo código curto.
// Descobre peers via DHT e conecta automaticamente a cada um encontrado.
// Retorna erro se o host P2P não estiver inicializado ou DHT indisponível.
func (a *App) JoinP2PRoom(code string) error {
	if a.p2pHost == nil {
		return fmt.Errorf("p2p host not initialized")
	}
	rendezvous := p2p.RoomRendezvous(code)
	peerChan, err := a.p2pHost.FindPeers(a.ctx, rendezvous)
	if err != nil {
		return err
	}

	// Consume the peer channel in a goroutine and auto-connect
	go func() {
		for pi := range peerChan {
			if pi.ID == a.p2pHost.LibP2PHost().ID() {
				continue // skip self
			}
			a.logger.Info().Str("peer_id", pi.ID.String()).Str("room", code).Msg("room: peer discovered via DHT")
			connectCtx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
			if err := a.p2pHost.LibP2PHost().Connect(connectCtx, pi); err != nil {
				a.logger.Debug().Err(err).Str("peer_id", pi.ID.String()).Msg("room: auto-connect failed")
			} else {
				a.logger.Info().Str("peer_id", pi.ID.String()).Msg("room: peer connected")
			}
			cancel()
		}
	}()

	return nil
}

// GetP2PPeers retorna a lista de peers conectados (LAN e sala).
// Retorna slice vazio se o host P2P não estiver inicializado.
func (a *App) GetP2PPeers() []p2p.PeerInfo {
	if a.p2pHost == nil {
		return []p2p.PeerInfo{}
	}
	return a.p2pHost.Peers()
}

// --- P2P Full Mode Bindings ---

// P2PMessage é a estrutura de mensagem P2P exposta ao frontend.
type P2PMessage = sqlite.P2PMessage

// InitP2PHost inicializa o host libp2p para o modo P2P.
// Idempotente — seguro de chamar múltiplas vezes.
// Complexity: O(1).
func (a *App) InitP2PHost() error {
	if a.p2pHost != nil {
		return nil // já inicializado
	}

	host, err := p2p.New(p2p.DefaultConfig(), a.logger)
	if err != nil {
		return fmt.Errorf("init p2p host: %w", err)
	}
	a.p2pHost = host
	a.p2pRepo = sqlite.NewP2PRepo(a.db)

	// Registrar handler de mensagens recebidas
	host.OnMessage(func(peerID string, data []byte) {
		env, err := p2p.DecodeEnvelope(data)
		if err != nil {
			a.logger.Warn().Err(err).Str("peer", peerID).Msg("p2p: invalid envelope")
			return
		}

		switch env.Type {
		case p2p.TypeProfile:
			var prof p2p.ProfilePayload
			if err := json.Unmarshal(env.Payload, &prof); err == nil {
				a.p2pPeerNames.Store(peerID, prof)
				a.logger.Info().Str("peer", peerID).Str("name", prof.DisplayName).Msg("p2p: profile received")
			}

		case p2p.TypeChat:
			var chat p2p.ChatPayload
			if err := json.Unmarshal(env.Payload, &chat); err == nil {
				msg := sqlite.P2PMessage{
					ID:        fmt.Sprintf("%s-%s", peerID, chat.SentAt),
					PeerID:    peerID,
					Direction: "received",
					Content:   chat.Content,
					SentAt:    chat.SentAt,
				}
				if err := a.p2pRepo.SaveMessage(a.ctx, msg); err != nil {
					a.logger.Warn().Err(err).Msg("p2p: save received message")
				}
				runtime.EventsEmit(a.ctx, "p2p:message", msg)
			}
		}
	})

	a.logger.Info().Str("id", host.ID()).Msg("p2p: host initialized")
	return nil
}

// SendP2PProfile envia o perfil local para todos os peers conectados.
func (a *App) SendP2PProfile(displayName, avatarDataURL string) error {
	if a.p2pHost == nil {
		return nil
	}
	prof := p2p.ProfilePayload{DisplayName: displayName, AvatarDataURL: avatarDataURL}
	for _, peer := range a.p2pHost.Peers() {
		a.sendProfileHandshake(peer.ID, prof)
	}
	return nil
}

func (a *App) sendProfileHandshake(peerID string, prof p2p.ProfilePayload) {
	data, err := p2p.EncodeEnvelope(p2p.TypeProfile, a.p2pHost.ID(), prof)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	if err := a.p2pHost.SendData(ctx, peerID, data); err != nil {
		a.logger.Debug().Err(err).Str("peer", peerID).Msg("p2p: profile handshake failed")
	}
}

// SendP2PMessage envia uma mensagem de chat para um peer e persiste localmente.
// Complexity: O(1).
func (a *App) SendP2PMessage(peerID, content string) error {
	if a.p2pHost == nil {
		return fmt.Errorf("p2p host not initialized")
	}
	if a.p2pRepo == nil {
		return fmt.Errorf("p2p repository not initialized")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	msg := sqlite.P2PMessage{
		ID:        fmt.Sprintf("%s-%s", a.p2pHost.ID(), now),
		PeerID:    peerID,
		Direction: "sent",
		Content:   content,
		SentAt:    now,
	}

	if err := a.p2pRepo.SaveMessage(a.ctx, msg); err != nil {
		return fmt.Errorf("save sent message: %w", err)
	}

	payload := p2p.ChatPayload{Content: content, SentAt: now}
	data, err := p2p.EncodeEnvelope(p2p.TypeChat, a.p2pHost.ID(), payload)
	if err != nil {
		return fmt.Errorf("encode chat: %w", err)
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return a.p2pHost.SendData(ctx, peerID, data)
}

// GetP2PMessages retorna o histórico de mensagens com um peer.
// Complexity: O(n) onde n = limit.
func (a *App) GetP2PMessages(peerID string, limit int) ([]sqlite.P2PMessage, error) {
	if a.p2pRepo == nil {
		return []sqlite.P2PMessage{}, nil
	}
	return a.p2pRepo.GetMessages(a.ctx, peerID, limit)
}

// GetP2PPeerName retorna o nome do perfil recebido de um peer.
func (a *App) GetP2PPeerName(peerID string) string {
	if v, ok := a.p2pPeerNames.Load(peerID); ok {
		if prof, ok := v.(p2p.ProfilePayload); ok {
			return prof.DisplayName
		}
	}
	if len(peerID) >= 8 {
		return peerID[:8]
	}
	return peerID
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Concord",
		Width:            1400,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
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

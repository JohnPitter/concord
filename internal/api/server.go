package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/chat"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/friends"
	"github.com/concord-chat/concord/internal/network/signaling"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/server"
)

// Server is the central HTTP API server for Concord.
// It wires chi routing, middleware, and service handlers.
type Server struct {
	router     chi.Router
	httpServer *http.Server
	auth       *auth.Service
	servers    *server.Service
	chat       *chat.Service
	friends    *friends.Service
	signaling  *signaling.Server
	health     *observability.HealthChecker
	metrics    *observability.Metrics
	logger     zerolog.Logger
	cfg        config.ServerConfig
}

// New creates and configures a new API Server with all routes and middleware.
// The jwtManager may be nil if only public routes are needed (e.g. health/metrics).
// Complexity: O(1)
func New(
	cfg config.ServerConfig,
	authSvc *auth.Service,
	serverSvc *server.Service,
	chatSvc *chat.Service,
	friendsSvc *friends.Service,
	sigServer *signaling.Server,
	jwtManager *auth.JWTManager,
	health *observability.HealthChecker,
	metrics *observability.Metrics,
	logger zerolog.Logger,
) *Server {
	s := &Server{
		auth:      authSvc,
		servers:   serverSvc,
		chat:      chatSvc,
		friends:   friendsSvc,
		signaling: sigServer,
		health:    health,
		metrics:   metrics,
		logger:    logger.With().Str("component", "api_server").Logger(),
		cfg:       cfg,
	}

	// Root router: keeps WebSocket signaling outside the API middleware stack.
	r := chi.NewRouter()

	// --- WebSocket signaling endpoint ---
	// Registered on root router so it bypasses API timeout/body-limit/rate-limit middleware.
	if sigServer != nil {
		// Support both with and without trailing slash.
		r.Get("/ws/signaling", sigServer.Handler())
		r.Get("/ws/signaling/", sigServer.Handler())
	}

	// API router with full middleware stack.
	apiRouter := chi.NewRouter()

	// --- Global middleware stack ---
	apiRouter.Use(middleware.RequestID)
	apiRouter.Use(middleware.RealIP)
	apiRouter.Use(RequestLogger(s.logger))
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(middleware.Timeout(30 * time.Second))
	apiRouter.Use(SecurityHeaders())
	apiRouter.Use(CORSMiddleware(cfg.CORS))
	apiRouter.Use(MaxBodySize(1 << 20)) // 1 MB default body limit

	// Rate limiting with standard headers (config-driven RPS, default 100/s)
	rps := cfg.RateLimitRPS
	if rps <= 0 {
		rps = 100
	}
	apiRouter.Use(RateLimitWithHeaders(rps))

	// Prometheus HTTP metrics
	if metrics != nil {
		apiRouter.Use(MetricsMiddleware(metrics))
	}

	// --- Public endpoints ---
	apiRouter.Get("/health", s.handleHealth)
	apiRouter.Get("/health/live", s.handleLiveness)
	apiRouter.Get("/health/ready", s.handleReadiness)
	apiRouter.Handle("/metrics", promhttp.Handler())

	// --- API v1 ---
	apiRouter.Route("/api/v1", func(api chi.Router) {
		// Auth routes (public)
		api.Route("/auth", func(ar chi.Router) {
			ar.Post("/device-code", s.handleDeviceCode)
			ar.Post("/token", s.handleToken)
			ar.Post("/refresh", s.handleRefresh)
		})

		// Protected routes — require valid JWT
		api.Group(func(protected chi.Router) {
			if jwtManager != nil {
				protected.Use(AuthMiddleware(jwtManager))
			}

			// Servers
			protected.Get("/servers", s.handleListServers)
			protected.Post("/servers", s.handleCreateServer)
			protected.Get("/servers/{serverID}", s.handleGetServer)
			protected.Put("/servers/{serverID}", s.handleUpdateServer)
			protected.Delete("/servers/{serverID}", s.handleDeleteServer)

			// Channels (nested under servers)
			protected.Get("/servers/{serverID}/channels", s.handleListChannels)
			protected.Post("/servers/{serverID}/channels", s.handleCreateChannel)

			// Members (nested under servers)
			protected.Get("/servers/{serverID}/members", s.handleListMembers)
			protected.Delete("/servers/{serverID}/members/{userID}", s.handleKickMember)
			protected.Put("/servers/{serverID}/members/{userID}/role", s.handleUpdateMemberRole)

			// Invites
			protected.Post("/servers/{serverID}/invite", s.handleGenerateInvite)
			protected.Post("/invite/{code}/redeem", s.handleRedeemInvite)

			// Messages
			protected.Get("/channels/{channelID}/messages", s.handleGetMessages)
			protected.Post("/channels/{channelID}/messages", s.handleSendMessage)
			protected.Put("/messages/{messageID}", s.handleEditMessage)
			protected.Delete("/messages/{messageID}", s.handleDeleteMessage)
			protected.Get("/channels/{channelID}/messages/search", s.handleSearchMessages)

			// Voice
			protected.Get("/servers/{serverID}/channels/{channelID}/voice/participants", s.handleVoiceParticipants)
			protected.Get("/servers/{serverID}/voice/participants", s.handleServerVoiceParticipants)

			// Friends
			protected.Post("/friends/request", s.handleSendFriendRequest)
			protected.Get("/friends/requests", s.handleGetPendingRequests)
			protected.Put("/friends/requests/{requestID}/accept", s.handleAcceptFriendRequest)
			protected.Delete("/friends/requests/{requestID}", s.handleRejectFriendRequest)
			protected.Get("/friends", s.handleGetFriends)
			protected.Delete("/friends/{friendID}", s.handleRemoveFriend)
			protected.Post("/friends/{friendID}/block", s.handleBlockUser)
			protected.Delete("/friends/{friendID}/block", s.handleUnblockUser)
		})
	})

	r.Mount("/", apiRouter)

	s.router = r
	return s
}

// Start begins listening for HTTP connections.
// It blocks until the server is shut down or an error occurs.
// Complexity: O(1) startup
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	s.logger.Info().
		Str("addr", addr).
		Bool("tls", s.cfg.TLSEnabled).
		Msg("starting HTTP server")

	if s.cfg.TLSEnabled && s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != "" {
		return s.httpServer.ListenAndServeTLS(s.cfg.TLSCertFile, s.cfg.TLSKeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
// Complexity: O(1)
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the chi router as an http.Handler for testing.
func (s *Server) Handler() http.Handler {
	return s.router
}

// handleHealth returns the aggregated health status from all registered checks.
// GET /health
// Complexity: O(n) where n is the number of registered health checks
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
		return
	}

	result := s.health.Check(r.Context())

	status := http.StatusOK
	if result.IsUnhealthy() {
		status = http.StatusServiceUnavailable
	} else if result.IsDegraded() {
		status = http.StatusOK // degraded but still serving
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(result)
}

// handleLiveness reports whether the process is alive.
// GET /health/live
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

// handleReadiness reports whether the service is ready to receive traffic.
// GET /health/ready
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if s.health == nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ready",
		})
		return
	}

	result := s.health.Check(r.Context())
	status := http.StatusOK
	if result.IsUnhealthy() {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]string{
		"status": string(result.Status),
	})
}

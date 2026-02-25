package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/concord-chat/concord/internal/presence"
)

func TestHandleSetPresenceOffline_Authenticated(t *testing.T) {
	jwt, err := auth.NewJWTManager("test-secret-that-is-at-least-32-characters-long")
	require.NoError(t, err)

	pair, err := jwt.GenerateTokenPair("user123", 42, "alice")
	require.NoError(t, err)

	tracker := presence.NewTracker(30 * time.Second)
	defer tracker.Stop()
	tracker.Touch("user123")
	require.True(t, tracker.IsOnline("user123"))

	logger := zerolog.Nop()
	health := observability.NewHealthChecker(logger, "test")
	cfg := config.ServerConfig{
		Host:         "127.0.0.1",
		Port:         0,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		CORS: config.CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"http://localhost:5173"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Authorization", "Content-Type"},
		},
	}

	s := New(cfg, nil, nil, nil, nil, nil, jwt, tracker, health, nil, logger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/presence/offline", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.False(t, tracker.IsOnline("user123"))
}

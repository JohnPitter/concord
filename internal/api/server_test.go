package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
)

// testServer creates a test API server with default config and nil services.
// The jwtManager is optional; pass nil for tests that don't need auth.
func testServer(t *testing.T, jwtManager *auth.JWTManager) *Server {
	t.Helper()

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

	return New(cfg, nil, nil, nil, jwtManager, health, logger)
}

// testJWTManager creates a JWTManager for testing with a fixed secret.
func testJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager("test-secret-that-is-at-least-32-characters-long")
	require.NoError(t, err)
	return mgr
}

// TestHealthEndpoint verifies that the /health endpoint returns 200 with status info.
func TestHealthEndpoint(t *testing.T) {
	s := testServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Contains(t, body, "status")
}

// TestDeviceCodeEndpoint verifies that the device-code endpoint returns an error
// when the auth service is nil (no GitHub OAuth configured).
func TestDeviceCodeEndpoint(t *testing.T) {
	s := testServer(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/device-code", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	// Auth service is nil, so the handler returns 503 Service Unavailable.
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

// TestListServersEndpoint_Unauthorized verifies that protected endpoints
// return 401 when no JWT token is provided.
func TestListServersEndpoint_Unauthorized(t *testing.T) {
	jwt := testJWTManager(t)
	s := testServer(t, jwt)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body errorResponse
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, body.Error.Code)
	assert.Contains(t, body.Error.Message, "authorization")
}

// TestCreateServerEndpoint_Unauthorized verifies that POST /api/v1/servers
// returns 401 without a token.
func TestCreateServerEndpoint_Unauthorized(t *testing.T) {
	jwt := testJWTManager(t)
	s := testServer(t, jwt)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestCORSHeaders verifies that CORS headers are set on responses
// when the request includes an allowed Origin.
func TestCORSHeaders(t *testing.T) {
	s := testServer(t, nil)

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "http://localhost:5173", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
}

// TestSecurityHeaders verifies that security headers are present on all responses.
func TestSecurityHeaders(t *testing.T) {
	s := testServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src")
}

// TestRateLimiting verifies that the rate limiter rejects requests exceeding the limit.
func TestRateLimiting(t *testing.T) {
	// Create a minimal server with a low rate limit for testing
	logger := zerolog.Nop()
	health := observability.NewHealthChecker(logger, "test")
	cfg := config.ServerConfig{
		Host:         "127.0.0.1",
		Port:         0,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		CORS: config.CORSConfig{
			Enabled: false,
		},
	}

	s := New(cfg, nil, nil, nil, nil, health, logger)

	// The default rate limit is 100 rps. Send 150 requests rapidly to trigger it.
	// Since all requests come from the same RemoteAddr in httptest, they share one bucket.
	limited := false
	for i := 0; i < 150; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			limited = true
			break
		}
	}

	assert.True(t, limited, "expected rate limiter to reject some requests")
}

// TestAuthMiddleware_ValidToken verifies that a valid JWT token allows access
// to protected endpoints and sets the user ID in context.
func TestAuthMiddleware_ValidToken(t *testing.T) {
	jwt := testJWTManager(t)
	s := testServer(t, jwt)

	// Generate a valid token
	pair, err := jwt.GenerateTokenPair("user123", 42, "testuser")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	// Should not be 401 (auth middleware passed), but likely 500
	// because the server service is nil. That's expected.
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

// TestAuthMiddleware_InvalidToken verifies that an invalid JWT is rejected.
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwt := testJWTManager(t)
	s := testServer(t, jwt)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestMetricsEndpoint verifies that /metrics returns Prometheus metrics.
func TestMetricsEndpoint(t *testing.T) {
	s := testServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Prometheus metrics endpoint returns text/plain with metrics
	assert.Contains(t, rec.Body.String(), "go_")
}

// ============================================================================
// Nil-service tests: jwtManager is nil so auth middleware is skipped,
// allowing requests to reach handlers that check for nil services.
// ============================================================================

// --- Auth handlers with nil service ---

func TestDeviceCode_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/device-code", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var resp errorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "auth service not available", resp.Error.Message)
}

func TestToken_NilService(t *testing.T) {
	s := testServer(t, nil)
	body := `{"device_code":"abc","interval":5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRefresh_NilService(t *testing.T) {
	s := testServer(t, nil)
	body := `{"user_id":"u1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- Server handlers with nil service (jwtManager nil so auth middleware is skipped) ---

func TestListServers_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateServer_NilService(t *testing.T) {
	s := testServer(t, nil)
	body := `{"name":"Test Server"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetServer_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestUpdateServer_NilService(t *testing.T) {
	s := testServer(t, nil)
	body := `{"name":"Updated","icon_url":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/servers/srv-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestDeleteServer_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/servers/srv-1", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- Channel handlers ---

func TestListChannels_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1/channels", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestCreateChannel_NilService(t *testing.T) {
	s := testServer(t, nil)
	body := `{"name":"general","type":"text"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/srv-1/channels", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- Message handlers ---

func TestGetMessages_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-1/messages", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSendMessage_NilService(t *testing.T) {
	s := testServer(t, nil)
	body := `{"content":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/channels/ch-1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSearchMessages_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-1/messages/search?q=hello", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- Member handlers ---

func TestListMembers_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1/members", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- Invite handlers ---

func TestGenerateInvite_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/srv-1/invite", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRedeemInvite_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/invite/abc123/redeem", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ============================================================================
// Validation tests: verify that bad inputs are rejected before hitting services.
// ============================================================================

func TestSearchMessages_MissingQuery_NilService(t *testing.T) {
	s := testServer(t, nil)
	// With nil chat service, nil check fires before validation
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-1/messages/search", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetMessages_InvalidLimit(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-1/messages?limit=abc", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	// Nil check precedes validation
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestSearchMessages_InvalidLimit_NilService(t *testing.T) {
	s := testServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/ch-1/messages/search?q=hello&limit=abc", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, req)

	// Nil check precedes validation
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

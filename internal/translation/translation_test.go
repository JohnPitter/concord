package translation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/concord-chat/concord/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger returns a no-op zerolog logger for tests.
func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

// testConfig returns a TranslationConfig suitable for testing.
func testConfig(url string) config.TranslationConfig {
	return config.TranslationConfig{
		Enabled:          true,
		PersonaPlexURL:   url,
		APIKey:           "test-api-key",
		DefaultLang:      "en",
		CacheEnabled:     true,
		CacheSize:        100,
		Timeout:          5 * time.Second,
		MaxLatency:       500 * time.Millisecond,
		CircuitBreaker:   true,
		FailureThreshold: 3,
	}
}

// newTestServer creates an httptest server that responds with translated text.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestTranslateText_Success(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		var req translateRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "Hello", req.Text)
		assert.Equal(t, "en", req.SourceLang)
		assert.Equal(t, "pt", req.TargetLang)

		resp := translateResponse{
			TranslatedText: "Olá",
			SourceLang:     "en",
			TargetLang:     "pt",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	client := NewClient(cfg, testLogger())

	result, err := client.TranslateText(context.Background(), "Hello", "en", "pt")
	require.NoError(t, err)
	assert.Equal(t, "Olá", result)
}

func TestTranslateText_APIError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	client := NewClient(cfg, testLogger())

	_, err := client.TranslateText(context.Background(), "Hello", "en", "pt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestCircuitBreaker_Activation(t *testing.T) {
	callCount := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unavailable"))
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.FailureThreshold = 3
	client := NewClient(cfg, testLogger())

	// Make FailureThreshold failing requests
	for i := 0; i < cfg.FailureThreshold; i++ {
		_, err := client.TranslateText(context.Background(), "test", "en", "pt")
		require.Error(t, err)
	}

	// Circuit should now be open
	assert.True(t, client.IsCircuitOpen())
	assert.Equal(t, cfg.FailureThreshold, client.ConsecutiveFailures())

	// Subsequent requests should fail immediately without hitting the server
	prevCount := callCount
	_, err := client.TranslateText(context.Background(), "test", "en", "pt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circuit breaker open")
	assert.Equal(t, prevCount, callCount, "should not have called the server")
}

func TestCircuitBreaker_Reset(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.FailureThreshold = 2
	client := NewClient(cfg, testLogger())

	// Trip the circuit
	for i := 0; i < cfg.FailureThreshold; i++ {
		client.TranslateText(context.Background(), "test", "en", "pt")
	}
	assert.True(t, client.IsCircuitOpen())

	// Reset circuit
	client.ResetCircuit()
	assert.False(t, client.IsCircuitOpen())
	assert.Equal(t, 0, client.ConsecutiveFailures())
}

func TestCircuitBreaker_LatencyTrip(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		resp := translateResponse{TranslatedText: "ok"}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.MaxLatency = 10 * time.Millisecond // Very low threshold
	cfg.FailureThreshold = 2
	client := NewClient(cfg, testLogger())

	// Each request should take >10ms (our threshold), incrementing failures
	for i := 0; i < cfg.FailureThreshold; i++ {
		client.TranslateText(context.Background(), "test", "en", "pt")
	}

	assert.True(t, client.IsCircuitOpen(), "circuit should open after consecutive latency violations")
}

func TestCache_HitAndMiss(t *testing.T) {
	tc := NewTranslationCache(100)

	// Miss
	_, ok := tc.Get("en", "pt", "Hello")
	assert.False(t, ok)

	// Set
	tc.Set("en", "pt", "Hello", "Olá")

	// Hit
	result, ok := tc.Get("en", "pt", "Hello")
	assert.True(t, ok)
	assert.Equal(t, "Olá", result)
}

func TestCache_DifferentLanguagePairs(t *testing.T) {
	tc := NewTranslationCache(100)

	tc.Set("en", "pt", "Hello", "Olá")
	tc.Set("en", "es", "Hello", "Hola")

	// Different language pair should be a different cache entry
	resultPT, ok := tc.Get("en", "pt", "Hello")
	require.True(t, ok)
	assert.Equal(t, "Olá", resultPT)

	resultES, ok := tc.Get("en", "es", "Hello")
	require.True(t, ok)
	assert.Equal(t, "Hola", resultES)
}

func TestCache_Eviction(t *testing.T) {
	tc := NewTranslationCache(2)

	tc.Set("en", "pt", "one", "um")
	tc.Set("en", "pt", "two", "dois")

	// Access "one" to make it recently used
	tc.Get("en", "pt", "one")

	// Adding a third should evict "two" (LRU)
	tc.Set("en", "pt", "three", "três")

	_, ok := tc.Get("en", "pt", "two")
	assert.False(t, ok, "should have been evicted")

	result, ok := tc.Get("en", "pt", "one")
	assert.True(t, ok)
	assert.Equal(t, "um", result)

	result, ok = tc.Get("en", "pt", "three")
	assert.True(t, ok)
	assert.Equal(t, "três", result)
}

func TestPipeline_StartAndStop(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// This handler won't be called for streaming (WebSocket),
		// but we set it up anyway. The stream will fail gracefully.
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.CircuitBreaker = false // Disable to avoid test complexity
	client := NewClient(cfg, testLogger())
	pipeline := NewPipeline(client, cfg, testLogger())

	audioIn := make(chan []byte, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start pipeline — WebSocket will fail, should gracefully degrade to pass-through
	out, err := pipeline.Start(ctx, audioIn, "en", "pt")
	require.NoError(t, err, "start should succeed even when translation stream fails (graceful degradation)")
	assert.NotNil(t, out)
	assert.True(t, pipeline.IsActive())

	// Send a frame through — should pass through since translation is in degraded mode
	testFrame := []byte("audio-frame-data")
	audioIn <- testFrame

	select {
	case received := <-out:
		assert.Equal(t, testFrame, received, "should receive original frame in pass-through mode")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for pass-through frame")
	}

	// Stop pipeline
	pipeline.Stop()
	assert.False(t, pipeline.IsActive())
}

func TestPipeline_GracefulDegradation(t *testing.T) {
	// Use an invalid URL to ensure connection fails immediately
	cfg := testConfig("http://invalid-host-that-does-not-exist:1")
	cfg.Timeout = 100 * time.Millisecond
	cfg.CircuitBreaker = false

	client := NewClient(cfg, testLogger())
	pipeline := NewPipeline(client, cfg, testLogger())

	audioIn := make(chan []byte, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should NOT return an error — graceful degradation kicks in
	out, err := pipeline.Start(ctx, audioIn, "en", "pt")
	require.NoError(t, err, "pipeline should start in pass-through mode")
	require.NotNil(t, out)

	// Send frames — they should pass through unchanged
	for i := 0; i < 3; i++ {
		frame := []byte{byte(i), byte(i + 1), byte(i + 2)}
		audioIn <- frame

		select {
		case received := <-out:
			assert.Equal(t, frame, received)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout on frame %d", i)
		}
	}

	pipeline.Stop()
}

func TestService_EnableDisable(t *testing.T) {
	cfg := testConfig("http://localhost:9999")
	svc := NewService(cfg, testLogger())

	// Initially disabled
	status := svc.GetStatus()
	assert.False(t, status.Enabled)

	// Enable
	err := svc.Enable("en", "pt")
	require.NoError(t, err)

	status = svc.GetStatus()
	assert.True(t, status.Enabled)
	assert.Equal(t, "en", status.SourceLang)
	assert.Equal(t, "pt", status.TargetLang)

	// Double enable should error
	err = svc.Enable("en", "es")
	require.Error(t, err)
	assert.Equal(t, ErrAlreadyEnabled, err)

	// Disable
	err = svc.Disable()
	require.NoError(t, err)

	status = svc.GetStatus()
	assert.False(t, status.Enabled)

	// Double disable should error
	err = svc.Disable()
	require.Error(t, err)
	assert.Equal(t, ErrNotEnabled, err)
}

func TestService_TranslateTextWithCache(t *testing.T) {
	callCount := 0
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := translateResponse{
			TranslatedText: "Olá",
			SourceLang:     "en",
			TargetLang:     "pt",
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	svc := NewService(cfg, testLogger())

	// Must enable first
	err := svc.Enable("en", "pt")
	require.NoError(t, err)

	// First call — cache miss, hits API
	result, err := svc.TranslateText(context.Background(), "Hello", "en", "pt")
	require.NoError(t, err)
	assert.Equal(t, "Olá", result)
	assert.Equal(t, 1, callCount)

	// Second call — cache hit, should NOT hit API
	result, err = svc.TranslateText(context.Background(), "Hello", "en", "pt")
	require.NoError(t, err)
	assert.Equal(t, "Olá", result)
	assert.Equal(t, 1, callCount, "should not have called API again — cache hit")
}

func TestService_TranslateText_Disabled(t *testing.T) {
	cfg := testConfig("http://localhost:9999")
	svc := NewService(cfg, testLogger())

	_, err := svc.TranslateText(context.Background(), "Hello", "en", "pt")
	require.Error(t, err)
	assert.Equal(t, ErrTranslationDisabled, err)
}

func TestClient_ConcurrentSafety(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := translateResponse{TranslatedText: "ok"}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	cfg := testConfig(srv.URL)
	client := NewClient(cfg, testLogger())

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.TranslateText(context.Background(), "test", "en", "pt")
		}()
	}
	wg.Wait()
	// No race condition = pass
}

func TestBuildCacheKey_Deterministic(t *testing.T) {
	key1 := buildCacheKey("en", "pt", "Hello")
	key2 := buildCacheKey("en", "pt", "Hello")
	key3 := buildCacheKey("en", "es", "Hello")
	key4 := buildCacheKey("en", "pt", "World")

	assert.Equal(t, key1, key2, "same inputs should produce same key")
	assert.NotEqual(t, key1, key3, "different target lang should produce different key")
	assert.NotEqual(t, key1, key4, "different text should produce different key")
	assert.Contains(t, key1, "translate:en:pt:")
}

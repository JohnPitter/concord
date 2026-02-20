package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthChecker(t *testing.T) {
	logger := NewNopLogger()
	checker := NewHealthChecker(logger, "1.0.0")

	assert.NotNil(t, checker)
	assert.Equal(t, "1.0.0", checker.version)
	assert.NotNil(t, checker.checks)
	assert.NotNil(t, checker.cache)
}

func TestHealthChecker_RegisterCheck(t *testing.T) {
	logger := NewNopLogger()
	checker := NewHealthChecker(logger, "1.0.0")

	check := func(ctx context.Context) error {
		return nil
	}

	checker.RegisterCheck("test_component", check)

	// Verify check was registered
	checker.mu.RLock()
	_, exists := checker.checks["test_component"]
	checker.mu.RUnlock()

	assert.True(t, exists)
}

func TestHealthChecker_UnregisterCheck(t *testing.T) {
	logger := NewNopLogger()
	checker := NewHealthChecker(logger, "1.0.0")

	check := func(ctx context.Context) error {
		return nil
	}

	checker.RegisterCheck("test_component", check)
	checker.UnregisterCheck("test_component")

	// Verify check was unregistered
	checker.mu.RLock()
	_, exists := checker.checks["test_component"]
	checker.mu.RUnlock()

	assert.False(t, exists)
}

func TestHealthChecker_Check(t *testing.T) {
	logger := NewNopLogger()
	checker := NewHealthChecker(logger, "1.0.0")

	t.Run("all healthy", func(t *testing.T) {
		checker.RegisterCheck("component1", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("component2", func(ctx context.Context) error {
			return nil
		})

		ctx := context.Background()
		health := checker.Check(ctx)

		assert.Equal(t, HealthStatusHealthy, health.Status)
		assert.Len(t, health.Components, 2)
		assert.Equal(t, "1.0.0", health.Version)
		assert.True(t, health.IsHealthy())
		assert.False(t, health.IsUnhealthy())
		assert.False(t, health.IsDegraded())
	})

	t.Run("one unhealthy", func(t *testing.T) {
		checker = NewHealthChecker(logger, "1.0.0")
		checker.RegisterCheck("healthy_component", func(ctx context.Context) error {
			return nil
		})
		checker.RegisterCheck("unhealthy_component", func(ctx context.Context) error {
			return errors.New("component is down")
		})

		ctx := context.Background()
		health := checker.Check(ctx)

		assert.Equal(t, HealthStatusUnhealthy, health.Status)
		assert.True(t, health.IsUnhealthy())
		unhealthy := health.GetUnhealthyComponents()
		assert.Contains(t, unhealthy, "unhealthy_component")
	})

	t.Run("no checks registered", func(t *testing.T) {
		checker = NewHealthChecker(logger, "1.0.0")

		ctx := context.Background()
		health := checker.Check(ctx)

		assert.Equal(t, HealthStatusUnknown, health.Status)
		assert.Empty(t, health.Components)
	})
}

func TestHealthChecker_CacheCheck(t *testing.T) {
	logger := NewNopLogger()
	checker := NewHealthChecker(logger, "1.0.0")
	checker.cacheTTL = 100 * time.Millisecond

	callCount := 0
	checker.RegisterCheck("cached_component", func(ctx context.Context) error {
		callCount++
		return nil
	})

	ctx := context.Background()

	// First call should execute the check
	health1 := checker.Check(ctx)
	assert.Equal(t, 1, callCount)
	assert.Equal(t, HealthStatusHealthy, health1.Status)

	// Second call within TTL should use cache
	health2 := checker.Check(ctx)
	assert.Equal(t, 1, callCount) // Still 1, cache was used
	assert.Equal(t, HealthStatusHealthy, health2.Status)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should execute the check again
	health3 := checker.Check(ctx)
	assert.Equal(t, 2, callCount)
	assert.Equal(t, HealthStatusHealthy, health3.Status)
}

func TestDatabaseHealthCheck(t *testing.T) {
	t.Run("healthy database", func(t *testing.T) {
		pingFunc := func(ctx context.Context) error {
			return nil
		}

		check := DatabaseHealthCheck(pingFunc)
		err := check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("unhealthy database", func(t *testing.T) {
		pingFunc := func(ctx context.Context) error {
			return errors.New("connection refused")
		}

		check := DatabaseHealthCheck(pingFunc)
		err := check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database ping failed")
	})
}

func TestRedisHealthCheck(t *testing.T) {
	t.Run("healthy redis", func(t *testing.T) {
		pingFunc := func(ctx context.Context) error {
			return nil
		}

		check := RedisHealthCheck(pingFunc)
		err := check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("unhealthy redis", func(t *testing.T) {
		pingFunc := func(ctx context.Context) error {
			return errors.New("connection timeout")
		}

		check := RedisHealthCheck(pingFunc)
		err := check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis ping failed")
	})
}

func TestP2PHostHealthCheck(t *testing.T) {
	t.Run("healthy p2p host", func(t *testing.T) {
		statusFunc := func() error {
			return nil
		}

		check := P2PHostHealthCheck(statusFunc)
		err := check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("unhealthy p2p host", func(t *testing.T) {
		statusFunc := func() error {
			return errors.New("no peers connected")
		}

		check := P2PHostHealthCheck(statusFunc)
		err := check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "p2p host unhealthy")
	})
}

func TestWebSocketHealthCheck(t *testing.T) {
	t.Run("connected websocket", func(t *testing.T) {
		isConnected := func() bool {
			return true
		}

		check := WebSocketHealthCheck(isConnected)
		err := check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("disconnected websocket", func(t *testing.T) {
		isConnected := func() bool {
			return false
		}

		check := WebSocketHealthCheck(isConnected)
		err := check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "websocket not connected")
	})
}

func TestVoiceEngineHealthCheck(t *testing.T) {
	t.Run("healthy voice engine", func(t *testing.T) {
		statusFunc := func() error {
			return nil
		}

		check := VoiceEngineHealthCheck(statusFunc)
		err := check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("unhealthy voice engine", func(t *testing.T) {
		statusFunc := func() error {
			return errors.New("audio device not available")
		}

		check := VoiceEngineHealthCheck(statusFunc)
		err := check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "voice engine unhealthy")
	})
}

func TestMemoryHealthCheck(t *testing.T) {
	t.Run("memory within limits", func(t *testing.T) {
		// Set a very high limit to ensure test passes
		check := MemoryHealthCheck(10 * 1024 * 1024 * 1024) // 10GB
		err := check(context.Background())
		assert.NoError(t, err)
	})

	t.Run("memory exceeds limits", func(t *testing.T) {
		// Set a very low limit to force failure
		check := MemoryHealthCheck(1) // 1 byte
		err := check(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "memory usage exceeded")
	})
}

func TestHealth_GetUnhealthyComponents(t *testing.T) {
	health := &Health{
		Status: HealthStatusUnhealthy,
		Components: map[string]ComponentHealth{
			"component1": {
				Name:   "component1",
				Status: HealthStatusHealthy,
			},
			"component2": {
				Name:   "component2",
				Status: HealthStatusUnhealthy,
			},
			"component3": {
				Name:   "component3",
				Status: HealthStatusUnhealthy,
			},
		},
	}

	unhealthy := health.GetUnhealthyComponents()
	assert.Len(t, unhealthy, 2)
	assert.Contains(t, unhealthy, "component2")
	assert.Contains(t, unhealthy, "component3")
}

func TestHealth_GetDegradedComponents(t *testing.T) {
	health := &Health{
		Status: HealthStatusDegraded,
		Components: map[string]ComponentHealth{
			"component1": {
				Name:   "component1",
				Status: HealthStatusHealthy,
			},
			"component2": {
				Name:   "component2",
				Status: HealthStatusDegraded,
			},
		},
	}

	degraded := health.GetDegradedComponents()
	assert.Len(t, degraded, 1)
	assert.Contains(t, degraded, "component2")
}

func TestGetMemoryStats(t *testing.T) {
	stats := getMemoryStats()
	assert.Greater(t, stats.Alloc, uint64(0))
	assert.Greater(t, stats.Sys, uint64(0))
}

package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 1*time.Second, 20)
	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.rate)
	assert.Equal(t, 1*time.Second, rl.interval)
	assert.Equal(t, 20, rl.capacity)
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		rl := NewRateLimiter(5, 1*time.Second, 5)

		// First 5 requests should be allowed
		for i := 0; i < 5; i++ {
			assert.True(t, rl.Allow("test-key"))
		}

		// 6th request should be denied (rate limit exceeded)
		assert.False(t, rl.Allow("test-key"))
	})

	t.Run("different keys have separate limits", func(t *testing.T) {
		rl := NewRateLimiter(2, 1*time.Second, 2)

		assert.True(t, rl.Allow("key1"))
		assert.True(t, rl.Allow("key2"))
		assert.True(t, rl.Allow("key1"))
		assert.True(t, rl.Allow("key2"))

		// Both should be rate limited now
		assert.False(t, rl.Allow("key1"))
		assert.False(t, rl.Allow("key2"))
	})
}

func TestRateLimiter_AllowN(t *testing.T) {
	t.Run("allows batch requests", func(t *testing.T) {
		rl := NewRateLimiter(10, 1*time.Second, 10)

		assert.True(t, rl.AllowN("test-key", 5))
		assert.True(t, rl.AllowN("test-key", 5))
		assert.False(t, rl.AllowN("test-key", 1))
	})

	t.Run("handles zero requests", func(t *testing.T) {
		rl := NewRateLimiter(1, 1*time.Second, 1)
		assert.True(t, rl.AllowN("test-key", 0))
	})
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(1, 1*time.Second, 1)

	// Use up the rate limit
	assert.True(t, rl.Allow("test-key"))
	assert.False(t, rl.Allow("test-key"))

	// Reset should allow new requests
	rl.Reset("test-key")
	assert.True(t, rl.Allow("test-key"))
}

func TestRateLimiter_WaitIfNeeded(t *testing.T) {
	t.Run("waits for available tokens", func(t *testing.T) {
		rl := NewRateLimiter(2, 100*time.Millisecond, 2)

		// Use up tokens
		rl.Allow("test-key")
		rl.Allow("test-key")

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := rl.WaitIfNeeded(ctx, "test-key")
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		rl := NewRateLimiter(1, 1*time.Hour, 1)

		// Use up token
		rl.Allow("test-key")

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := rl.WaitIfNeeded(ctx, "test-key")
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestNewBruteForceProtector(t *testing.T) {
	bfp := NewBruteForceProtector(5, 1*time.Minute)
	assert.NotNil(t, bfp)
	assert.Equal(t, 5, bfp.maxAttempts)
	assert.Equal(t, 1*time.Minute, bfp.lockoutPeriod)
}

func TestBruteForceProtector_RecordFailure(t *testing.T) {
	bfp := NewBruteForceProtector(3, 100*time.Millisecond)

	// First 2 failures should not trigger lockout
	for i := 0; i < 2; i++ {
		bfp.RecordFailure("user1")
		allowed, _, _ := bfp.IsAllowed("user1")
		assert.True(t, allowed, "should be allowed after %d failures", i+1)
	}

	// 3rd failure reaches maxAttempts and triggers lockout
	bfp.RecordFailure("user1")
	allowed, retryAfter, err := bfp.IsAllowed("user1")
	assert.False(t, allowed, "should be locked after 3rd failure")
	assert.Greater(t, retryAfter, time.Duration(0))
	assert.Error(t, err)
}

func TestBruteForceProtector_RecordSuccess(t *testing.T) {
	bfp := NewBruteForceProtector(3, 1*time.Minute)

	// Record failures
	bfp.RecordFailure("user1")
	bfp.RecordFailure("user1")
	assert.Equal(t, 2, bfp.GetAttempts("user1"))

	// Success should reset
	bfp.RecordSuccess("user1")
	assert.Equal(t, 0, bfp.GetAttempts("user1"))
}

func TestBruteForceProtector_IsAllowed(t *testing.T) {
	t.Run("allows before max attempts", func(t *testing.T) {
		bfp := NewBruteForceProtector(5, 1*time.Minute)

		allowed, retryAfter, err := bfp.IsAllowed("new-user")
		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), retryAfter)
		assert.NoError(t, err)
	})

	t.Run("unlocks after lockout period", func(t *testing.T) {
		bfp := NewBruteForceProtector(1, 50*time.Millisecond)

		// Trigger lockout
		bfp.RecordFailure("user1")
		bfp.RecordFailure("user1")

		allowed, _, _ := bfp.IsAllowed("user1")
		assert.False(t, allowed)

		// Wait for lockout to expire
		time.Sleep(100 * time.Millisecond)

		allowed, retryAfter, err := bfp.IsAllowed("user1")
		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), retryAfter)
		assert.NoError(t, err)
	})
}

func TestBruteForceProtector_GetAttempts(t *testing.T) {
	bfp := NewBruteForceProtector(10, 1*time.Minute)

	assert.Equal(t, 0, bfp.GetAttempts("new-user"))

	bfp.RecordFailure("user1")
	bfp.RecordFailure("user1")
	bfp.RecordFailure("user1")

	assert.Equal(t, 3, bfp.GetAttempts("user1"))
	assert.Equal(t, 0, bfp.GetAttempts("user2"))
}

package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestRedisConfig returns a RedisConfig suitable for integration tests.
func getTestRedisConfig() config.RedisConfig {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	password := os.Getenv("REDIS_PASSWORD")

	return config.RedisConfig{
		Enabled:      true,
		Host:         host,
		Port:         6379,
		Password:     password,
		DB:           15, // Use DB 15 for testing to avoid conflicts
		MaxRetries:   3,
		PoolSize:     5,
		MinIdleConns: 1,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// skipIfNoRedis skips the test if Redis is not available.
func skipIfNoRedis(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("REDIS_HOST") == "" {
		t.Skip("skipping integration test: REDIS_HOST not set")
	}
}

func TestIntegrationNew(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.NotNil(t, client.Underlying())
}

func TestIntegrationNew_InvalidConfig(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := config.RedisConfig{
		Enabled:      true,
		Host:         "nonexistent-host-that-should-fail.local",
		Port:         6379,
		Password:     "",
		DB:           0,
		MaxRetries:   0,
		PoolSize:     1,
		MinIdleConns: 0,
		DialTimeout:  1 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	_, err := New(cfg, logger)
	assert.Error(t, err)
}

func TestIntegrationPing(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	err = client.Ping(ctx)
	assert.NoError(t, err)
}

func TestIntegrationSetGet(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Set a value
	err = client.Set(ctx, "test:key1", "hello-world", 10*time.Second)
	require.NoError(t, err)

	// Get the value
	val, err := client.Get(ctx, "test:key1")
	require.NoError(t, err)
	assert.Equal(t, "hello-world", val)

	// Clean up
	_ = client.Delete(ctx, "test:key1")
}

func TestIntegrationGet_NonExistent(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	_, err = client.Get(ctx, "test:nonexistent-key-12345")
	assert.Error(t, err)
	assert.ErrorIs(t, err, goredis.Nil)
}

func TestIntegrationDelete(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Set some keys
	err = client.Set(ctx, "test:del1", "a", 10*time.Second)
	require.NoError(t, err)
	err = client.Set(ctx, "test:del2", "b", 10*time.Second)
	require.NoError(t, err)

	// Delete multiple keys
	err = client.Delete(ctx, "test:del1", "test:del2")
	require.NoError(t, err)

	// Verify they're gone
	_, err = client.Get(ctx, "test:del1")
	assert.ErrorIs(t, err, goredis.Nil)
	_, err = client.Get(ctx, "test:del2")
	assert.ErrorIs(t, err, goredis.Nil)
}

func TestIntegrationSetNX(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Ensure key doesn't exist
	_ = client.Delete(ctx, "test:setnx-key")

	// First SetNX should succeed
	ok, err := client.SetNX(ctx, "test:setnx-key", "locked", 10*time.Second)
	require.NoError(t, err)
	assert.True(t, ok)

	// Second SetNX should fail (key already exists)
	ok, err = client.SetNX(ctx, "test:setnx-key", "locked-again", 10*time.Second)
	require.NoError(t, err)
	assert.False(t, ok)

	// Value should be the first one
	val, err := client.Get(ctx, "test:setnx-key")
	require.NoError(t, err)
	assert.Equal(t, "locked", val)

	// Clean up
	_ = client.Delete(ctx, "test:setnx-key")
}

func TestIntegrationIncr(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Ensure key doesn't exist
	_ = client.Delete(ctx, "test:counter")

	// First incr (should create key and set to 1)
	val, err := client.Incr(ctx, "test:counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	// Second incr
	val, err = client.Incr(ctx, "test:counter")
	require.NoError(t, err)
	assert.Equal(t, int64(2), val)

	// Third incr
	val, err = client.Incr(ctx, "test:counter")
	require.NoError(t, err)
	assert.Equal(t, int64(3), val)

	// Clean up
	_ = client.Delete(ctx, "test:counter")
}

func TestIntegrationExpire(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Set a key without TTL (ttl=0 means no expiration)
	err = client.Set(ctx, "test:expire-key", "value", 0)
	require.NoError(t, err)

	// Set expire
	err = client.Expire(ctx, "test:expire-key", 1*time.Second)
	require.NoError(t, err)

	// Key should exist right now
	val, err := client.Get(ctx, "test:expire-key")
	require.NoError(t, err)
	assert.Equal(t, "value", val)

	// Wait for expiration
	time.Sleep(1500 * time.Millisecond)

	// Key should be gone
	_, err = client.Get(ctx, "test:expire-key")
	assert.ErrorIs(t, err, goredis.Nil)
}

func TestIntegrationPublishSubscribe(t *testing.T) {
	skipIfNoRedis(t)

	logger := observability.NewNopLogger()
	cfg := getTestRedisConfig()

	client, err := New(cfg, logger)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Subscribe
	pubsub := client.Subscribe(ctx, "test:channel")
	defer pubsub.Close()

	// Wait for subscription to be ready
	_, err = pubsub.Receive(ctx)
	require.NoError(t, err)

	// Publish a message
	err = client.Publish(ctx, "test:channel", "hello-pubsub")
	require.NoError(t, err)

	// Receive the message
	msg, err := pubsub.ReceiveMessage(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test:channel", msg.Channel)
	assert.Equal(t, "hello-pubsub", msg.Payload)
}

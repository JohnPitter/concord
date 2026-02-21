package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/concord-chat/concord/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Client wraps a go-redis client with logging and convenience methods
type Client struct {
	rdb    *redis.Client
	logger zerolog.Logger
}

// New creates a new Redis client, pings the server, and returns the Client wrapper.
// Complexity: O(1)
func New(cfg config.RedisConfig, logger zerolog.Logger) (*Client, error) {
	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Int("db", cfg.DB).
		Int("pool_size", cfg.PoolSize).
		Msg("initializing redis client")

	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Ping to verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info().Msg("redis client initialized successfully")

	return &Client{
		rdb:    rdb,
		logger: logger,
	}, nil
}

// Ping checks if the Redis server is reachable.
// Complexity: O(1)
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis connection and releases all resources.
func (c *Client) Close() error {
	c.logger.Info().Msg("closing redis client")
	return c.rdb.Close()
}

// Set stores a key-value pair with an optional TTL.
// If ttl is 0, the key will not expire.
// Complexity: O(1)
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}
	return nil
}

// Get retrieves the value for a key.
// Returns redis.Nil error if the key does not exist.
// Complexity: O(1)
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("failed to get key %s: %w", key, err)
	}
	return val, nil
}

// Delete removes one or more keys.
// Complexity: O(n) where n is the number of keys
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys: %w", err)
	}
	return nil
}

// Publish publishes a message to a Redis Pub/Sub channel.
// Complexity: O(n+m) where n is the number of clients subscribed to the channel
// and m is the total number of subscribed patterns
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	if err := c.rdb.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("failed to publish to channel %s: %w", channel, err)
	}
	return nil
}

// Subscribe subscribes to a Redis Pub/Sub channel and returns the subscription handle.
// The caller is responsible for closing the returned PubSub.
// Complexity: O(1)
func (c *Client) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channel)
}

// SetNX sets a key only if it does not already exist (SET if Not eXists).
// Returns true if the key was set, false if it already existed.
// Useful for distributed locking and rate limiting.
// Complexity: O(1)
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	ok, err := c.rdb.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx key %s: %w", key, err)
	}
	return ok, nil
}

// Incr atomically increments the integer value of a key by one.
// If the key does not exist, it is set to 0 before incrementing.
// Useful for counters and rate limiting.
// Complexity: O(1)
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to incr key %s: %w", key, err)
	}
	return val, nil
}

// Expire sets a timeout on a key. After the timeout, the key will be automatically deleted.
// Complexity: O(1)
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set expire on key %s: %w", key, err)
	}
	return nil
}

// Underlying returns the underlying *redis.Client for advanced operations.
func (c *Client) Underlying() *redis.Client {
	return c.rdb
}

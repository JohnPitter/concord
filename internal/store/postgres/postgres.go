package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/concord-chat/concord/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

// DB wraps a PostgreSQL connection pool with logging and configuration
type DB struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// New creates a new PostgreSQL connection pool, pings the database, and returns the DB wrapper.
// Complexity: O(1)
func New(cfg config.PostgresConfig, logger zerolog.Logger) (*DB, error) {
	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Str("user", cfg.User).
		Str("ssl_mode", cfg.SSLMode).
		Int("max_open_conns", cfg.MaxOpenConns).
		Int("max_idle_conns", cfg.MaxIdleConns).
		Msg("initializing postgresql database")

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	// Parse pool config
	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Apply pool settings
	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	if cfg.ConnMaxLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	}

	// Create pool with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Ping to verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	logger.Info().Msg("postgresql database initialized successfully")

	return &DB{
		pool:   pool,
		logger: logger,
	}, nil
}

// Ping checks if the database connection is alive.
// Complexity: O(1)
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close closes the connection pool and releases all resources.
func (db *DB) Close() {
	db.logger.Info().Msg("closing postgresql database")
	db.pool.Close()
}

// Pool returns the underlying *pgxpool.Pool for direct access.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// StdlibDB returns a *sql.DB that uses the underlying pgx pool via the stdlib bridge.
// This allows existing repositories that use database/sql interface to work with PostgreSQL.
func (db *DB) StdlibDB() *sql.DB {
	return stdlib.OpenDBFromPool(db.pool)
}

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestPostgresConfig returns a PostgresConfig suitable for integration tests.
// It reads connection details from environment variables with sensible defaults.
func getTestPostgresConfig() config.PostgresConfig {
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "concord"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	database := os.Getenv("POSTGRES_DB")
	if database == "" {
		database = "concord_test"
	}
	sslMode := os.Getenv("POSTGRES_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	return config.PostgresConfig{
		Host:            host,
		Port:            5432,
		Database:        database,
		User:            user,
		Password:        password,
		SSLMode:         sslMode,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	}
}

// skipIfNoPostgres skips the test if PostgreSQL is not available.
// Uses testing.Short() or checks for the POSTGRES_HOST env var.
func skipIfNoPostgres(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("skipping integration test: POSTGRES_HOST not set")
	}
}

func TestIntegrationNew(t *testing.T) {
	skipIfNoPostgres(t)

	logger := observability.NewNopLogger()
	cfg := getTestPostgresConfig()

	db, err := New(cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify pool is returned
	assert.NotNil(t, db.Pool())
}

func TestIntegrationNew_InvalidConfig(t *testing.T) {
	skipIfNoPostgres(t)

	logger := observability.NewNopLogger()
	cfg := config.PostgresConfig{
		Host:            "nonexistent-host-that-should-fail.local",
		Port:            5432,
		Database:        "nonexistent_db",
		User:            "nonexistent_user",
		Password:        "bad_password",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
	}

	_, err := New(cfg, logger)
	assert.Error(t, err)
}

func TestIntegrationPing(t *testing.T) {
	skipIfNoPostgres(t)

	logger := observability.NewNopLogger()
	cfg := getTestPostgresConfig()

	db, err := New(cfg, logger)
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.Ping(ctx)
	assert.NoError(t, err)
}

func TestIntegrationClose(t *testing.T) {
	skipIfNoPostgres(t)

	logger := observability.NewNopLogger()
	cfg := getTestPostgresConfig()

	db, err := New(cfg, logger)
	require.NoError(t, err)

	// Close should not panic
	db.Close()

	// After close, ping should fail
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = db.Ping(ctx)
	assert.Error(t, err)
}

func TestIntegrationMigrationRun(t *testing.T) {
	skipIfNoPostgres(t)

	logger := observability.NewNopLogger()
	cfg := getTestPostgresConfig()

	db, err := New(cfg, logger)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Clean up any previous test artifacts
	_, _ = db.pool.Exec(ctx, "DROP TABLE IF EXISTS server_invites, audit_log, auth_sessions, attachments, messages, server_members, channels, servers, users, schema_migrations CASCADE")

	migrator := NewMigrator(db, logger)

	// Run migrations
	err = migrator.Run(ctx)
	require.NoError(t, err)

	// Verify schema_migrations table was created and has records
	status, err := migrator.Status(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, status)
	assert.Equal(t, 1, status[0].Version)
	assert.Equal(t, "init", status[0].Name)

	// Run again should be idempotent (no-op)
	err = migrator.Run(ctx)
	require.NoError(t, err)

	// Clean up
	_, _ = db.pool.Exec(ctx, "DROP TABLE IF EXISTS server_invites, audit_log, auth_sessions, attachments, messages, server_members, channels, servers, users, schema_migrations CASCADE")
}

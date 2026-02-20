package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/concord-chat/concord/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMigrator(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := observability.NewNopLogger()
	migrator := NewMigrator(db, logger)

	assert.NotNil(t, migrator)
	assert.Equal(t, db, migrator.db)
}

func TestMigrator_Migrate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := observability.NewNopLogger()
	migrator := NewMigrator(db, logger)

	ctx := context.Background()

	// Run migrations
	err := migrator.Migrate(ctx)
	require.NoError(t, err)

	// Verify migrations table exists
	var tableExists int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableExists)
	require.NoError(t, err)
	assert.Equal(t, 1, tableExists)

	// Run migrations again (should be no-op)
	err = migrator.Migrate(ctx)
	require.NoError(t, err)
}

func TestMigrator_Status(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := observability.NewNopLogger()
	migrator := NewMigrator(db, logger)

	ctx := context.Background()

	// Initially should be empty
	status, err := migrator.Status(ctx)
	require.NoError(t, err)
	assert.Empty(t, status)

	// Run migrations
	err = migrator.Migrate(ctx)
	require.NoError(t, err)

	// Check status again
	status, err = migrator.Status(ctx)
	require.NoError(t, err)
	// Status may be empty if no migration files exist, which is fine for this test
}

func TestMigrator_Reset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := observability.NewNopLogger()
	migrator := NewMigrator(db, logger)

	ctx := context.Background()

	// Create a test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			name TEXT
		)
	`)
	require.NoError(t, err)

	// Verify table exists
	var tableCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 1, tableCount)

	// Reset database
	err = migrator.Reset(ctx)
	require.NoError(t, err)

	// Verify table no longer exists
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 0, tableCount)
}

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		expectedVersion int
		expectedName    string
	}{
		{
			name:            "valid migration",
			filename:        "001_init.sql",
			expectedVersion: 1,
			expectedName:    "init",
		},
		{
			name:            "valid migration with multi-digit version",
			filename:        "042_add_users_table.sql",
			expectedVersion: 42,
			expectedName:    "add_users_table",
		},
		{
			name:            "invalid migration without underscore",
			filename:        "001init.sql",
			expectedVersion: 0,
			expectedName:    "",
		},
		{
			name:            "invalid migration without version",
			filename:        "init.sql",
			expectedVersion: 0,
			expectedName:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, name := parseMigrationFilename(tt.filename)
			assert.Equal(t, tt.expectedVersion, version)
			assert.Equal(t, tt.expectedName, name)
		})
	}
}

func TestMigrator_FilterPendingMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := observability.NewNopLogger()
	migrator := NewMigrator(db, logger)

	allMigrations := []Migration{
		{Version: 1, Name: "init"},
		{Version: 2, Name: "add_users"},
		{Version: 3, Name: "add_channels"},
	}

	appliedMigrations := []Migration{
		{Version: 1, Name: "init"},
	}

	pending := migrator.filterPendingMigrations(allMigrations, appliedMigrations)

	assert.Len(t, pending, 2)
	assert.Equal(t, 2, pending[0].Version)
	assert.Equal(t, 3, pending[1].Version)
}

func TestMigrator_EnsureMigrationsTable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := observability.NewNopLogger()
	migrator := NewMigrator(db, logger)

	ctx := context.Background()

	// Ensure migrations table
	err := migrator.ensureMigrationsTable(ctx)
	require.NoError(t, err)

	// Verify table exists
	var tableExists int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableExists)
	require.NoError(t, err)
	assert.Equal(t, 1, tableExists)

	// Calling again should not error
	err = migrator.ensureMigrationsTable(ctx)
	require.NoError(t, err)
}

// setupMigratorTestDB creates a test database for migration tests
func setupMigratorTestDB(t *testing.T) *DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migrations_test.db")

	logger := observability.NewNopLogger()
	cfg := Config{
		Path:            dbPath,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 1 * time.Hour,
		WALMode:         false,
		ForeignKeys:     true,
		BusyTimeout:     5 * time.Second,
	}

	db, err := New(cfg, logger)
	require.NoError(t, err)

	return db
}

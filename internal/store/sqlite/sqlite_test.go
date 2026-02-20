package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/concord-chat/concord/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates database with default config", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		logger := observability.NewNopLogger()
		cfg := Config{
			Path:            dbPath,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Hour,
			WALMode:         true,
			ForeignKeys:     true,
			BusyTimeout:     5 * time.Second,
		}

		db, err := New(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify connection is alive
		ctx := context.Background()
		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("fails with invalid path", func(t *testing.T) {
		logger := observability.NewNopLogger()
		cfg := Config{
			Path:            "/invalid/path/to/database.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1 * time.Hour,
			WALMode:         true,
			ForeignKeys:     true,
			BusyTimeout:     5 * time.Second,
		}

		_, err := New(cfg, logger)
		assert.Error(t, err)
	})
}

func TestDB_ExecContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert data
	result, err := db.ExecContext(ctx, "INSERT INTO test_users (username, email) VALUES (?, ?)", "testuser", "test@example.com")
	require.NoError(t, err)

	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
}

func TestDB_QueryContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create and populate test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "INSERT INTO test_users (username, email) VALUES (?, ?)", "alice", "alice@example.com")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO test_users (username, email) VALUES (?, ?)", "bob", "bob@example.com")
	require.NoError(t, err)

	// Query data
	rows, err := db.QueryContext(ctx, "SELECT username, email FROM test_users ORDER BY username")
	require.NoError(t, err)
	defer rows.Close()

	var users []struct {
		Username string
		Email    string
	}

	for rows.Next() {
		var user struct {
			Username string
			Email    string
		}
		err := rows.Scan(&user.Username, &user.Email)
		require.NoError(t, err)
		users = append(users, user)
	}

	require.NoError(t, rows.Err())
	assert.Len(t, users, 2)
	assert.Equal(t, "alice", users[0].Username)
	assert.Equal(t, "bob", users[1].Username)
}

func TestDB_QueryRowContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create and populate test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "INSERT INTO test_users (username) VALUES (?)", "testuser")
	require.NoError(t, err)

	// Query single row
	var username string
	err = db.QueryRowContext(ctx, "SELECT username FROM test_users WHERE id = ?", 1).Scan(&username)
	require.NoError(t, err)
	assert.Equal(t, "testuser", username)
}

func TestDB_InTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	t.Run("commits on success", func(t *testing.T) {
		err := db.InTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_users (username) VALUES (?)", "alice")
			return err
		})
		require.NoError(t, err)

		// Verify data was committed
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_users").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("rolls back on error", func(t *testing.T) {
		err := db.InTransaction(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO test_users (username) VALUES (?)", "bob")
			if err != nil {
				return err
			}
			return fmt.Errorf("intentional error")
		})
		require.Error(t, err)

		// Verify data was not committed
		var count int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_users").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count) // Should still be 1 from previous test
	})
}

func TestDB_Vacuum(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	err := db.Vacuum(ctx)
	assert.NoError(t, err)
}

func TestDB_Optimize(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	err := db.Optimize(ctx)
	assert.NoError(t, err)
}

func TestDB_GetDatabaseSize(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a table to ensure database has some size
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY,
			data TEXT
		)
	`)
	require.NoError(t, err)

	size, err := db.GetDatabaseSize(ctx)
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))
}

func TestDB_Backup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create and populate test table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE test_users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, "INSERT INTO test_users (username) VALUES (?)", "testuser")
	require.NoError(t, err)

	// Create backup
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.db")

	err = db.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Verify backup exists
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)

	// Verify backup is valid SQLite database
	logger := observability.NewNopLogger()
	backupDB, err := New(Config{
		Path:            backupPath,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 1 * time.Hour,
		WALMode:         false,
		ForeignKeys:     true,
		BusyTimeout:     5 * time.Second,
	}, logger)
	require.NoError(t, err)
	defer backupDB.Close()

	// Verify data is in backup
	var username string
	err = backupDB.QueryRowContext(ctx, "SELECT username FROM test_users WHERE id = 1").Scan(&username)
	require.NoError(t, err)
	assert.Equal(t, "testuser", username)
}

func TestDB_CheckIntegrity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	results, err := db.CheckIntegrity(ctx)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "ok", results[0])
}

func TestDB_Stats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats := db.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1)
}

// setupTestDB creates a test database in a temporary directory
func setupTestDB(t *testing.T) *DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	logger := observability.NewNopLogger()
	cfg := Config{
		Path:            dbPath,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 1 * time.Hour,
		WALMode:         true,
		ForeignKeys:     true,
		BusyTimeout:     5 * time.Second,
	}

	db, err := New(cfg, logger)
	require.NoError(t, err)

	return db
}

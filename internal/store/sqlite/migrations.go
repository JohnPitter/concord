package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// Migration represents a single database migration
type Migration struct {
	Version   int
	Name      string
	SQL       string
	AppliedAt time.Time
}

// Migrator handles database migrations
type Migrator struct {
	db     *DB
	logger zerolog.Logger
}

// NewMigrator creates a new migration manager
func NewMigrator(db *DB, logger zerolog.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// Migrate runs all pending migrations
// Complexity: O(n) where n is the number of pending migrations
func (m *Migrator) Migrate(ctx context.Context) error {
	m.logger.Info().Msg("starting database migration")

	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Load all migrations from embedded files
	migrations, err := m.loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Filter out already applied migrations
	pending := m.filterPendingMigrations(migrations, applied)

	if len(pending) == 0 {
		m.logger.Info().Msg("no pending migrations")
		return nil
	}

	m.logger.Info().
		Int("pending_count", len(pending)).
		Msg("found pending migrations")

	// Apply each pending migration
	for _, migration := range pending {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	m.logger.Info().
		Int("applied_count", len(pending)).
		Msg("all migrations applied successfully")

	return nil
}

// Rollback rolls back the last migration
func (m *Migrator) Rollback(ctx context.Context) error {
	m.logger.Info().Msg("rolling back last migration")

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		m.logger.Info().Msg("no migrations to rollback")
		return nil
	}

	// Get the last applied migration
	lastMigration := applied[len(applied)-1]

	// Remove from migrations table
	query := "DELETE FROM schema_migrations WHERE version = ?"
	if _, err := m.db.ExecContext(ctx, query, lastMigration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	m.logger.Info().
		Int("version", lastMigration.Version).
		Str("name", lastMigration.Name).
		Msg("migration rolled back successfully")

	return nil
}

// Status returns the current migration status
func (m *Migrator) Status(ctx context.Context) ([]Migration, error) {
	// Ensure migrations table exists before querying
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure migrations table: %w", err)
	}
	return m.getAppliedMigrations(ctx)
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist
func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return err
	}

	return nil
}

// getAppliedMigrations retrieves all applied migrations from the database
// Complexity: O(n log n) where n is the number of applied migrations (due to sorting)
func (m *Migrator) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := "SELECT version, name, applied_at FROM schema_migrations ORDER BY version ASC"

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		if err := rows.Scan(&migration.Version, &migration.Name, &migration.AppliedAt); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return migrations, nil
}

// loadMigrations loads all migration files from the embedded filesystem
// Complexity: O(n log n) where n is the number of migration files (due to sorting)
func (m *Migrator) loadMigrations() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []Migration

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Parse version and name from filename
		// Expected format: 001_init.sql
		version, name := parseMigrationFilename(entry.Name())
		if version == 0 {
			m.logger.Warn().
				Str("filename", entry.Name()).
				Msg("skipping invalid migration filename")
			continue
		}

		// Read migration SQL
		// Note: embed.FS always uses forward slashes, not filepath.Join which uses OS separators
		sql, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(sql),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFilename extracts version and name from migration filename
// Expected format: 001_init.sql -> version=1, name="init"
func parseMigrationFilename(filename string) (int, string) {
	// Remove .sql extension
	name := strings.TrimSuffix(filename, ".sql")

	// Split by underscore
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, ""
	}

	// Parse version
	var version int
	if _, err := fmt.Sscanf(parts[0], "%d", &version); err != nil {
		return 0, ""
	}

	return version, parts[1]
}

// filterPendingMigrations returns migrations that haven't been applied yet
// Complexity: O(n*m) where n is total migrations and m is applied migrations
// Could be optimized to O(n) with a map
func (m *Migrator) filterPendingMigrations(all, applied []Migration) []Migration {
	appliedVersions := make(map[int]bool)
	for _, migration := range applied {
		appliedVersions[migration.Version] = true
	}

	var pending []Migration
	for _, migration := range all {
		if !appliedVersions[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending
}

// applyMigration applies a single migration within a transaction
func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	m.logger.Info().
		Int("version", migration.Version).
		Str("name", migration.Name).
		Msg("applying migration")

	start := time.Now()

	err := m.db.InTransaction(ctx, func(tx *sql.Tx) error {
		// Execute migration SQL
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("failed to execute migration SQL: %w", err)
		}

		// Record migration in schema_migrations table
		insertQuery := "INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)"
		if _, err := tx.ExecContext(ctx, insertQuery, migration.Version, migration.Name, time.Now()); err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		return nil
	})

	duration := time.Since(start)

	if err != nil {
		m.logger.Error().
			Err(err).
			Int("version", migration.Version).
			Str("name", migration.Name).
			Dur("duration_ms", duration).
			Msg("migration failed")
		return err
	}

	m.logger.Info().
		Int("version", migration.Version).
		Str("name", migration.Name).
		Dur("duration_ms", duration).
		Msg("migration applied successfully")

	return nil
}

// Reset drops all tables and resets the database (DANGEROUS)
func (m *Migrator) Reset(ctx context.Context) error {
	m.logger.Warn().Msg("resetting database - this will drop all tables")

	// Get all table names
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return err
		}
		tables = append(tables, table)
	}

	// Drop all tables
	for _, table := range tables {
		dropQuery := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
		if _, err := m.db.ExecContext(ctx, dropQuery); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
		m.logger.Info().Str("table", table).Msg("table dropped")
	}

	m.logger.Info().Msg("database reset complete")

	return nil
}

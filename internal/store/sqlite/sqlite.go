package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// DB wraps a SQLite database connection with additional functionality
type DB struct {
	conn   *sql.DB
	path   string
	logger zerolog.Logger
}

// Config contains configuration for SQLite connection
type Config struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	WALMode         bool
	ForeignKeys     bool
	BusyTimeout     time.Duration
}

// New creates a new SQLite database connection
// Complexity: O(1)
func New(cfg Config, logger zerolog.Logger) (*DB, error) {
	logger.Info().
		Str("path", cfg.Path).
		Bool("wal_mode", cfg.WALMode).
		Bool("foreign_keys", cfg.ForeignKeys).
		Msg("initializing sqlite database")

	// Build DSN with pragmas
	dsn := buildDSN(cfg)

	// Open database connection
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)
	conn.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn:   conn,
		path:   cfg.Path,
		logger: logger,
	}

	// Apply additional pragmas
	if err := db.applyPragmas(cfg); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to apply pragmas: %w", err)
	}

	logger.Info().Msg("sqlite database initialized successfully")

	return db, nil
}

// buildDSN builds the SQLite DSN with pragmas
func buildDSN(cfg Config) string {
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", cfg.Path)

	if cfg.BusyTimeout > 0 {
		dsn += fmt.Sprintf("&_busy_timeout=%d", cfg.BusyTimeout.Milliseconds())
	}

	return dsn
}

// applyPragmas applies SQLite pragmas to the connection
func (db *DB) applyPragmas(cfg Config) error {
	pragmas := []string{}

	// Enable Write-Ahead Logging for better concurrency
	if cfg.WALMode {
		pragmas = append(pragmas, "PRAGMA journal_mode=WAL")
		pragmas = append(pragmas, "PRAGMA synchronous=NORMAL") // NORMAL is safe with WAL
	} else {
		pragmas = append(pragmas, "PRAGMA synchronous=FULL")
	}

	// Enable foreign key constraints
	if cfg.ForeignKeys {
		pragmas = append(pragmas, "PRAGMA foreign_keys=ON")
	}

	// Performance optimizations
	pragmas = append(pragmas,
		"PRAGMA temp_store=MEMORY",       // Store temp tables in memory
		"PRAGMA mmap_size=30000000000",   // Use memory-mapped I/O (30GB)
		"PRAGMA page_size=4096",          // 4KB page size
		"PRAGMA cache_size=-64000",       // 64MB cache (negative means KB)
	)

	// Execute pragmas
	for _, pragma := range pragmas {
		if _, err := db.conn.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
		db.logger.Debug().Str("pragma", pragma).Msg("pragma applied")
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info().Msg("closing sqlite database")
	return db.conn.Close()
}

// Conn returns the underlying *sql.DB connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// Ping checks if the database connection is alive
// Complexity: O(1)
func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

// Stats returns database statistics
func (db *DB) Stats() sql.DBStats {
	return db.conn.Stats()
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.conn.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning any rows
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := db.conn.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	db.logger.Debug().
		Str("query", query).
		Dur("duration_ms", duration).
		Err(err).
		Msg("executed query")

	return result, err
}

// QueryContext executes a query that returns rows
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.conn.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	db.logger.Debug().
		Str("query", query).
		Dur("duration_ms", duration).
		Err(err).
		Msg("executed query")

	return rows, err
}

// QueryRowContext executes a query that returns at most one row
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := db.conn.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	db.logger.Debug().
		Str("query", query).
		Dur("duration_ms", duration).
		Msg("executed query")

	return row
}

// Vacuum performs a VACUUM operation to reclaim space
func (db *DB) Vacuum(ctx context.Context) error {
	db.logger.Info().Msg("running vacuum on database")
	start := time.Now()

	_, err := db.conn.ExecContext(ctx, "VACUUM")
	duration := time.Since(start)

	if err != nil {
		db.logger.Error().
			Err(err).
			Dur("duration_ms", duration).
			Msg("vacuum failed")
		return err
	}

	db.logger.Info().
		Dur("duration_ms", duration).
		Msg("vacuum completed successfully")

	return nil
}

// Optimize runs OPTIMIZE to update statistics
func (db *DB) Optimize(ctx context.Context) error {
	db.logger.Info().Msg("optimizing database")
	start := time.Now()

	_, err := db.conn.ExecContext(ctx, "PRAGMA optimize")
	duration := time.Since(start)

	if err != nil {
		db.logger.Error().
			Err(err).
			Dur("duration_ms", duration).
			Msg("optimize failed")
		return err
	}

	db.logger.Info().
		Dur("duration_ms", duration).
		Msg("optimize completed successfully")

	return nil
}

// GetDatabaseSize returns the size of the database file in bytes
func (db *DB) GetDatabaseSize(ctx context.Context) (int64, error) {
	var pageCount, pageSize int64

	err := db.conn.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return 0, fmt.Errorf("failed to get page count: %w", err)
	}

	err = db.conn.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get page size: %w", err)
	}

	return pageCount * pageSize, nil
}

// Backup creates a backup of the database to the specified path
func (db *DB) Backup(ctx context.Context, destPath string) error {
	db.logger.Info().
		Str("dest_path", destPath).
		Msg("creating database backup")

	start := time.Now()

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	query := fmt.Sprintf("VACUUM INTO '%s'", destPath)

	_, err := db.conn.ExecContext(ctx, query)
	duration := time.Since(start)

	if err != nil {
		db.logger.Error().
			Err(err).
			Str("dest_path", destPath).
			Dur("duration_ms", duration).
			Msg("backup failed")
		return fmt.Errorf("backup failed: %w", err)
	}

	db.logger.Info().
		Str("dest_path", destPath).
		Str("dest_dir", destDir).
		Dur("duration_ms", duration).
		Msg("backup completed successfully")

	return nil
}

// CheckIntegrity runs PRAGMA integrity_check
func (db *DB) CheckIntegrity(ctx context.Context) ([]string, error) {
	db.logger.Info().Msg("checking database integrity")

	rows, err := db.conn.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return nil, fmt.Errorf("integrity check failed: %w", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(results) == 1 && results[0] == "ok" {
		db.logger.Info().Msg("database integrity check passed")
	} else {
		db.logger.Warn().
			Strs("issues", results).
			Msg("database integrity check found issues")
	}

	return results, nil
}

// InTransaction executes a function within a transaction
// Automatically rolls back on error, commits on success
// Complexity: O(f) where f is the complexity of the function
func (db *DB) InTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.Error().
				Err(rbErr).
				Msg("failed to rollback transaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

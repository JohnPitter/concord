package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Adapter wraps a *sql.DB and translates SQLite-style ? placeholders to PostgreSQL $N style.
// It implements the querier interface used by auth, server, and chat repositories.
type Adapter struct {
	db *sql.DB
}

// NewAdapter creates a new PostgreSQL adapter from a *sql.DB.
func NewAdapter(db *sql.DB) *Adapter {
	return &Adapter{db: db}
}

// ExecContext executes a query with ? placeholders translated to $N.
func (a *Adapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.db.ExecContext(ctx, replacePlaceholders(query), args...)
}

// QueryRowContext executes a query that returns a single row, with ? placeholders translated to $N.
func (a *Adapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return a.db.QueryRowContext(ctx, replacePlaceholders(query), args...)
}

// QueryContext executes a query that returns rows, with ? placeholders translated to $N.
func (a *Adapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.db.QueryContext(ctx, replacePlaceholders(query), args...)
}

// DB returns the underlying *sql.DB for direct access when needed.
func (a *Adapter) DB() *sql.DB {
	return a.db
}

// genericQuerier is a minimal interface for wrapping any querier-compatible type.
type genericQuerier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// QuerierAdapter wraps any querier and translates SQLite-style ? placeholders to PostgreSQL $N.
// Used to wrap transaction queriers inside friends repository.
type QuerierAdapter struct {
	inner genericQuerier
}

// NewQuerierAdapter creates a placeholder-translating wrapper around any querier.
func NewQuerierAdapter(q genericQuerier) *QuerierAdapter {
	return &QuerierAdapter{inner: q}
}

func (a *QuerierAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return a.inner.ExecContext(ctx, replacePlaceholders(query), args...)
}

func (a *QuerierAdapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return a.inner.QueryRowContext(ctx, replacePlaceholders(query), args...)
}

func (a *QuerierAdapter) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return a.inner.QueryContext(ctx, replacePlaceholders(query), args...)
}

// replacePlaceholders replaces SQLite-style ? placeholders with PostgreSQL $1, $2, ... style.
// It correctly handles ? inside single-quoted string literals by skipping them.
// Complexity: O(n) where n is the length of the query string.
func replacePlaceholders(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 16)
	n := 1
	inString := false
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' {
			inString = !inString
			b.WriteByte(ch)
		} else if ch == '?' && !inString {
			fmt.Fprintf(&b, "$%d", n)
			n++
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}

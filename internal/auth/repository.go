package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

// User represents a Concord user stored in the local database.
type User struct {
	ID          string    `json:"id"`
	GitHubID    int64     `json:"github_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	AvatarURL   string    `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Session represents an auth session stored locally.
type Session struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	RefreshTokenHash  string    `json:"refresh_token_hash"`
	EncryptedRefresh  string    `json:"encrypted_refresh"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// Repository handles auth-related database operations.
type Repository struct {
	db     querier
	logger zerolog.Logger
}

type querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// NewRepository creates a new auth repository.
func NewRepository(db querier, logger zerolog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger.With().Str("component", "auth_repo").Logger(),
	}
}

// UpsertUser creates or updates a user from GitHub profile data.
// Complexity: O(1)
func (r *Repository) UpsertUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, github_id, username, display_name, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			username = excluded.username,
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.ExecContext(ctx, query, user.ID, user.GitHubID, user.Username, user.DisplayName, user.AvatarURL)
	if err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	r.logger.Info().
		Str("user_id", user.ID).
		Str("username", user.Username).
		Msg("user upserted")

	return nil
}

// GetUserByGitHubID retrieves a user by their GitHub ID.
// Complexity: O(1) — indexed lookup
func (r *Repository) GetUserByGitHubID(ctx context.Context, githubID int64) (*User, error) {
	query := `SELECT id, github_id, username, display_name, avatar_url, created_at, updated_at
		FROM users WHERE github_id = ?`

	var user User
	err := r.db.QueryRowContext(ctx, query, githubID).Scan(
		&user.ID, &user.GitHubID, &user.Username, &user.DisplayName,
		&user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by github_id: %w", err)
	}

	return &user, nil
}

// SaveSession stores an encrypted refresh token session.
// Complexity: O(1)
func (r *Repository) SaveSession(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO auth_sessions (id, user_id, refresh_token_hash, encrypted_refresh, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := r.db.ExecContext(ctx, query, session.ID, session.UserID, session.RefreshTokenHash, session.EncryptedRefresh, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	r.logger.Info().
		Str("session_id", session.ID).
		Str("user_id", session.UserID).
		Msg("session saved")

	return nil
}

// GetActiveSession retrieves the most recent non-expired session for a user.
// Complexity: O(1)
func (r *Repository) GetActiveSession(ctx context.Context, userID string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, encrypted_refresh, expires_at, created_at
		FROM auth_sessions
		WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC LIMIT 1
	`

	var session Session
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&session.ID, &session.UserID, &session.RefreshTokenHash,
		&session.EncryptedRefresh, &session.ExpiresAt, &session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}

	return &session, nil
}

// DeleteUserSessions removes all sessions for a user (logout).
// Complexity: O(n) where n = number of sessions
func (r *Repository) DeleteUserSessions(ctx context.Context, userID string) error {
	query := `DELETE FROM auth_sessions WHERE user_id = ?`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	r.logger.Info().
		Str("user_id", userID).
		Msg("user sessions deleted")

	return nil
}

// GetUser retrieves a user by their primary ID.
// Complexity: O(1) — indexed lookup
func (r *Repository) GetUser(ctx context.Context, userID string) (*User, error) {
	query := `SELECT id, github_id, username, display_name, avatar_url, created_at, updated_at
		FROM users WHERE id = ?`

	var user User
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.GitHubID, &user.Username, &user.DisplayName,
		&user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// CleanExpiredSessions removes all expired sessions.
// Complexity: O(n) where n = number of expired sessions
func (r *Repository) CleanExpiredSessions(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth_sessions WHERE expires_at <= CURRENT_TIMESTAMP`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to clean expired sessions: %w", err)
	}

	count, _ := result.RowsAffected()
	if count > 0 {
		r.logger.Info().Int64("count", count).Msg("expired sessions cleaned")
	}

	return count, nil
}

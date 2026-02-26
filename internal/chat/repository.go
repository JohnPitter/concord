package chat

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

type querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Repository handles message-related database operations.
type Repository struct {
	db     querier
	logger zerolog.Logger
}

// NewRepository creates a new chat repository.
func NewRepository(db querier, logger zerolog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger.With().Str("component", "chat_repo").Logger(),
	}
}

// Save inserts a new message.
// Complexity: O(1) + O(log n) FTS index update via trigger
func (r *Repository) Save(ctx context.Context, msg *Message) error {
	query := `INSERT INTO messages (id, channel_id, author_id, content, type, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.ChannelID, msg.AuthorID, msg.Content, msg.Type)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	r.logger.Debug().
		Str("message_id", msg.ID).
		Str("channel_id", msg.ChannelID).
		Str("author_id", msg.AuthorID).
		Msg("message saved")

	return nil
}

// GetByID retrieves a single message by ID with author info.
// Complexity: O(1)
func (r *Repository) GetByID(ctx context.Context, id string) (*Message, error) {
	query := `SELECT m.id, m.channel_id, m.author_id, m.content, m.type, m.edited_at, m.created_at,
			u.username, COALESCE(u.avatar_url, '')
		FROM messages m
		INNER JOIN users u ON m.author_id = u.id
		WHERE m.id = ?`

	var msg Message
	var editedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content, &msg.Type,
		&editedAt, &msg.CreatedAt, &msg.AuthorName, &msg.AuthorAvatar,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}
	if editedAt.Valid {
		s := editedAt.Time.UTC().Format(time.RFC3339)
		msg.EditedAt = &s
	}
	return &msg, nil
}

// GetByChannel retrieves messages for a channel with cursor-based pagination.
// Returns messages ordered by created_at DESC (newest first).
// Complexity: O(log n) — indexed on (channel_id, created_at DESC)
func (r *Repository) GetByChannel(ctx context.Context, channelID string, opts PaginationOpts) ([]*Message, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var query string
	var args []interface{}

	if opts.Before != "" {
		// Load messages older than the given message
		query = `SELECT m.id, m.channel_id, m.author_id, m.content, m.type, m.edited_at, m.created_at,
				u.username, COALESCE(u.avatar_url, '')
			FROM messages m
			INNER JOIN users u ON m.author_id = u.id
			WHERE m.channel_id = ? AND m.created_at < (SELECT created_at FROM messages WHERE id = ?)
			ORDER BY m.created_at DESC
			LIMIT ?`
		args = []interface{}{channelID, opts.Before, limit}
	} else if opts.After != "" {
		// Load messages newer than the given message
		query = `SELECT m.id, m.channel_id, m.author_id, m.content, m.type, m.edited_at, m.created_at,
				u.username, COALESCE(u.avatar_url, '')
			FROM messages m
			INNER JOIN users u ON m.author_id = u.id
			WHERE m.channel_id = ? AND m.created_at >= (SELECT created_at FROM messages WHERE id = ?)
			ORDER BY m.created_at ASC
			LIMIT ?`
		args = []interface{}{channelID, opts.After, limit}
	} else {
		// Load most recent messages
		query = `SELECT m.id, m.channel_id, m.author_id, m.content, m.type, m.edited_at, m.created_at,
				u.username, COALESCE(u.avatar_url, '')
			FROM messages m
			INNER JOIN users u ON m.author_id = u.id
			WHERE m.channel_id = ?
			ORDER BY m.created_at DESC
			LIMIT ?`
		args = []interface{}{channelID, limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var editedAt sql.NullTime
		if err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.AuthorID, &msg.Content, &msg.Type,
			&editedAt, &msg.CreatedAt, &msg.AuthorName, &msg.AuthorAvatar,
		); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		if editedAt.Valid {
			s := editedAt.Time.UTC().Format(time.RFC3339)
			msg.EditedAt = &s
		}
		messages = append(messages, &msg)
	}

	return messages, rows.Err()
}

// Update modifies the content of an existing message and sets edited_at.
// Complexity: O(1) + O(log n) FTS update via trigger
func (r *Repository) Update(ctx context.Context, id, content string) error {
	query := `UPDATE messages SET content = ?, edited_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, content, id)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found")
	}

	r.logger.Debug().Str("message_id", id).Msg("message updated")
	return nil
}

// Delete removes a message by ID.
// Complexity: O(1) + O(log n) FTS cleanup via trigger
func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM messages WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("message not found")
	}

	r.logger.Debug().Str("message_id", id).Msg("message deleted")
	return nil
}

// Search performs full-text search across messages in a channel using FTS5.
// Complexity: O(log n) — FTS5 inverted index lookup
func (r *Repository) Search(ctx context.Context, channelID, query string, limit int) ([]*SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	sqlQuery := `SELECT m.id, m.channel_id, m.author_id, m.content, m.type, m.edited_at, m.created_at,
			u.username, COALESCE(u.avatar_url, ''),
			snippet(messages_fts, 0, '<mark>', '</mark>', '...', 32) as snippet
		FROM messages_fts
		INNER JOIN messages m ON messages_fts.rowid = m.rowid
		INNER JOIN users u ON m.author_id = u.id
		WHERE m.channel_id = ? AND messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, sqlQuery, channelID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var sr SearchResult
		var editedAt sql.NullTime
		if err := rows.Scan(
			&sr.ID, &sr.ChannelID, &sr.AuthorID, &sr.Content, &sr.Type,
			&editedAt, &sr.CreatedAt, &sr.AuthorName, &sr.AuthorAvatar,
			&sr.Snippet,
		); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		if editedAt.Valid {
			s := editedAt.Time.UTC().Format(time.RFC3339)
			sr.EditedAt = &s
		}
		results = append(results, &sr)
	}

	r.logger.Info().
		Str("channel_id", channelID).
		Str("query", query).
		Int("results", len(results)).
		Msg("message search completed")

	return results, rows.Err()
}

// GetUnreadCounts returns how many messages exist after a given message ID per channel.
// lastRead maps channel_id → last_read_message_id. Returns channel_id → unread count.
// Complexity: O(k * log n) where k = number of channels
func (r *Repository) GetUnreadCounts(ctx context.Context, lastRead map[string]string) (map[string]int64, error) {
	result := make(map[string]int64, len(lastRead))
	for channelID, afterMsgID := range lastRead {
		var count int64
		var err error
		if afterMsgID == "" {
			// Never read — count all messages
			err = r.db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM messages WHERE channel_id = ?`, channelID,
			).Scan(&count)
		} else {
			err = r.db.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM messages WHERE channel_id = ? AND created_at > (SELECT created_at FROM messages WHERE id = ?)`,
				channelID, afterMsgID,
			).Scan(&count)
		}
		if err != nil {
			r.logger.Warn().Err(err).Str("channel_id", channelID).Msg("failed to get unread count")
			continue
		}
		if count > 0 {
			result[channelID] = count
		}
	}
	return result, nil
}

// CountByChannel returns the total message count for a channel.
// Complexity: O(1) with index
func (r *Repository) CountByChannel(ctx context.Context, channelID string) (int64, error) {
	query := `SELECT COUNT(*) FROM messages WHERE channel_id = ?`
	var count int64
	err := r.db.QueryRowContext(ctx, query, channelID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}
	return count, nil
}

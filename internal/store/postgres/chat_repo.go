package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/concord-chat/concord/internal/chat"
	"github.com/rs/zerolog"
)

// ChatSearcher provides PostgreSQL-specific full-text search for messages.
// It uses tsvector/tsquery instead of SQLite FTS5.
type ChatSearcher struct {
	db     *DB
	logger zerolog.Logger
}

// NewChatSearcher creates a new PostgreSQL chat searcher.
func NewChatSearcher(db *DB, logger zerolog.Logger) *ChatSearcher {
	return &ChatSearcher{
		db:     db,
		logger: logger.With().Str("component", "pg_chat_search").Logger(),
	}
}

// Search performs full-text search across messages in a channel using PostgreSQL tsvector.
// Uses plainto_tsquery for safe user input (no special syntax needed).
// ts_headline generates snippets with <mark> highlighting.
// Complexity: O(log n) -- GIN index lookup
func (s *ChatSearcher) Search(ctx context.Context, channelID, query string, limit int) ([]*chat.SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// plainto_tsquery safely converts user input into a tsquery (no special operators needed).
	// ts_headline produces HTML-highlighted snippets matching the query.
	// search_vector @@ plainto_tsquery uses the GIN index for fast lookup.
	// ts_rank orders results by relevance.
	sqlQuery := `
		SELECT m.id, m.channel_id, m.author_id, m.content, m.type, m.edited_at, m.created_at,
			u.username, COALESCE(u.avatar_url, ''),
			ts_headline('english', m.content, plainto_tsquery('english', $1),
				'StartSel=<mark>, StopSel=</mark>, MaxFragments=1, MaxWords=32') AS snippet
		FROM messages m
		INNER JOIN users u ON m.author_id = u.id
		WHERE m.channel_id = $2 AND m.search_vector @@ plainto_tsquery('english', $3)
		ORDER BY ts_rank(m.search_vector, plainto_tsquery('english', $4)) DESC
		LIMIT $5`

	rows, err := s.db.pool.Query(ctx, sqlQuery, query, channelID, query, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var results []*chat.SearchResult
	for rows.Next() {
		var sr chat.SearchResult
		var editedAt *time.Time
		var createdAt time.Time

		if err := rows.Scan(
			&sr.ID, &sr.ChannelID, &sr.AuthorID, &sr.Content, &sr.Type,
			&editedAt, &createdAt, &sr.AuthorName, &sr.AuthorAvatar,
			&sr.Snippet,
		); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		sr.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if editedAt != nil {
			t := editedAt.UTC().Format(time.RFC3339)
			sr.EditedAt = &t
		}

		results = append(results, &sr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	s.logger.Info().
		Str("channel_id", channelID).
		Str("query", query).
		Int("results", len(results)).
		Msg("pg message search completed")

	return results, nil
}

package files

import (
	"context"
	"fmt"

	"github.com/concord-chat/concord/internal/store/sqlite"
	"github.com/rs/zerolog"
)

// Repository handles attachment persistence in SQLite.
type Repository struct {
	db     *sqlite.DB
	logger zerolog.Logger
}

// NewRepository creates a new file attachment repository.
func NewRepository(db *sqlite.DB, logger zerolog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger.With().Str("component", "file_repo").Logger(),
	}
}

// Save inserts a new attachment record.
func (r *Repository) Save(ctx context.Context, a *Attachment) error {
	query := `INSERT INTO attachments (id, message_id, filename, size_bytes, mime_type, hash, local_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.MessageID, a.Filename, a.SizeBytes, a.MimeType, a.Hash, a.LocalPath, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("files: save attachment: %w", err)
	}
	return nil
}

// GetByID retrieves an attachment by ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Attachment, error) {
	query := `SELECT id, message_id, filename, size_bytes, mime_type, hash, local_path, created_at
		FROM attachments WHERE id = ?`

	var a Attachment
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID, &a.MessageID, &a.Filename, &a.SizeBytes, &a.MimeType, &a.Hash, &a.LocalPath, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("files: get attachment: %w", err)
	}
	return &a, nil
}

// GetByMessageID returns all attachments for a message.
func (r *Repository) GetByMessageID(ctx context.Context, messageID string) ([]*Attachment, error) {
	query := `SELECT id, message_id, filename, size_bytes, mime_type, hash, local_path, created_at
		FROM attachments WHERE message_id = ? ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("files: list attachments: %w", err)
	}
	defer rows.Close()

	var attachments []*Attachment
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.Filename, &a.SizeBytes, &a.MimeType, &a.Hash, &a.LocalPath, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("files: scan attachment: %w", err)
		}
		attachments = append(attachments, &a)
	}
	return attachments, rows.Err()
}

// GetByHash finds an attachment by its SHA-256 hash (deduplication).
func (r *Repository) GetByHash(ctx context.Context, hash string) (*Attachment, error) {
	query := `SELECT id, message_id, filename, size_bytes, mime_type, hash, local_path, created_at
		FROM attachments WHERE hash = ? LIMIT 1`

	var a Attachment
	err := r.db.QueryRowContext(ctx, query, hash).Scan(
		&a.ID, &a.MessageID, &a.Filename, &a.SizeBytes, &a.MimeType, &a.Hash, &a.LocalPath, &a.CreatedAt)
	if err != nil {
		return nil, err // may be sql.ErrNoRows
	}
	return &a, nil
}

// Delete removes an attachment record.
func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM attachments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("files: delete attachment: %w", err)
	}
	return nil
}

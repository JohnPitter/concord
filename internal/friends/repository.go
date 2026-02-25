package friends

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Querier is the database query interface used by the friends repository.
// Exported so callers (e.g. cmd/server) can wrap it for placeholder translation.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// querier is an alias for internal use.
type querier = Querier

type transactor interface {
	InTransaction(ctx context.Context, fn func(querier) error) error
}

// StdlibTransactor wraps a *sql.DB to implement the transactor interface.
// It provides a querier-compatible wrapper around *sql.Tx so that the caller
// does not need to know about placeholder differences between SQLite and PostgreSQL.
type StdlibTransactor struct {
	db      *sql.DB
	wrapper func(querier) querier // optional wrapper applied to the tx (e.g. placeholder translation)
}

// NewStdlibTransactor creates a transactor from a standard *sql.DB.
func NewStdlibTransactor(db *sql.DB) *StdlibTransactor {
	return &StdlibTransactor{db: db}
}

// NewStdlibTransactorWithWrapper creates a transactor that wraps each transaction's
// querier with the given function (e.g. for placeholder translation on PostgreSQL).
func NewStdlibTransactorWithWrapper(db *sql.DB, wrapper func(querier) querier) *StdlibTransactor {
	return &StdlibTransactor{db: db, wrapper: wrapper}
}

// txQuerier adapts *sql.Tx to the querier interface.
type txQuerier struct {
	tx *sql.Tx
}

func (t *txQuerier) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *txQuerier) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *txQuerier) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// InTransaction runs fn inside a database transaction.
func (t *StdlibTransactor) InTransaction(ctx context.Context, fn func(querier) error) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	var q querier = &txQuerier{tx: tx}
	if t.wrapper != nil {
		q = t.wrapper(q)
	}
	if err := fn(q); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// Repository handles friend-related database operations.
type Repository struct {
	db     querier
	tx     transactor
	logger zerolog.Logger
}

// NewRepository creates a new friends repository.
func NewRepository(db querier, tx transactor, logger zerolog.Logger) *Repository {
	return &Repository{
		db:     db,
		tx:     tx,
		logger: logger.With().Str("component", "friends_repo").Logger(),
	}
}

// GetUserByUsername looks up a user by their username.
// Complexity: O(1) — indexed lookup.
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (id, displayName, avatarURL string, err error) {
	query := `SELECT id, COALESCE(display_name, username), COALESCE(avatar_url, '') FROM users WHERE username = ?`
	err = r.db.QueryRowContext(ctx, query, username).Scan(&id, &displayName, &avatarURL)
	if err == sql.ErrNoRows {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", fmt.Errorf("lookup user by username: %w", err)
	}
	return id, displayName, avatarURL, nil
}

// SendRequest creates a friend request from senderID to receiverID.
// Complexity: O(1).
func (r *Repository) SendRequest(ctx context.Context, senderID, receiverID string) (*FriendRequest, error) {
	id := uuid.New().String()
	query := `INSERT INTO friend_requests (id, sender_id, receiver_id, status)
		VALUES (?, ?, ?, 'pending')`

	_, err := r.db.ExecContext(ctx, query, id, senderID, receiverID)
	if err != nil {
		return nil, fmt.Errorf("send friend request: %w", err)
	}

	r.logger.Info().
		Str("request_id", id).
		Str("sender", senderID).
		Str("receiver", receiverID).
		Msg("friend request sent")

	return &FriendRequest{
		ID:         id,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Status:     StatusPending,
	}, nil
}

// GetPendingRequests returns all pending friend requests (incoming + outgoing) for a user.
// Complexity: O(n) where n = number of pending requests.
func (r *Repository) GetPendingRequests(ctx context.Context, userID string) ([]FriendRequestView, error) {
	query := `
		SELECT
			fr.id,
			CASE WHEN fr.sender_id = ? THEN u_recv.username ELSE u_send.username END AS username,
			CASE WHEN fr.sender_id = ? THEN COALESCE(u_recv.display_name, u_recv.username)
				ELSE COALESCE(u_send.display_name, u_send.username) END AS display_name,
			CASE WHEN fr.sender_id = ? THEN COALESCE(u_recv.avatar_url, '')
				ELSE COALESCE(u_send.avatar_url, '') END AS avatar_url,
			CASE WHEN fr.sender_id = ? THEN 'outgoing' ELSE 'incoming' END AS direction,
			fr.created_at
		FROM friend_requests fr
		JOIN users u_send ON u_send.id = fr.sender_id
		JOIN users u_recv ON u_recv.id = fr.receiver_id
		WHERE fr.status = 'pending'
			AND (fr.sender_id = ? OR fr.receiver_id = ?)
		ORDER BY fr.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID, userID, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("get pending requests: %w", err)
	}
	defer rows.Close()

	var results []FriendRequestView
	for rows.Next() {
		var v FriendRequestView
		if err := rows.Scan(&v.ID, &v.Username, &v.DisplayName, &v.AvatarURL, &v.Direction, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pending request: %w", err)
		}
		results = append(results, v)
	}
	if results == nil {
		results = []FriendRequestView{}
	}
	return results, rows.Err()
}

// AcceptRequest accepts a friend request and creates bidirectional friendship.
// Only the receiver can accept.
// Complexity: O(1).
func (r *Repository) AcceptRequest(ctx context.Context, requestID, userID string) error {
	return r.tx.InTransaction(ctx, func(q querier) error {
		// Verify the request exists, is pending, and user is the receiver
		var senderID, receiverID string
		err := q.QueryRowContext(ctx,
			`SELECT sender_id, receiver_id FROM friend_requests WHERE id = ? AND status = 'pending'`,
			requestID,
		).Scan(&senderID, &receiverID)
		if err == sql.ErrNoRows {
			return fmt.Errorf("friend request not found or already processed")
		}
		if err != nil {
			return fmt.Errorf("check request: %w", err)
		}

		if receiverID != userID {
			return fmt.Errorf("only the receiver can accept a friend request")
		}

		// Update status
		_, err = q.ExecContext(ctx,
			`UPDATE friend_requests SET status = 'accepted', updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			requestID,
		)
		if err != nil {
			return fmt.Errorf("update request status: %w", err)
		}

		// Create bidirectional friendship
		_, err = q.ExecContext(ctx,
			`INSERT INTO friends (user_id, friend_id) VALUES (?, ?), (?, ?)
			 ON CONFLICT DO NOTHING`,
			senderID, receiverID, receiverID, senderID,
		)
		if err != nil {
			return fmt.Errorf("create friendship: %w", err)
		}

		r.logger.Info().
			Str("request_id", requestID).
			Str("user_a", senderID).
			Str("user_b", receiverID).
			Msg("friend request accepted")

		return nil
	})
}

// RejectRequest rejects (or cancels) a friend request.
// Both sender and receiver can reject/cancel.
// Complexity: O(1).
func (r *Repository) RejectRequest(ctx context.Context, requestID, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE friend_requests SET status = 'rejected', updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND status = 'pending' AND (sender_id = ? OR receiver_id = ?)`,
		requestID, userID, userID,
	)
	if err != nil {
		return fmt.Errorf("reject request: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("friend request not found or already processed")
	}

	r.logger.Info().Str("request_id", requestID).Str("user_id", userID).Msg("friend request rejected")
	return nil
}

// GetFriends returns all friends for a user with profile info.
// Complexity: O(n) where n = number of friends.
func (r *Repository) GetFriends(ctx context.Context, userID string) ([]FriendView, error) {
	query := `
		SELECT
			u.id,
			u.username,
			COALESCE(u.display_name, u.username),
			COALESCE(u.avatar_url, ''),
			'offline' AS status
		FROM friends f
		JOIN users u ON u.id = f.friend_id
		WHERE f.user_id = ?
		ORDER BY u.username`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get friends: %w", err)
	}
	defer rows.Close()

	var results []FriendView
	for rows.Next() {
		var v FriendView
		if err := rows.Scan(&v.ID, &v.Username, &v.DisplayName, &v.AvatarURL, &v.Status); err != nil {
			return nil, fmt.Errorf("scan friend: %w", err)
		}
		results = append(results, v)
	}
	if results == nil {
		results = []FriendView{}
	}
	return results, rows.Err()
}

// RemoveFriend removes a bidirectional friendship.
// Complexity: O(1).
func (r *Repository) RemoveFriend(ctx context.Context, userID, friendID string) error {
	return r.tx.InTransaction(ctx, func(q querier) error {
		_, err := q.ExecContext(ctx,
			`DELETE FROM friends WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`,
			userID, friendID, friendID, userID,
		)
		if err != nil {
			return fmt.Errorf("remove friend: %w", err)
		}

		// Also clean up any accepted friend_request between them
		_, err = q.ExecContext(ctx,
			`DELETE FROM friend_requests
			 WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)`,
			userID, friendID, friendID, userID,
		)
		if err != nil {
			return fmt.Errorf("clean up friend requests: %w", err)
		}

		r.logger.Info().Str("user_id", userID).Str("friend_id", friendID).Msg("friend removed")
		return nil
	})
}

// BlockUser blocks a user. Removes any existing friendship and marks the request as blocked.
// Complexity: O(1).
func (r *Repository) BlockUser(ctx context.Context, userID, targetID string) error {
	return r.tx.InTransaction(ctx, func(q querier) error {
		// Remove friendship if exists
		_, err := q.ExecContext(ctx,
			`DELETE FROM friends WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`,
			userID, targetID, targetID, userID,
		)
		if err != nil {
			return fmt.Errorf("remove friendship for block: %w", err)
		}

		// Upsert friend_request as blocked
		_, err = q.ExecContext(ctx,
			`INSERT INTO friend_requests (id, sender_id, receiver_id, status)
			 VALUES (?, ?, ?, 'blocked')
			 ON CONFLICT(sender_id, receiver_id) DO UPDATE SET status = 'blocked', updated_at = CURRENT_TIMESTAMP`,
			uuid.New().String(), userID, targetID,
		)
		if err != nil {
			return fmt.Errorf("block user: %w", err)
		}

		r.logger.Info().Str("user_id", userID).Str("target_id", targetID).Msg("user blocked")
		return nil
	})
}

// UnblockUser removes a block.
// Complexity: O(1).
func (r *Repository) UnblockUser(ctx context.Context, userID, targetID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM friend_requests WHERE sender_id = ? AND receiver_id = ? AND status = 'blocked'`,
		userID, targetID,
	)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}

	r.logger.Info().Str("user_id", userID).Str("target_id", targetID).Msg("user unblocked")
	return nil
}

// ExistingRequest checks if there's already a pending or blocked request between two users.
// Complexity: O(1).
func (r *Repository) ExistingRequest(ctx context.Context, userA, userB string) (*FriendRequest, error) {
	query := `SELECT id, sender_id, receiver_id, status, created_at, COALESCE(updated_at, created_at)
		FROM friend_requests
		WHERE ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
			AND status IN ('pending', 'blocked')
		LIMIT 1`

	var req FriendRequest
	err := r.db.QueryRowContext(ctx, query, userA, userB, userB, userA).
		Scan(&req.ID, &req.SenderID, &req.ReceiverID, &req.Status, &req.CreatedAt, &req.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check existing request: %w", err)
	}
	return &req, nil
}

// AreFriends checks if two users are friends.
// Complexity: O(1).
func (r *Repository) AreFriends(ctx context.Context, userA, userB string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM friends WHERE user_id = ? AND friend_id = ?`,
		userA, userB,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check friendship: %w", err)
	}
	return count > 0, nil
}

// SaveDirectMessage creates a direct message between two users.
// Complexity: O(1).
func (r *Repository) SaveDirectMessage(ctx context.Context, senderID, receiverID, content string) (*DirectMessage, error) {
	id := uuid.New().String()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO friend_messages (id, sender_id, receiver_id, content, created_at)
		 VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		id, senderID, receiverID, content,
	)
	if err != nil {
		return nil, fmt.Errorf("save direct message: %w", err)
	}

	var msg DirectMessage
	err = r.db.QueryRowContext(ctx,
		`SELECT id, sender_id, receiver_id, content, created_at
		 FROM friend_messages
		 WHERE id = ?`,
		id,
	).Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("read direct message: %w", err)
	}

	r.logger.Info().
		Str("message_id", id).
		Str("sender_id", senderID).
		Str("receiver_id", receiverID).
		Msg("direct message saved")

	return &msg, nil
}

// GetDirectMessages lists direct messages between two users.
// Returns newest-first to keep parity with channel message APIs.
// Complexity: O(log n) with pair indexes.
func (r *Repository) GetDirectMessages(ctx context.Context, userID, friendID string, opts DMPaginationOpts) ([]DirectMessage, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var (
		query string
		args  []interface{}
	)

	if opts.After != "" {
		query = `
			SELECT id, sender_id, receiver_id, content, created_at
			FROM friend_messages
			WHERE ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
			  AND created_at >= (SELECT created_at FROM friend_messages WHERE id = ?)
			ORDER BY created_at DESC, id DESC
			LIMIT ?`
		args = []interface{}{userID, friendID, friendID, userID, opts.After, limit}
	} else {
		query = `
			SELECT id, sender_id, receiver_id, content, created_at
			FROM friend_messages
			WHERE ((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))
			ORDER BY created_at DESC, id DESC
			LIMIT ?`
		args = []interface{}{userID, friendID, friendID, userID, limit}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get direct messages: %w", err)
	}
	defer rows.Close()

	results := make([]DirectMessage, 0, limit)
	for rows.Next() {
		var msg DirectMessage
		if err := rows.Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan direct message: %w", err)
		}
		results = append(results, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

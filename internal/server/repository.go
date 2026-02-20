package server

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rs/zerolog"
)

type querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Repository handles server-related database operations.
type Repository struct {
	db     querier
	logger zerolog.Logger
}

// NewRepository creates a new server repository.
func NewRepository(db querier, logger zerolog.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger.With().Str("component", "server_repo").Logger(),
	}
}

// --- Server CRUD ---

// CreateServer inserts a new server.
// Complexity: O(1)
func (r *Repository) CreateServer(ctx context.Context, s *Server) error {
	query := `INSERT INTO servers (id, name, icon_url, owner_id, invite_code, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query, s.ID, s.Name, s.IconURL, s.OwnerID, s.InviteCode)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	r.logger.Info().Str("server_id", s.ID).Str("name", s.Name).Msg("server created")
	return nil
}

// GetServer retrieves a server by ID.
// Complexity: O(1)
func (r *Repository) GetServer(ctx context.Context, id string) (*Server, error) {
	query := `SELECT id, name, icon_url, owner_id, invite_code, created_at FROM servers WHERE id = ?`

	var s Server
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.IconURL, &s.OwnerID, &s.InviteCode, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server: %w", err)
	}
	return &s, nil
}

// ListServersByUser retrieves all servers a user belongs to.
// Complexity: O(n) where n = number of user's servers
func (r *Repository) ListServersByUser(ctx context.Context, userID string) ([]*Server, error) {
	query := `SELECT s.id, s.name, s.icon_url, s.owner_id, s.invite_code, s.created_at
		FROM servers s
		INNER JOIN server_members sm ON s.id = sm.server_id
		WHERE sm.user_id = ?
		ORDER BY s.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer rows.Close()

	var servers []*Server
	for rows.Next() {
		var s Server
		if err := rows.Scan(&s.ID, &s.Name, &s.IconURL, &s.OwnerID, &s.InviteCode, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan server: %w", err)
		}
		servers = append(servers, &s)
	}
	return servers, rows.Err()
}

// UpdateServer updates server name and icon.
// Complexity: O(1)
func (r *Repository) UpdateServer(ctx context.Context, id, name, iconURL string) error {
	query := `UPDATE servers SET name = ?, icon_url = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, name, iconURL, id)
	if err != nil {
		return fmt.Errorf("failed to update server: %w", err)
	}
	r.logger.Info().Str("server_id", id).Msg("server updated")
	return nil
}

// DeleteServer removes a server and cascades to channels/members.
// Complexity: O(n) where n = channels + members
func (r *Repository) DeleteServer(ctx context.Context, id string) error {
	query := `DELETE FROM servers WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}
	r.logger.Info().Str("server_id", id).Msg("server deleted")
	return nil
}

// GetServerByInvite retrieves a server by its invite code.
// Complexity: O(1) — indexed lookup
func (r *Repository) GetServerByInvite(ctx context.Context, code string) (*Server, error) {
	query := `SELECT id, name, icon_url, owner_id, invite_code, created_at FROM servers WHERE invite_code = ?`

	var s Server
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&s.ID, &s.Name, &s.IconURL, &s.OwnerID, &s.InviteCode, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get server by invite: %w", err)
	}
	return &s, nil
}

// UpdateInviteCode sets a new invite code for a server.
// Complexity: O(1)
func (r *Repository) UpdateInviteCode(ctx context.Context, serverID, code string) error {
	query := `UPDATE servers SET invite_code = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, code, serverID)
	if err != nil {
		return fmt.Errorf("failed to update invite code: %w", err)
	}
	return nil
}

// --- Channel CRUD ---

// CreateChannel inserts a new channel.
// Complexity: O(1)
func (r *Repository) CreateChannel(ctx context.Context, ch *Channel) error {
	query := `INSERT INTO channels (id, server_id, name, type, position, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query, ch.ID, ch.ServerID, ch.Name, ch.Type, ch.Position)
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}

	r.logger.Info().Str("channel_id", ch.ID).Str("server_id", ch.ServerID).Msg("channel created")
	return nil
}

// ListChannels retrieves all channels for a server, ordered by position.
// Complexity: O(n) where n = number of channels
func (r *Repository) ListChannels(ctx context.Context, serverID string) ([]*Channel, error) {
	query := `SELECT id, server_id, name, type, position, created_at
		FROM channels WHERE server_id = ? ORDER BY position ASC, created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}
	defer rows.Close()

	var channels []*Channel
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.ServerID, &ch.Name, &ch.Type, &ch.Position, &ch.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, &ch)
	}
	return channels, rows.Err()
}

// UpdateChannel updates channel name, type, and position.
// Complexity: O(1)
func (r *Repository) UpdateChannel(ctx context.Context, id, name, chType string, position int) error {
	query := `UPDATE channels SET name = ?, type = ?, position = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, name, chType, position, id)
	if err != nil {
		return fmt.Errorf("failed to update channel: %w", err)
	}
	return nil
}

// DeleteChannel removes a channel.
// Complexity: O(1)
func (r *Repository) DeleteChannel(ctx context.Context, id string) error {
	query := `DELETE FROM channels WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}
	r.logger.Info().Str("channel_id", id).Msg("channel deleted")
	return nil
}

// GetChannel retrieves a channel by ID.
// Complexity: O(1)
func (r *Repository) GetChannel(ctx context.Context, id string) (*Channel, error) {
	query := `SELECT id, server_id, name, type, position, created_at FROM channels WHERE id = ?`

	var ch Channel
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ch.ID, &ch.ServerID, &ch.Name, &ch.Type, &ch.Position, &ch.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}
	return &ch, nil
}

// --- Member Management ---

// AddMember adds a user as a member of a server.
// Complexity: O(1)
func (r *Repository) AddMember(ctx context.Context, serverID, userID string, role Role) error {
	query := `INSERT INTO server_members (server_id, user_id, role, joined_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := r.db.ExecContext(ctx, query, serverID, userID, string(role))
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	r.logger.Info().Str("server_id", serverID).Str("user_id", userID).Str("role", string(role)).Msg("member added")
	return nil
}

// RemoveMember removes a user from a server.
// Complexity: O(1)
func (r *Repository) RemoveMember(ctx context.Context, serverID, userID string) error {
	query := `DELETE FROM server_members WHERE server_id = ? AND user_id = ?`
	_, err := r.db.ExecContext(ctx, query, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}
	r.logger.Info().Str("server_id", serverID).Str("user_id", userID).Msg("member removed")
	return nil
}

// ListMembers retrieves all members of a server with their user info.
// Complexity: O(n) where n = number of members
func (r *Repository) ListMembers(ctx context.Context, serverID string) ([]*Member, error) {
	query := `SELECT sm.server_id, sm.user_id, u.username, COALESCE(u.avatar_url, ''), sm.role, sm.joined_at
		FROM server_members sm
		INNER JOIN users u ON sm.user_id = u.id
		WHERE sm.server_id = ?
		ORDER BY
			CASE sm.role
				WHEN 'owner' THEN 0
				WHEN 'admin' THEN 1
				WHEN 'moderator' THEN 2
				ELSE 3
			END,
			sm.joined_at ASC`

	rows, err := r.db.QueryContext(ctx, query, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	var members []*Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ServerID, &m.UserID, &m.Username, &m.Avatar, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		members = append(members, &m)
	}
	return members, rows.Err()
}

// GetMember retrieves a specific member's role in a server.
// Complexity: O(1)
func (r *Repository) GetMember(ctx context.Context, serverID, userID string) (*Member, error) {
	query := `SELECT sm.server_id, sm.user_id, u.username, COALESCE(u.avatar_url, ''), sm.role, sm.joined_at
		FROM server_members sm
		INNER JOIN users u ON sm.user_id = u.id
		WHERE sm.server_id = ? AND sm.user_id = ?`

	var m Member
	err := r.db.QueryRowContext(ctx, query, serverID, userID).Scan(
		&m.ServerID, &m.UserID, &m.Username, &m.Avatar, &m.Role, &m.JoinedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}
	return &m, nil
}

// UpdateMemberRole changes a member's role.
// Complexity: O(1)
func (r *Repository) UpdateMemberRole(ctx context.Context, serverID, userID string, role Role) error {
	query := `UPDATE server_members SET role = ? WHERE server_id = ? AND user_id = ?`
	_, err := r.db.ExecContext(ctx, query, string(role), serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}
	r.logger.Info().Str("server_id", serverID).Str("user_id", userID).Str("role", string(role)).Msg("member role updated")
	return nil
}

// CountMembers returns the number of members in a server.
// Complexity: O(1)
func (r *Repository) CountMembers(ctx context.Context, serverID string) (int, error) {
	query := `SELECT COUNT(*) FROM server_members WHERE server_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, serverID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count members: %w", err)
	}
	return count, nil
}

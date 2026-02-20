package server

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Service orchestrates server management operations.
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new server management service.
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("component", "server_service").Logger(),
	}
}

// CreateServer creates a new server with a default #general channel.
// The creator becomes the owner.
func (s *Service) CreateServer(ctx context.Context, name, ownerID string) (*Server, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("server name cannot be empty")
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("server name cannot exceed 100 characters")
	}

	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite code: %w", err)
	}

	srv := &Server{
		ID:         uuid.New().String(),
		Name:       strings.TrimSpace(name),
		OwnerID:    ownerID,
		InviteCode: inviteCode,
	}

	if err := s.repo.CreateServer(ctx, srv); err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Add creator as owner
	if err := s.repo.AddMember(ctx, srv.ID, ownerID, RoleOwner); err != nil {
		return nil, fmt.Errorf("failed to add owner: %w", err)
	}

	// Create default #general text channel
	generalCh := &Channel{
		ID:       uuid.New().String(),
		ServerID: srv.ID,
		Name:     "general",
		Type:     "text",
		Position: 0,
	}
	if err := s.repo.CreateChannel(ctx, generalCh); err != nil {
		return nil, fmt.Errorf("failed to create default channel: %w", err)
	}

	// Create default General voice channel
	voiceCh := &Channel{
		ID:       uuid.New().String(),
		ServerID: srv.ID,
		Name:     "General",
		Type:     "voice",
		Position: 1,
	}
	if err := s.repo.CreateChannel(ctx, voiceCh); err != nil {
		return nil, fmt.Errorf("failed to create default voice channel: %w", err)
	}

	s.logger.Info().
		Str("server_id", srv.ID).
		Str("name", srv.Name).
		Str("owner_id", ownerID).
		Msg("server created with default channels")

	return srv, nil
}

// GetServer retrieves a server by ID.
func (s *Service) GetServer(ctx context.Context, serverID string) (*Server, error) {
	return s.repo.GetServer(ctx, serverID)
}

// ListUserServers returns all servers a user belongs to.
func (s *Service) ListUserServers(ctx context.Context, userID string) ([]*Server, error) {
	return s.repo.ListServersByUser(ctx, userID)
}

// UpdateServer updates server name and icon. Requires PermManageServer.
func (s *Service) UpdateServer(ctx context.Context, serverID, userID, name, iconURL string) error {
	if err := s.requirePermission(ctx, serverID, userID, PermManageServer); err != nil {
		return err
	}

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	return s.repo.UpdateServer(ctx, serverID, strings.TrimSpace(name), iconURL)
}

// DeleteServer removes a server. Only the owner can delete.
func (s *Service) DeleteServer(ctx context.Context, serverID, userID string) error {
	srv, err := s.repo.GetServer(ctx, serverID)
	if err != nil {
		return err
	}
	if srv == nil {
		return fmt.Errorf("server not found")
	}
	if srv.OwnerID != userID {
		return fmt.Errorf("only the server owner can delete the server")
	}

	return s.repo.DeleteServer(ctx, serverID)
}

// --- Channels ---

// CreateChannel creates a new channel. Requires PermManageChannels.
func (s *Service) CreateChannel(ctx context.Context, serverID, userID, name, chType string) (*Channel, error) {
	if err := s.requirePermission(ctx, serverID, userID, PermManageChannels); err != nil {
		return nil, err
	}

	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("channel name cannot be empty")
	}
	if chType != "text" && chType != "voice" {
		return nil, fmt.Errorf("channel type must be 'text' or 'voice'")
	}

	ch := &Channel{
		ID:       uuid.New().String(),
		ServerID: serverID,
		Name:     strings.TrimSpace(name),
		Type:     chType,
	}

	if err := s.repo.CreateChannel(ctx, ch); err != nil {
		return nil, err
	}

	return ch, nil
}

// ListChannels returns all channels for a server.
func (s *Service) ListChannels(ctx context.Context, serverID string) ([]*Channel, error) {
	return s.repo.ListChannels(ctx, serverID)
}

// UpdateChannel updates a channel. Requires PermManageChannels.
func (s *Service) UpdateChannel(ctx context.Context, serverID, userID, channelID, name, chType string, position int) error {
	if err := s.requirePermission(ctx, serverID, userID, PermManageChannels); err != nil {
		return err
	}
	return s.repo.UpdateChannel(ctx, channelID, name, chType, position)
}

// DeleteChannel removes a channel. Requires PermManageChannels.
func (s *Service) DeleteChannel(ctx context.Context, serverID, userID, channelID string) error {
	if err := s.requirePermission(ctx, serverID, userID, PermManageChannels); err != nil {
		return err
	}
	return s.repo.DeleteChannel(ctx, channelID)
}

// --- Members ---

// ListMembers returns all members of a server.
func (s *Service) ListMembers(ctx context.Context, serverID string) ([]*Member, error) {
	return s.repo.ListMembers(ctx, serverID)
}

// KickMember removes a member from a server. Requires PermManageMembers.
// Cannot kick someone with a higher or equal role.
func (s *Service) KickMember(ctx context.Context, serverID, actorID, targetID string) error {
	if err := s.requirePermission(ctx, serverID, actorID, PermManageMembers); err != nil {
		return err
	}

	actor, err := s.repo.GetMember(ctx, serverID, actorID)
	if err != nil || actor == nil {
		return fmt.Errorf("actor not found")
	}

	target, err := s.repo.GetMember(ctx, serverID, targetID)
	if err != nil || target == nil {
		return fmt.Errorf("target member not found")
	}

	if RoleHierarchy(actor.Role) <= RoleHierarchy(target.Role) {
		return fmt.Errorf("cannot kick a member with equal or higher role")
	}

	return s.repo.RemoveMember(ctx, serverID, targetID)
}

// UpdateMemberRole changes a member's role. Requires PermManageMembers.
// Cannot promote above your own role or demote someone with higher role.
func (s *Service) UpdateMemberRole(ctx context.Context, serverID, actorID, targetID string, newRole Role) error {
	if err := s.requirePermission(ctx, serverID, actorID, PermManageMembers); err != nil {
		return err
	}

	if newRole == RoleOwner {
		return fmt.Errorf("cannot assign owner role directly")
	}

	actor, err := s.repo.GetMember(ctx, serverID, actorID)
	if err != nil || actor == nil {
		return fmt.Errorf("actor not found")
	}

	target, err := s.repo.GetMember(ctx, serverID, targetID)
	if err != nil || target == nil {
		return fmt.Errorf("target member not found")
	}

	if RoleHierarchy(actor.Role) <= RoleHierarchy(target.Role) {
		return fmt.Errorf("cannot modify a member with equal or higher role")
	}

	if RoleHierarchy(newRole) >= RoleHierarchy(actor.Role) {
		return fmt.Errorf("cannot promote a member to your role or above")
	}

	return s.repo.UpdateMemberRole(ctx, serverID, targetID, newRole)
}

// --- Invites ---

// GenerateInvite creates a new invite code for a server. Requires PermCreateInvite.
func (s *Service) GenerateInvite(ctx context.Context, serverID, userID string) (string, error) {
	if err := s.requirePermission(ctx, serverID, userID, PermCreateInvite); err != nil {
		return "", err
	}

	code, err := generateInviteCode()
	if err != nil {
		return "", fmt.Errorf("failed to generate invite: %w", err)
	}

	if err := s.repo.UpdateInviteCode(ctx, serverID, code); err != nil {
		return "", err
	}

	s.logger.Info().Str("server_id", serverID).Str("code", code).Msg("invite code generated")
	return code, nil
}

// RedeemInvite adds a user to a server via invite code.
func (s *Service) RedeemInvite(ctx context.Context, code, userID string) (*Server, error) {
	srv, err := s.repo.GetServerByInvite(ctx, code)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, fmt.Errorf("invalid invite code")
	}

	// Check if already a member
	existing, err := s.repo.GetMember(ctx, srv.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return srv, nil // Already a member, return server
	}

	if err := s.repo.AddMember(ctx, srv.ID, userID, RoleMember); err != nil {
		return nil, fmt.Errorf("failed to join server: %w", err)
	}

	s.logger.Info().
		Str("server_id", srv.ID).
		Str("user_id", userID).
		Msg("user joined server via invite")

	return srv, nil
}

// GetInviteInfo returns info about a server from an invite code.
func (s *Service) GetInviteInfo(ctx context.Context, code string) (*InviteInfo, error) {
	srv, err := s.repo.GetServerByInvite(ctx, code)
	if err != nil {
		return nil, err
	}
	if srv == nil {
		return nil, fmt.Errorf("invalid invite code")
	}

	count, err := s.repo.CountMembers(ctx, srv.ID)
	if err != nil {
		return nil, err
	}

	return &InviteInfo{
		ServerID:    srv.ID,
		ServerName:  srv.Name,
		InviteCode:  code,
		MemberCount: count,
	}, nil
}

// --- Helpers ---

// requirePermission checks that a user has a specific permission in a server.
func (s *Service) requirePermission(ctx context.Context, serverID, userID string, perm Permission) error {
	member, err := s.repo.GetMember(ctx, serverID, userID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if member == nil {
		return fmt.Errorf("not a member of this server")
	}
	if !HasPermission(member.Role, perm) {
		return fmt.Errorf("insufficient permissions")
	}
	return nil
}

// generateInviteCode generates a random 8-character invite code.
// Complexity: O(1)
func generateInviteCode() (string, error) {
	b := make([]byte, 5) // 5 bytes = 8 base32 chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)), nil
}

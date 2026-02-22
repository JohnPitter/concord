package friends

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// Service orchestrates friend management operations.
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new friends service.
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("component", "friends_service").Logger(),
	}
}

// SendRequest sends a friend request from senderID to the user with the given username.
// Validates: not self, not already friends, no duplicate pending request, user exists.
// Complexity: O(1).
func (s *Service) SendRequest(ctx context.Context, senderID, receiverUsername string) error {
	receiverUsername = strings.TrimSpace(receiverUsername)
	if receiverUsername == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Look up receiver
	receiverID, _, _, err := s.repo.GetUserByUsername(ctx, receiverUsername)
	if err != nil {
		return fmt.Errorf("failed to look up user: %w", err)
	}
	if receiverID == "" {
		return fmt.Errorf("user '%s' not found", receiverUsername)
	}

	// Cannot add yourself
	if senderID == receiverID {
		return fmt.Errorf("you cannot send a friend request to yourself")
	}

	// Check if already friends
	areFriends, err := s.repo.AreFriends(ctx, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to check friendship: %w", err)
	}
	if areFriends {
		return fmt.Errorf("you are already friends with %s", receiverUsername)
	}

	// Check for existing pending/blocked request
	existing, err := s.repo.ExistingRequest(ctx, senderID, receiverID)
	if err != nil {
		return fmt.Errorf("failed to check existing request: %w", err)
	}
	if existing != nil {
		if existing.Status == StatusBlocked {
			return fmt.Errorf("cannot send request to this user")
		}
		return fmt.Errorf("friend request already pending")
	}

	_, err = s.repo.SendRequest(ctx, senderID, receiverID)
	return err
}

// GetPendingRequests returns all pending friend requests for a user.
func (s *Service) GetPendingRequests(ctx context.Context, userID string) ([]FriendRequestView, error) {
	return s.repo.GetPendingRequests(ctx, userID)
}

// AcceptRequest accepts a friend request. Only the receiver can accept.
func (s *Service) AcceptRequest(ctx context.Context, requestID, userID string) error {
	return s.repo.AcceptRequest(ctx, requestID, userID)
}

// RejectRequest rejects or cancels a friend request.
func (s *Service) RejectRequest(ctx context.Context, requestID, userID string) error {
	return s.repo.RejectRequest(ctx, requestID, userID)
}

// GetFriends returns all friends for a user.
func (s *Service) GetFriends(ctx context.Context, userID string) ([]FriendView, error) {
	return s.repo.GetFriends(ctx, userID)
}

// RemoveFriend removes a friendship.
func (s *Service) RemoveFriend(ctx context.Context, userID, friendID string) error {
	return s.repo.RemoveFriend(ctx, userID, friendID)
}

// BlockUser blocks a target user.
func (s *Service) BlockUser(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return fmt.Errorf("you cannot block yourself")
	}
	return s.repo.BlockUser(ctx, userID, targetID)
}

// UnblockUser unblocks a target user by username lookup.
func (s *Service) UnblockUser(ctx context.Context, userID, targetUsername string) error {
	targetID, _, _, err := s.repo.GetUserByUsername(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("failed to look up user: %w", err)
	}
	if targetID == "" {
		return fmt.Errorf("user '%s' not found", targetUsername)
	}
	return s.repo.UnblockUser(ctx, userID, targetID)
}

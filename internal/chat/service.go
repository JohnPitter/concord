package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	maxMessageLength = 4000
)

// Service orchestrates chat operations.
type Service struct {
	repo   *Repository
	logger zerolog.Logger
}

// NewService creates a new chat service.
func NewService(repo *Repository, logger zerolog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With().Str("component", "chat_service").Logger(),
	}
}

// SendMessage creates and stores a new message.
func (s *Service) SendMessage(ctx context.Context, channelID, authorID, content string) (*Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("message content cannot be empty")
	}
	if len(content) > maxMessageLength {
		return nil, fmt.Errorf("message exceeds maximum length of %d characters", maxMessageLength)
	}

	msg := &Message{
		ID:        uuid.New().String(),
		ChannelID: channelID,
		AuthorID:  authorID,
		Content:   content,
		Type:      "text",
	}

	if err := s.repo.Save(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Re-fetch to get joined author fields + created_at
	saved, err := s.repo.GetByID(ctx, msg.ID)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("message_id", msg.ID).
		Str("channel_id", channelID).
		Str("author_id", authorID).
		Msg("message sent")

	return saved, nil
}

// GetMessages retrieves messages for a channel with cursor-based pagination.
func (s *Service) GetMessages(ctx context.Context, channelID string, opts PaginationOpts) ([]*Message, error) {
	return s.repo.GetByChannel(ctx, channelID, opts)
}

// EditMessage updates the content of a message. Only the author can edit.
func (s *Service) EditMessage(ctx context.Context, messageID, authorID, content string) (*Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("message content cannot be empty")
	}
	if len(content) > maxMessageLength {
		return nil, fmt.Errorf("message exceeds maximum length of %d characters", maxMessageLength)
	}

	existing, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("message not found")
	}
	if existing.AuthorID != authorID {
		return nil, fmt.Errorf("only the author can edit a message")
	}

	if err := s.repo.Update(ctx, messageID, content); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("message_id", messageID).
		Str("author_id", authorID).
		Msg("message edited")

	return updated, nil
}

// DeleteMessage removes a message. The author or someone with PermManageMessages can delete.
// For simplicity in Phase 4, only the author can delete. Permission-based delete uses isManager flag.
func (s *Service) DeleteMessage(ctx context.Context, messageID, actorID string, isManager bool) error {
	existing, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("message not found")
	}

	if existing.AuthorID != actorID && !isManager {
		return fmt.Errorf("insufficient permissions to delete this message")
	}

	if err := s.repo.Delete(ctx, messageID); err != nil {
		return err
	}

	s.logger.Info().
		Str("message_id", messageID).
		Str("actor_id", actorID).
		Bool("is_manager", isManager).
		Msg("message deleted")

	return nil
}

// SearchMessages performs FTS5 search within a channel.
func (s *Service) SearchMessages(ctx context.Context, channelID, query string, limit int) ([]*SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	return s.repo.Search(ctx, channelID, query, limit)
}

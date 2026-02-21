package chat

import (
	"sync"

	"github.com/rs/zerolog"
)

// MessageQueue is a simple in-memory queue for offline message delivery.
// Each user has an independent slice of pending messages.
// Thread-safe via sync.RWMutex.
type MessageQueue struct {
	mu     sync.RWMutex
	queue  map[string][]*Message // userID -> pending messages
	logger zerolog.Logger
}

// NewMessageQueue creates a new empty message queue.
// Complexity: O(1)
func NewMessageQueue(logger zerolog.Logger) *MessageQueue {
	return &MessageQueue{
		queue:  make(map[string][]*Message),
		logger: logger.With().Str("component", "message_queue").Logger(),
	}
}

// Enqueue appends a message to a user's pending queue.
// Complexity: O(1) amortized (slice append)
func (q *MessageQueue) Enqueue(userID string, msg *Message) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.queue[userID] = append(q.queue[userID], msg)

	q.logger.Debug().
		Str("user_id", userID).
		Str("message_id", msg.ID).
		Int("queue_size", len(q.queue[userID])).
		Msg("message enqueued for offline delivery")
}

// Drain removes and returns all pending messages for a user.
// Returns nil if the user has no pending messages.
// Complexity: O(1) — swap and nil the slice
func (q *MessageQueue) Drain(userID string) []*Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	messages := q.queue[userID]
	if len(messages) == 0 {
		return nil
	}

	delete(q.queue, userID)

	q.logger.Info().
		Str("user_id", userID).
		Int("count", len(messages)).
		Msg("drained pending messages")

	return messages
}

// Pending returns the number of pending messages for a user.
// Complexity: O(1)
func (q *MessageQueue) Pending(userID string) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.queue[userID])
}

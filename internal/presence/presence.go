package presence

import (
	"sync"
	"time"
)

// Tracker keeps an in-memory record of recent user activity.
// It is used to derive near-real-time online status.
type Tracker struct {
	mu   sync.RWMutex
	seen map[string]time.Time
	ttl  time.Duration
}

// NewTracker creates a presence tracker with the given online TTL.
func NewTracker(ttl time.Duration) *Tracker {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	return &Tracker{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
}

// Touch marks a user as active "now".
func (t *Tracker) Touch(userID string) {
	if userID == "" {
		return
	}
	t.mu.Lock()
	t.seen[userID] = time.Now().UTC()
	t.mu.Unlock()
}

// IsOnline reports whether the user has been active within the TTL window.
func (t *Tracker) IsOnline(userID string) bool {
	if userID == "" {
		return false
	}
	t.mu.RLock()
	seenAt, ok := t.seen[userID]
	ttl := t.ttl
	t.mu.RUnlock()
	if !ok {
		return false
	}
	return time.Since(seenAt) <= ttl
}

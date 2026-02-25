package presence

import (
	"sync"
	"time"
)

// Tracker keeps an in-memory record of recent user activity.
// It is used to derive near-real-time online status.
// A background reaper goroutine periodically removes entries that have
// exceeded the TTL, preventing unbounded memory growth and ensuring
// users that close the app without a clean offline signal still appear
// offline after the TTL expires.
type Tracker struct {
	mu   sync.RWMutex
	seen map[string]time.Time
	ttl  time.Duration
	stop chan struct{}
}

// NewTracker creates a presence tracker with the given online TTL.
// It starts a background reaper goroutine that runs every ttl/2 to
// evict stale entries.
func NewTracker(ttl time.Duration) *Tracker {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	t := &Tracker{
		seen: make(map[string]time.Time),
		ttl:  ttl,
		stop: make(chan struct{}),
	}
	go t.reapLoop()
	return t
}

// reapLoop periodically removes stale presence entries.
func (t *Tracker) reapLoop() {
	interval := t.ttl / 2
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.evictStale()
		case <-t.stop:
			return
		}
	}
}

// evictStale removes all entries older than the TTL.
func (t *Tracker) evictStale() {
	cutoff := time.Now().UTC().Add(-t.ttl)
	t.mu.Lock()
	for uid, seenAt := range t.seen {
		if seenAt.Before(cutoff) {
			delete(t.seen, uid)
		}
	}
	t.mu.Unlock()
}

// Stop terminates the background reaper goroutine.
// Safe to call multiple times.
func (t *Tracker) Stop() {
	select {
	case <-t.stop:
		// already stopped
	default:
		close(t.stop)
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

// SetOffline removes the user from the active set immediately.
func (t *Tracker) SetOffline(userID string) {
	if userID == "" {
		return
	}
	t.mu.Lock()
	delete(t.seen, userID)
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

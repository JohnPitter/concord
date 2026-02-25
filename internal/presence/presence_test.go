package presence

import (
	"testing"
	"time"
)

func TestTrackerSetOffline(t *testing.T) {
	tracker := NewTracker(30 * time.Second)
	defer tracker.Stop()
	userID := "user-1"

	tracker.Touch(userID)
	if !tracker.IsOnline(userID) {
		t.Fatalf("expected user to be online after touch")
	}

	tracker.SetOffline(userID)
	if tracker.IsOnline(userID) {
		t.Fatalf("expected user to be offline after SetOffline")
	}
}

func TestTrackerReaperEvictsStaleEntries(t *testing.T) {
	// TTL must be >= 2s so reaper interval (ttl/2) >= 1s (the enforced minimum).
	tracker := NewTracker(2 * time.Second)
	defer tracker.Stop()

	tracker.Touch("user-a")
	tracker.Touch("user-b")

	if !tracker.IsOnline("user-a") || !tracker.IsOnline("user-b") {
		t.Fatal("expected both users to be online right after touch")
	}

	// Wait for TTL to expire (2s) + reaper cycle (1s) + margin.
	time.Sleep(3500 * time.Millisecond)

	if tracker.IsOnline("user-a") {
		t.Error("expected user-a to be offline after TTL expired")
	}
	if tracker.IsOnline("user-b") {
		t.Error("expected user-b to be offline after TTL expired")
	}

	// Verify the map was actually cleaned up (not just TTL-expired).
	tracker.mu.RLock()
	count := len(tracker.seen)
	tracker.mu.RUnlock()
	if count != 0 {
		t.Errorf("expected 0 entries in seen map after reaper, got %d", count)
	}
}

func TestTrackerStop(t *testing.T) {
	tracker := NewTracker(1 * time.Second)
	// Stop should not panic even when called multiple times.
	tracker.Stop()
	tracker.Stop()
}

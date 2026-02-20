package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
// This prevents brute force attacks and DDoS attempts
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*bucket
	rate     int           // tokens per interval
	interval time.Duration // time interval
	capacity int           // maximum tokens in bucket
	ttl      time.Duration // time to live for inactive buckets
}

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens    int
	lastCheck time.Time
	mu        sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed per interval
// interval: time window for rate limiting
// capacity: maximum burst size
// Complexity: O(1)
func NewRateLimiter(rate int, interval time.Duration, capacity int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		capacity: capacity,
		ttl:      1 * time.Hour, // Clean up inactive buckets after 1 hour
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given key should be allowed
// Returns true if request is allowed, false if rate limit exceeded
// Complexity: O(1)
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		// Create new bucket
		b = &bucket{
			tokens:    rl.capacity,
			lastCheck: time.Now(),
		}

		rl.mu.Lock()
		rl.buckets[key] = b
		rl.mu.Unlock()
	}

	return b.take(rl.rate, rl.interval, rl.capacity)
}

// AllowN checks if N requests from the given key should be allowed
// Complexity: O(1)
func (rl *RateLimiter) AllowN(key string, n int) bool {
	if n <= 0 {
		return true
	}

	rl.mu.RLock()
	b, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if !exists {
		b = &bucket{
			tokens:    rl.capacity,
			lastCheck: time.Now(),
		}

		rl.mu.Lock()
		rl.buckets[key] = b
		rl.mu.Unlock()
	}

	return b.takeN(n, rl.rate, rl.interval, rl.capacity)
}

// Reset resets the rate limit for a specific key
// Complexity: O(1)
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.buckets, key)
}

// take attempts to take one token from the bucket
func (b *bucket) take(rate int, interval time.Duration, capacity int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastCheck)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Nanoseconds() * int64(rate) / interval.Nanoseconds())
	b.tokens += tokensToAdd

	if b.tokens > capacity {
		b.tokens = capacity
	}

	b.lastCheck = now

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// takeN attempts to take N tokens from the bucket
func (b *bucket) takeN(n, rate int, interval time.Duration, capacity int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastCheck)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Nanoseconds() * int64(rate) / interval.Nanoseconds())
	b.tokens += tokensToAdd

	if b.tokens > capacity {
		b.tokens = capacity
	}

	b.lastCheck = now

	// Check if we have enough tokens
	if b.tokens >= n {
		b.tokens -= n
		return true
	}

	return false
}

// cleanup periodically removes inactive buckets to prevent memory leaks
// Complexity: O(n) where n is the number of buckets
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()

		now := time.Now()
		for key, b := range rl.buckets {
			b.mu.Lock()
			if now.Sub(b.lastCheck) > rl.ttl {
				delete(rl.buckets, key)
			}
			b.mu.Unlock()
		}

		rl.mu.Unlock()
	}
}

// BruteForceProtector protects against brute force attacks
// Uses exponential backoff for repeated failures
type BruteForceProtector struct {
	mu            sync.RWMutex
	attempts      map[string]*attemptTracker
	maxAttempts   int
	lockoutPeriod time.Duration
	ttl           time.Duration
}

// attemptTracker tracks failed attempts for a specific key
type attemptTracker struct {
	count       int
	firstAttempt time.Time
	lockUntil   time.Time
	mu          sync.Mutex
}

// NewBruteForceProtector creates a new brute force protector
// maxAttempts: maximum failed attempts before lockout
// lockoutPeriod: how long to lock out after max attempts
func NewBruteForceProtector(maxAttempts int, lockoutPeriod time.Duration) *BruteForceProtector {
	bfp := &BruteForceProtector{
		attempts:      make(map[string]*attemptTracker),
		maxAttempts:   maxAttempts,
		lockoutPeriod: lockoutPeriod,
		ttl:           24 * time.Hour,
	}

	// Start cleanup goroutine
	go bfp.cleanup()

	return bfp
}

// RecordFailure records a failed attempt
// Complexity: O(1)
func (bfp *BruteForceProtector) RecordFailure(key string) {
	bfp.mu.RLock()
	tracker, exists := bfp.attempts[key]
	bfp.mu.RUnlock()

	if !exists {
		tracker = &attemptTracker{
			count:        0,
			firstAttempt: time.Now(),
		}

		bfp.mu.Lock()
		bfp.attempts[key] = tracker
		bfp.mu.Unlock()
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	tracker.count++

	// Implement exponential backoff
	if tracker.count >= bfp.maxAttempts {
		lockDuration := bfp.lockoutPeriod * time.Duration(1<<uint(tracker.count-bfp.maxAttempts))
		if lockDuration > 24*time.Hour {
			lockDuration = 24 * time.Hour // Cap at 24 hours
		}
		tracker.lockUntil = time.Now().Add(lockDuration)
	}
}

// RecordSuccess records a successful attempt and resets the counter
// Complexity: O(1)
func (bfp *BruteForceProtector) RecordSuccess(key string) {
	bfp.mu.Lock()
	defer bfp.mu.Unlock()

	delete(bfp.attempts, key)
}

// IsAllowed checks if an attempt should be allowed
// Returns (allowed bool, retryAfter time.Duration, error)
// Complexity: O(1)
func (bfp *BruteForceProtector) IsAllowed(key string) (bool, time.Duration, error) {
	bfp.mu.RLock()
	tracker, exists := bfp.attempts[key]
	bfp.mu.RUnlock()

	if !exists {
		return true, 0, nil
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := time.Now()

	// Check if still locked out
	if now.Before(tracker.lockUntil) {
		retryAfter := tracker.lockUntil.Sub(now)
		return false, retryAfter, fmt.Errorf("too many failed attempts, try again in %v", retryAfter.Round(time.Second))
	}

	// Reset if lockout period has passed
	if now.After(tracker.lockUntil) && tracker.count >= bfp.maxAttempts {
		tracker.count = 0
		tracker.firstAttempt = now
		tracker.lockUntil = time.Time{}
	}

	return true, 0, nil
}

// GetAttempts returns the number of failed attempts for a key
// Complexity: O(1)
func (bfp *BruteForceProtector) GetAttempts(key string) int {
	bfp.mu.RLock()
	defer bfp.mu.RUnlock()

	tracker, exists := bfp.attempts[key]
	if !exists {
		return 0
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	return tracker.count
}

// cleanup periodically removes old attempt trackers
func (bfp *BruteForceProtector) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		bfp.mu.Lock()

		now := time.Now()
		for key, tracker := range bfp.attempts {
			tracker.mu.Lock()
			if now.Sub(tracker.firstAttempt) > bfp.ttl {
				delete(bfp.attempts, key)
			}
			tracker.mu.Unlock()
		}

		bfp.mu.Unlock()
	}
}

// WaitIfNeeded waits if rate limit is exceeded
// Returns context error if context is canceled
// Complexity: O(1)
func (rl *RateLimiter) WaitIfNeeded(ctx context.Context, key string) error {
	for {
		if rl.Allow(key) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rl.interval / time.Duration(rl.rate)):
			// Wait for next token
		}
	}
}

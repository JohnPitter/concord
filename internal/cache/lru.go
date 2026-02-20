package cache

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// entry holds a cached value with its expiration time.
type entry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// LRU is a thread-safe LRU cache with per-key TTL.
// Complexity: Get O(1), Set O(1), Delete O(1), DeletePrefix O(n).
type LRU struct {
	mu       sync.RWMutex
	maxSize  int
	items    map[string]*list.Element
	eviction *list.List
}

// NewLRU creates a new LRU cache with the given maximum number of entries.
func NewLRU(maxSize int) *LRU {
	if maxSize <= 0 {
		maxSize = 1024
	}
	return &LRU{
		maxSize:  maxSize,
		items:    make(map[string]*list.Element, maxSize),
		eviction: list.New(),
	}
}

// Get retrieves a value by key. Returns nil and false if not found or expired.
// Complexity: O(1)
func (c *LRU) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, false
	}

	e := el.Value.(*entry)
	if time.Now().After(e.expiresAt) {
		c.removeLocked(el)
		return nil, false
	}

	// Move to front (most recently used)
	c.eviction.MoveToFront(el)
	return e.value, true
}

// Set adds or updates a key with a value and TTL.
// Complexity: O(1)
func (c *LRU) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.eviction.MoveToFront(el)
		e := el.Value.(*entry)
		e.value = value
		e.expiresAt = time.Now().Add(ttl)
		return
	}

	// Evict LRU if at capacity
	if c.eviction.Len() >= c.maxSize {
		c.removeLocked(c.eviction.Back())
	}

	e := &entry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	el := c.eviction.PushFront(e)
	c.items[key] = el
}

// Delete removes a key from the cache.
// Complexity: O(1)
func (c *LRU) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.removeLocked(el)
	}
}

// DeletePrefix removes all keys that start with the given prefix.
// Useful for group invalidation (e.g. all channels of a server).
// Complexity: O(n) where n = total cache entries
func (c *LRU) DeletePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, el := range c.items {
		if strings.HasPrefix(key, prefix) {
			c.removeLocked(el)
		}
	}
}

// Len returns the number of entries in the cache.
func (c *LRU) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.eviction.Len()
}

// removeLocked removes an element from both the list and map.
// Must be called with mu held.
func (c *LRU) removeLocked(el *list.Element) {
	c.eviction.Remove(el)
	e := el.Value.(*entry)
	delete(c.items, e.key)
}

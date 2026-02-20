package translation

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/concord-chat/concord/internal/cache"
)

const (
	// cacheTTL is the default time-to-live for translation cache entries.
	cacheTTL = 1 * time.Hour
)

// TranslationCache wraps an LRU cache for translated text lookups.
// Cache key format: translate:{sourceLang}:{targetLang}:{sha256(text)}
// Complexity: Get O(1), Set O(1)
type TranslationCache struct {
	lru *cache.LRU
}

// NewTranslationCache creates a new translation cache with the given maximum size.
func NewTranslationCache(maxSize int) *TranslationCache {
	return &TranslationCache{
		lru: cache.NewLRU(maxSize),
	}
}

// Get retrieves a cached translation result.
// Returns the translated text and true if found; empty string and false otherwise.
// Complexity: O(1)
func (tc *TranslationCache) Get(sourceLang, targetLang, text string) (string, bool) {
	key := buildCacheKey(sourceLang, targetLang, text)
	val, ok := tc.lru.Get(key)
	if !ok {
		return "", false
	}
	result, ok := val.(string)
	if !ok {
		return "", false
	}
	return result, true
}

// Set stores a translation result in the cache.
// Complexity: O(1)
func (tc *TranslationCache) Set(sourceLang, targetLang, text, result string) {
	key := buildCacheKey(sourceLang, targetLang, text)
	tc.lru.Set(key, result, cacheTTL)
}

// Len returns the number of entries in the cache.
func (tc *TranslationCache) Len() int {
	return tc.lru.Len()
}

// buildCacheKey generates the cache key using the format:
// translate:{src}:{tgt}:{sha256(text)}
// Complexity: O(n) where n = len(text) for hashing
func buildCacheKey(sourceLang, targetLang, text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("translate:%s:%s:%x", sourceLang, targetLang, hash)
}

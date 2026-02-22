package translation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/concord-chat/concord/internal/config"
	"github.com/rs/zerolog"
)

// Errors returned by the translation service.
var (
	ErrTranslationDisabled = errors.New("translation: service is disabled")
	ErrAlreadyEnabled      = errors.New("translation: already enabled")
	ErrNotEnabled          = errors.New("translation: not enabled")
)

// Status represents the current state of the translation service.
type Status struct {
	Enabled            bool   `json:"enabled"`
	SourceLang         string `json:"source_lang"`
	TargetLang         string `json:"target_lang"`
	CircuitBreakerOpen bool   `json:"circuit_breaker_open"`
	CacheEntries       int    `json:"cache_entries"`
}

// Service wraps the LibreTranslate Client and TranslationCache
// to provide a unified translation API for Wails bindings.
type Service struct {
	mu         sync.RWMutex
	client     *Client
	cache      *TranslationCache
	cfg        config.TranslationConfig
	logger     zerolog.Logger
	enabled    bool
	sourceLang string
	targetLang string
}

// NewService creates a new translation service.
// Complexity: O(1)
func NewService(cfg config.TranslationConfig, logger zerolog.Logger) *Service {
	client := NewClient(cfg, logger)

	var translationCache *TranslationCache
	if cfg.CacheEnabled {
		translationCache = NewTranslationCache(cfg.CacheSize)
	}

	return &Service{
		client: client,
		cache:  translationCache,
		cfg:    cfg,
		logger: logger.With().Str("component", "translation-service").Logger(),
	}
}

// Enable activates translation between two languages.
// Complexity: O(1)
func (s *Service) Enable(sourceLang, targetLang string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.enabled {
		return ErrAlreadyEnabled
	}

	if sourceLang == "" || targetLang == "" {
		return fmt.Errorf("translation: source and target languages are required")
	}

	s.enabled = true
	s.sourceLang = sourceLang
	s.targetLang = targetLang

	s.logger.Info().
		Str("source_lang", sourceLang).
		Str("target_lang", targetLang).
		Msg("translation enabled")

	return nil
}

// Disable deactivates translation.
// Complexity: O(1)
func (s *Service) Disable() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return ErrNotEnabled
	}

	s.enabled = false
	s.sourceLang = ""
	s.targetLang = ""

	s.logger.Info().Msg("translation disabled")
	return nil
}

// GetStatus returns the current status of the translation service.
// Complexity: O(1)
func (s *Service) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := Status{
		Enabled:            s.enabled,
		SourceLang:         s.sourceLang,
		TargetLang:         s.targetLang,
		CircuitBreakerOpen: s.client.IsCircuitOpen(),
	}

	if s.cache != nil {
		st.CacheEntries = s.cache.Len()
	}

	return st
}

// TranslateText translates text using the LibreTranslate API with optional caching.
// If caching is enabled and a cached result exists, returns it without calling the API.
// Complexity: O(1) for cache hit, O(1) + network for cache miss
func (s *Service) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	s.mu.RLock()
	enabled := s.enabled
	s.mu.RUnlock()

	if !enabled {
		return "", ErrTranslationDisabled
	}

	// Check cache
	if s.cache != nil {
		if result, ok := s.cache.Get(sourceLang, targetLang, text); ok {
			s.logger.Debug().
				Str("source_lang", sourceLang).
				Str("target_lang", targetLang).
				Msg("translation cache hit")
			return result, nil
		}
	}

	// Call LibreTranslate API
	result, err := s.client.TranslateText(ctx, text, sourceLang, targetLang)
	if err != nil {
		return "", err
	}

	// Store in cache
	if s.cache != nil {
		s.cache.Set(sourceLang, targetLang, text, result)
	}

	return result, nil
}

// TranslateTextDirect translates text without requiring the service to be enabled.
// Used by the frontend for on-demand message translation (always available).
// Complexity: O(1) for cache hit, O(1) + network for cache miss
func (s *Service) TranslateTextDirect(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	// Check cache
	if s.cache != nil {
		if result, ok := s.cache.Get(sourceLang, targetLang, text); ok {
			s.logger.Debug().
				Str("source_lang", sourceLang).
				Str("target_lang", targetLang).
				Msg("translation cache hit (direct)")
			return result, nil
		}
	}

	// Call LibreTranslate API
	result, err := s.client.TranslateText(ctx, text, sourceLang, targetLang)
	if err != nil {
		return "", err
	}

	// Store in cache
	if s.cache != nil {
		s.cache.Set(sourceLang, targetLang, text, result)
	}

	return result, nil
}

// ResetCircuitBreaker resets the circuit breaker to allow new requests.
// Complexity: O(1)
func (s *Service) ResetCircuitBreaker() {
	s.client.ResetCircuit()
}

// Client returns the underlying LibreTranslate client (for advanced usage).
func (s *Service) Client() *Client {
	return s.client
}

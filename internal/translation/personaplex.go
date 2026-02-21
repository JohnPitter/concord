package translation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/concord-chat/concord/internal/config"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

// circuitState represents the state of the circuit breaker.
type circuitState int

const (
	circuitClosed circuitState = iota // Normal operation
	circuitOpen                       // Requests blocked, waiting for reset
)

// translateRequest is the HTTP request body for the PersonaPlex translation API.
type translateRequest struct {
	Text       string `json:"text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

// translateResponse is the HTTP response body from the PersonaPlex translation API.
type translateResponse struct {
	TranslatedText string `json:"translated_text"`
	SourceLang     string `json:"source_lang"`
	TargetLang     string `json:"target_lang"`
}

// Client provides HTTP and WebSocket access to the NVIDIA PersonaPlex translation API.
// It includes a circuit breaker that auto-disables when latency exceeds MaxLatency
// for consecutive FailureThreshold calls.
type Client struct {
	mu               sync.RWMutex
	cfg              config.TranslationConfig
	httpClient       *http.Client
	logger           zerolog.Logger
	consecutiveFails int
	state            circuitState
	lastFailure      time.Time
}

// NewClient creates a new PersonaPlex API client.
// Complexity: O(1)
func NewClient(cfg config.TranslationConfig, logger zerolog.Logger) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger.With().Str("component", "personaplex-client").Logger(),
		state:  circuitClosed,
	}
}

// TranslateText performs a synchronous HTTP-based text translation.
// Returns the translated text or an error.
// If the circuit breaker is open, returns an error immediately.
// Complexity: O(1) excluding network I/O
func (c *Client) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if err := c.checkCircuit(); err != nil {
		return "", err
	}

	start := time.Now()

	reqBody := translateRequest{
		Text:       text,
		SourceLang: sourceLang,
		TargetLang: targetLang,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("translation: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/translate", c.cfg.PersonaPlexURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("translation: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordFailure()
		return "", fmt.Errorf("translation: http request: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		c.recordFailure()
		return "", fmt.Errorf("translation: API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.recordFailure()
		return "", fmt.Errorf("translation: decode response: %w", err)
	}

	// Check latency and update circuit breaker
	c.recordLatency(latency)

	c.logger.Debug().
		Str("source_lang", sourceLang).
		Str("target_lang", targetLang).
		Dur("latency_ms", latency).
		Int("text_len", len(text)).
		Msg("translation completed")

	return result.TranslatedText, nil
}

// TranslateStream opens a WebSocket connection for streaming audio translation.
// Reads audio frames from audioIn, sends to PersonaPlex, returns translated audio on output channel.
// If the circuit breaker is open, returns an error immediately.
// Complexity: O(n) where n = number of audio frames streamed
func (c *Client) TranslateStream(ctx context.Context, audioIn <-chan []byte, sourceLang, targetLang string) (<-chan []byte, error) {
	if err := c.checkCircuit(); err != nil {
		return nil, err
	}

	wsURL := fmt.Sprintf("%s/translate/stream?source_lang=%s&target_lang=%s",
		c.cfg.PersonaPlexURL, sourceLang, targetLang)

	header := http.Header{}
	if c.cfg.APIKey != "" {
		header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		c.recordFailure()
		return nil, fmt.Errorf("translation: websocket dial: %w", err)
	}

	out := make(chan []byte, 64)

	// Writer goroutine: reads from audioIn and sends to WebSocket
	go func() {
		defer conn.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-audioIn:
				if !ok {
					// Input channel closed, send close message
					conn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
				start := time.Now()
				if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
					c.logger.Warn().Err(err).Msg("translation: websocket write failed")
					c.recordFailure()
					return
				}
				c.recordLatency(time.Since(start))
			}
		}
	}()

	// Reader goroutine: reads translated audio from WebSocket and sends to output
	go func() {
		defer close(out)
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					c.logger.Warn().Err(err).Msg("translation: websocket read failed")
				}
				return
			}
			if msgType == websocket.BinaryMessage {
				select {
				case out <- data:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	c.logger.Info().
		Str("source_lang", sourceLang).
		Str("target_lang", targetLang).
		Msg("translation stream started")

	return out, nil
}

// checkCircuit returns an error if the circuit breaker is open.
// Complexity: O(1)
func (c *Client) checkCircuit() error {
	if !c.cfg.CircuitBreaker {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.state == circuitOpen {
		return fmt.Errorf("translation: circuit breaker open — service auto-disabled after %d consecutive high-latency failures", c.cfg.FailureThreshold)
	}
	return nil
}

// recordFailure increments the failure counter and may trip the circuit breaker.
// Complexity: O(1)
func (c *Client) recordFailure() {
	if !c.cfg.CircuitBreaker {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.consecutiveFails++
	c.lastFailure = time.Now()

	if c.consecutiveFails >= c.cfg.FailureThreshold {
		c.state = circuitOpen
		c.logger.Warn().
			Int("consecutive_failures", c.consecutiveFails).
			Int("threshold", c.cfg.FailureThreshold).
			Msg("translation: circuit breaker opened — auto-disabling translation service")
	}
}

// recordLatency checks if the response latency exceeds MaxLatency.
// If it does, it increments the failure counter; otherwise resets it.
// Complexity: O(1)
func (c *Client) recordLatency(latency time.Duration) {
	if !c.cfg.CircuitBreaker {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if latency > c.cfg.MaxLatency {
		c.consecutiveFails++
		c.logger.Warn().
			Dur("latency", latency).
			Dur("max_latency", c.cfg.MaxLatency).
			Int("consecutive_failures", c.consecutiveFails).
			Msg("translation: latency exceeded threshold")

		if c.consecutiveFails >= c.cfg.FailureThreshold {
			c.state = circuitOpen
			c.logger.Warn().
				Int("consecutive_failures", c.consecutiveFails).
				Int("threshold", c.cfg.FailureThreshold).
				Msg("translation: circuit breaker opened — auto-disabling translation service")
		}
	} else {
		// Reset on successful low-latency response
		c.consecutiveFails = 0
	}
}

// ResetCircuit manually resets the circuit breaker to closed state.
// Complexity: O(1)
func (c *Client) ResetCircuit() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.state = circuitClosed
	c.consecutiveFails = 0
	c.logger.Info().Msg("translation: circuit breaker reset")
}

// IsCircuitOpen returns true if the circuit breaker is currently open.
// Complexity: O(1)
func (c *Client) IsCircuitOpen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == circuitOpen
}

// ConsecutiveFailures returns the current count of consecutive failures.
// Complexity: O(1)
func (c *Client) ConsecutiveFailures() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.consecutiveFails
}

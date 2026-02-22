package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// TTSClient is an HTTP client for OpenAI-compatible TTS APIs.
type TTSClient struct {
	httpClient *http.Client
	apiURL     string // e.g. "https://api.openai.com/v1/audio/speech"
	apiKey     string
	voice      string // e.g. "alloy", "echo", "nova"
	format     string // e.g. "mp3", "wav"
	logger     zerolog.Logger
}

// TTSConfig holds configuration for the TTS client.
type TTSConfig struct {
	APIURL  string
	APIKey  string
	Voice   string
	Format  string
	Timeout time.Duration
}

// ttsRequest is the JSON request body for the TTS API.
type ttsRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format"`
}

// NewTTSClient creates a new OpenAI-compatible TTS client.
func NewTTSClient(cfg TTSConfig, logger zerolog.Logger) *TTSClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	voice := cfg.Voice
	if voice == "" {
		voice = "alloy"
	}

	format := cfg.Format
	if format == "" {
		format = "mp3"
	}

	return &TTSClient{
		httpClient: &http.Client{Timeout: timeout},
		apiURL:     cfg.APIURL,
		apiKey:     cfg.APIKey,
		voice:      voice,
		format:     format,
		logger:     logger.With().Str("component", "tts-client").Logger(),
	}
}

// Synthesize converts text to audio using the TTS API.
// Returns the raw audio bytes in the configured format (MP3/WAV).
func (c *TTSClient) Synthesize(ctx context.Context, text, lang string) ([]byte, error) {
	start := time.Now()

	reqBody := ttsRequest{
		Model:          "tts-1",
		Input:          text,
		Voice:          c.voice,
		ResponseFormat: c.format,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("tts: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("tts: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: http request: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("tts: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts: read response body: %w", err)
	}

	c.logger.Debug().
		Dur("latency", latency).
		Str("lang", lang).
		Int("text_len", len(text)).
		Int("audio_bytes", len(audioData)).
		Msg("speech synthesis completed")

	return audioData, nil
}

package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// STTClient is an HTTP client for OpenAI Whisper-compatible STT APIs.
type STTClient struct {
	httpClient *http.Client
	apiURL     string // e.g. "https://api.openai.com/v1/audio/transcriptions"
	apiKey     string
	model      string // e.g. "whisper-1"
	logger     zerolog.Logger
}

// STTResult holds the transcription result from the STT API.
type STTResult struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

// STTConfig holds configuration for the STT client.
type STTConfig struct {
	APIURL  string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// NewSTTClient creates a new Whisper-compatible STT client.
func NewSTTClient(cfg STTConfig, logger zerolog.Logger) *STTClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &STTClient{
		httpClient: &http.Client{Timeout: timeout},
		apiURL:     cfg.APIURL,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		logger:     logger.With().Str("component", "stt-client").Logger(),
	}
}

// Transcribe sends OGG audio to the Whisper API and returns the transcribed text.
func (c *STTClient) Transcribe(ctx context.Context, audioOGG []byte, lang string) (*STTResult, error) {
	start := time.Now()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add audio file
	part, err := writer.CreateFormFile("file", "audio.ogg")
	if err != nil {
		return nil, fmt.Errorf("stt: create form file: %w", err)
	}
	if _, err := part.Write(audioOGG); err != nil {
		return nil, fmt.Errorf("stt: write audio data: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", c.model); err != nil {
		return nil, fmt.Errorf("stt: write model field: %w", err)
	}

	// Add language hint if provided
	if lang != "" {
		if err := writer.WriteField("language", lang); err != nil {
			return nil, fmt.Errorf("stt: write language field: %w", err)
		}
	}

	// Add response format
	if err := writer.WriteField("response_format", "json"); err != nil {
		return nil, fmt.Errorf("stt: write format field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("stt: close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, &body)
	if err != nil {
		return nil, fmt.Errorf("stt: create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stt: http request: %w", err)
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("stt: API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result STTResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("stt: decode response: %w", err)
	}

	c.logger.Debug().
		Dur("latency", latency).
		Str("language", lang).
		Int("audio_bytes", len(audioOGG)).
		Int("text_len", len(result.Text)).
		Msg("transcription completed")

	return &result, nil
}

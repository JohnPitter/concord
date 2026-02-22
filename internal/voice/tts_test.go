package voice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTS_Synthesize_Success(t *testing.T) {
	fakeAudio := []byte("fake-mp3-audio-data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req ttsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "tts-1", req.Model)
		assert.Equal(t, "Hello world", req.Input)
		assert.Equal(t, "alloy", req.Voice)
		assert.Equal(t, "mp3", req.ResponseFormat)

		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeAudio)
	}))
	defer server.Close()

	client := NewTTSClient(TTSConfig{
		APIURL:  server.URL,
		APIKey:  "test-key",
		Voice:   "alloy",
		Format:  "mp3",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	result, err := client.Synthesize(context.Background(), "Hello world", "en")
	require.NoError(t, err)
	assert.Equal(t, fakeAudio, result)
}

func TestTTS_Synthesize_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"synthesis failed"}`))
	}))
	defer server.Close()

	client := NewTTSClient(TTSConfig{
		APIURL:  server.URL,
		APIKey:  "test-key",
		Voice:   "alloy",
		Format:  "mp3",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	result, err := client.Synthesize(context.Background(), "Hello", "en")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 500")
}

func TestTTS_Synthesize_DefaultVoiceAndFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ttsRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "alloy", req.Voice)
		assert.Equal(t, "mp3", req.ResponseFormat)

		w.Write([]byte("audio"))
	}))
	defer server.Close()

	// Empty voice and format should use defaults
	client := NewTTSClient(TTSConfig{
		APIURL:  server.URL,
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	result, err := client.Synthesize(context.Background(), "test", "en")
	require.NoError(t, err)
	assert.Equal(t, []byte("audio"), result)
}

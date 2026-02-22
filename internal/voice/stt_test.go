package voice

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSTT_Transcribe_Success(t *testing.T) {
	expected := STTResult{
		Text:     "Hello, how are you?",
		Language: "en",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Verify multipart form fields
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		assert.Equal(t, "whisper-1", r.FormValue("model"))
		assert.Equal(t, "en", r.FormValue("language"))
		assert.Equal(t, "json", r.FormValue("response_format"))

		// Verify file was uploaded
		file, header, err := r.FormFile("file")
		require.NoError(t, err)
		defer file.Close()
		assert.Equal(t, "audio.ogg", header.Filename)
		data, _ := io.ReadAll(file)
		assert.Equal(t, []byte("fake-ogg-data"), data)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewSTTClient(STTConfig{
		APIURL:  server.URL,
		APIKey:  "test-key",
		Model:   "whisper-1",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	result, err := client.Transcribe(context.Background(), []byte("fake-ogg-data"), "en")
	require.NoError(t, err)
	assert.Equal(t, expected.Text, result.Text)
}

func TestSTT_Transcribe_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid audio"}`))
	}))
	defer server.Close()

	client := NewSTTClient(STTConfig{
		APIURL:  server.URL,
		APIKey:  "test-key",
		Model:   "whisper-1",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	result, err := client.Transcribe(context.Background(), []byte("bad-data"), "en")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 400")
}

func TestSTT_Transcribe_NoAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Authorization header when API key is empty
		assert.Empty(t, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(STTResult{Text: "test"})
	}))
	defer server.Close()

	client := NewSTTClient(STTConfig{
		APIURL:  server.URL,
		APIKey:  "",
		Model:   "whisper-1",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	result, err := client.Transcribe(context.Background(), []byte("data"), "")
	require.NoError(t, err)
	assert.Equal(t, "test", result.Text)
}

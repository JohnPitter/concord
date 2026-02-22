package voice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTranslateService implements TranslateService for testing.
type mockTranslateService struct {
	result string
	err    error
}

func (m *mockTranslateService) TranslateTextDirect(_ context.Context, text, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.result != "" {
		return m.result, nil
	}
	return text + " (translated)", nil
}

func TestVoiceTranslator_EnableDisable(t *testing.T) {
	vt := NewVoiceTranslator(nil, nil, nil, nil, 3*time.Second, zerolog.Nop())

	assert.False(t, vt.IsEnabled())

	err := vt.Enable("en", "pt")
	require.NoError(t, err)
	assert.True(t, vt.IsEnabled())

	status := vt.GetStatus()
	assert.True(t, status.Enabled)
	assert.Equal(t, "en", status.SourceLang)
	assert.Equal(t, "pt", status.TargetLang)

	err = vt.Disable()
	require.NoError(t, err)
	assert.False(t, vt.IsEnabled())

	status = vt.GetStatus()
	assert.False(t, status.Enabled)
}

func TestVoiceTranslator_Enable_RequiresLanguages(t *testing.T) {
	vt := NewVoiceTranslator(nil, nil, nil, nil, 3*time.Second, zerolog.Nop())

	err := vt.Enable("", "pt")
	assert.Error(t, err)

	err = vt.Enable("en", "")
	assert.Error(t, err)
}

func TestVoiceTranslator_FramesToOGG(t *testing.T) {
	vt := NewVoiceTranslator(nil, nil, nil, nil, 3*time.Second, zerolog.Nop())

	// Create some fake Opus frames (just random bytes for testing the OGG container)
	frames := [][]byte{
		{0x01, 0x02, 0x03, 0x04},
		{0x05, 0x06, 0x07, 0x08},
		{0x09, 0x0a, 0x0b, 0x0c},
	}

	ogg, err := vt.framesToOGG(frames)
	require.NoError(t, err)
	assert.NotEmpty(t, ogg)

	// OGG files start with "OggS" magic bytes
	assert.Equal(t, "OggS", string(ogg[:4]))
}

func TestVoiceTranslator_FramesToOGG_Empty(t *testing.T) {
	vt := NewVoiceTranslator(nil, nil, nil, nil, 3*time.Second, zerolog.Nop())

	ogg, err := vt.framesToOGG([][]byte{})
	require.NoError(t, err)
	// Empty frames should still produce valid OGG header
	assert.NotEmpty(t, ogg)
}

func TestVoiceTranslator_ProcessSegment(t *testing.T) {
	// Setup fake STT server
	sttServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(STTResult{Text: "Hello world", Language: "en"})
	}))
	defer sttServer.Close()

	// Setup fake TTS server
	fakeAudio := []byte("fake-mp3-audio")
	ttsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(fakeAudio)
	}))
	defer ttsServer.Close()

	sttClient := NewSTTClient(STTConfig{
		APIURL:  sttServer.URL,
		Model:   "whisper-1",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	ttsClient := NewTTSClient(TTSConfig{
		APIURL:  ttsServer.URL,
		Voice:   "alloy",
		Format:  "mp3",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	translateSvc := &mockTranslateService{result: "Olá mundo"}

	var mu sync.Mutex
	var emittedEvents []TranslatedAudioPayload

	emitter := func(eventName string, data ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if eventName == "voice:translated-audio" && len(data) > 0 {
			if payload, ok := data[0].(TranslatedAudioPayload); ok {
				emittedEvents = append(emittedEvents, payload)
			}
		}
	}

	vt := NewVoiceTranslator(sttClient, ttsClient, translateSvc, emitter, 3*time.Second, zerolog.Nop())

	// Create fake Opus frames
	frames := make([][]byte, 10)
	for i := range frames {
		frames[i] = []byte{0x01, 0x02, 0x03, 0x04}
	}

	vt.processSegment(context.Background(), "peer-1", frames, "en", "pt")

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, emittedEvents, 1)
	assert.Equal(t, "peer-1", emittedEvents[0].PeerID)
	assert.Equal(t, "mp3", emittedEvents[0].Format)
	assert.NotEmpty(t, emittedEvents[0].Audio)
}

func TestVoiceTranslator_AccumulatorFlush(t *testing.T) {
	// Use a very short segment length for testing
	segmentLength := 50 * time.Millisecond

	// Setup fake STT server
	sttServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(STTResult{Text: "Test", Language: "en"})
	}))
	defer sttServer.Close()

	// Setup fake TTS server
	ttsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("audio"))
	}))
	defer ttsServer.Close()

	sttClient := NewSTTClient(STTConfig{
		APIURL:  sttServer.URL,
		Model:   "whisper-1",
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	ttsClient := NewTTSClient(TTSConfig{
		APIURL:  ttsServer.URL,
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	translateSvc := &mockTranslateService{}

	var mu sync.Mutex
	eventCount := 0

	emitter := func(eventName string, data ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		if eventName == "voice:translated-audio" {
			eventCount++
		}
	}

	vt := NewVoiceTranslator(sttClient, ttsClient, translateSvc, emitter, segmentLength, zerolog.Nop())

	err := vt.Enable("en", "pt")
	require.NoError(t, err)

	// Push frames until segment flushes
	for i := 0; i < 5; i++ {
		vt.PushOpusFrame("peer-1", []byte{0x01, 0x02})
		time.Sleep(15 * time.Millisecond)
	}

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := eventCount
	mu.Unlock()

	assert.GreaterOrEqual(t, count, 1, "expected at least one segment flush")
}

func TestVoiceTranslator_PushFrameWhenDisabled(t *testing.T) {
	vt := NewVoiceTranslator(nil, nil, nil, nil, 3*time.Second, zerolog.Nop())

	// Should not panic when disabled
	vt.PushOpusFrame("peer-1", []byte{0x01, 0x02})

	vt.mu.RLock()
	_, exists := vt.accumulators["peer-1"]
	vt.mu.RUnlock()

	assert.False(t, exists, "should not accumulate when disabled")
}

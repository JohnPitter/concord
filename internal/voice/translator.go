package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
	"github.com/rs/zerolog"
)

// VoiceTranslatorConfig holds configuration for the voice translator.
type VoiceTranslatorConfig struct {
	SegmentLength time.Duration
	SourceLang    string
	TargetLang    string
}

// TranslatedAudioPayload is the Wails event payload for translated audio.
type TranslatedAudioPayload struct {
	PeerID string `json:"peerID"`
	Audio  string `json:"audio"`  // base64 encoded
	Format string `json:"format"` // "mp3"
}

// VoiceTranslationStatus represents the current state of voice translation.
type VoiceTranslationStatus struct {
	Enabled    bool   `json:"enabled"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

// TranslateService is the interface the VoiceTranslator needs from translation.Service.
type TranslateService interface {
	TranslateTextDirect(ctx context.Context, text, sourceLang, targetLang string) (string, error)
}

// EventEmitter is the interface for emitting Wails events.
type EventEmitter func(eventName string, data ...interface{})

// opusAccumulator accumulates Opus frames for a peer until a segment is ready.
type opusAccumulator struct {
	frames    [][]byte
	startTime time.Time
	peerID    string
}

// VoiceTranslator orchestrates the voice translation pipeline:
// Opus frames -> OGG -> STT (Whisper) -> translate (LibreTranslate) -> TTS -> Wails event
type VoiceTranslator struct {
	mu          sync.RWMutex
	enabled     bool
	sourceLang  string
	targetLang  string
	stt         *STTClient
	tts         *TTSClient
	translation TranslateService
	logger      zerolog.Logger
	emitEvent   EventEmitter

	segmentLength time.Duration
	accumulators  map[string]*opusAccumulator

	ctx    context.Context
	cancel context.CancelFunc
}

// NewVoiceTranslator creates a new voice translator pipeline.
func NewVoiceTranslator(
	stt *STTClient,
	tts *TTSClient,
	translationSvc TranslateService,
	emitEvent EventEmitter,
	segmentLength time.Duration,
	logger zerolog.Logger,
) *VoiceTranslator {
	if segmentLength == 0 {
		segmentLength = 3 * time.Second
	}

	return &VoiceTranslator{
		stt:           stt,
		tts:           tts,
		translation:   translationSvc,
		emitEvent:     emitEvent,
		segmentLength: segmentLength,
		accumulators:  make(map[string]*opusAccumulator),
		logger:        logger.With().Str("component", "voice-translator").Logger(),
	}
}

// Enable activates voice translation between two languages.
func (vt *VoiceTranslator) Enable(sourceLang, targetLang string) error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if sourceLang == "" || targetLang == "" {
		return fmt.Errorf("voice-translator: source and target languages required")
	}

	vt.enabled = true
	vt.sourceLang = sourceLang
	vt.targetLang = targetLang
	vt.accumulators = make(map[string]*opusAccumulator)
	vt.ctx, vt.cancel = context.WithCancel(context.Background())

	vt.logger.Info().
		Str("source_lang", sourceLang).
		Str("target_lang", targetLang).
		Msg("voice translation enabled")

	return nil
}

// Disable deactivates voice translation and clears accumulators.
func (vt *VoiceTranslator) Disable() error {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	vt.enabled = false
	vt.sourceLang = ""
	vt.targetLang = ""
	vt.accumulators = make(map[string]*opusAccumulator)

	if vt.cancel != nil {
		vt.cancel()
		vt.cancel = nil
	}

	vt.logger.Info().Msg("voice translation disabled")
	return nil
}

// IsEnabled returns whether voice translation is active.
func (vt *VoiceTranslator) IsEnabled() bool {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.enabled
}

// GetStatus returns the current voice translation status.
func (vt *VoiceTranslator) GetStatus() VoiceTranslationStatus {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return VoiceTranslationStatus{
		Enabled:    vt.enabled,
		SourceLang: vt.sourceLang,
		TargetLang: vt.targetLang,
	}
}

// PushOpusFrame receives an Opus frame from a peer and accumulates it.
// When the accumulator reaches the segment length, it dispatches processSegment in a goroutine.
func (vt *VoiceTranslator) PushOpusFrame(peerID string, frame []byte) {
	vt.mu.Lock()

	if !vt.enabled {
		vt.mu.Unlock()
		return
	}

	acc, ok := vt.accumulators[peerID]
	if !ok {
		acc = &opusAccumulator{
			peerID:    peerID,
			startTime: time.Now(),
		}
		vt.accumulators[peerID] = acc
	}

	// Copy frame to avoid data races (buffer is reused by caller)
	frameCopy := make([]byte, len(frame))
	copy(frameCopy, frame)
	acc.frames = append(acc.frames, frameCopy)

	elapsed := time.Since(acc.startTime)
	if elapsed >= vt.segmentLength {
		// Flush: take accumulated frames and reset
		frames := acc.frames
		acc.frames = nil
		acc.startTime = time.Now()

		sourceLang := vt.sourceLang
		targetLang := vt.targetLang
		ctx := vt.ctx
		vt.mu.Unlock()

		go vt.processSegment(ctx, peerID, frames, sourceLang, targetLang)
		return
	}

	vt.mu.Unlock()
}

// processSegment executes the full pipeline: OGG -> STT -> Translate -> TTS -> emit event
func (vt *VoiceTranslator) processSegment(ctx context.Context, peerID string, frames [][]byte, sourceLang, targetLang string) {
	start := time.Now()

	// 1. Pack Opus frames into OGG container
	oggData, err := vt.framesToOGG(frames)
	if err != nil {
		vt.logger.Error().Err(err).Str("peer_id", peerID).Msg("failed to create OGG from frames")
		return
	}

	// 2. STT: OGG -> text
	sttResult, err := vt.stt.Transcribe(ctx, oggData, sourceLang)
	if err != nil {
		vt.logger.Error().Err(err).Str("peer_id", peerID).Msg("STT transcription failed")
		return
	}

	// Skip empty transcriptions (silence)
	text := strings.TrimSpace(sttResult.Text)
	if text == "" {
		vt.logger.Debug().Str("peer_id", peerID).Msg("empty transcription, skipping")
		return
	}

	// 3. Translate text
	translatedText, err := vt.translation.TranslateTextDirect(ctx, text, sourceLang, targetLang)
	if err != nil {
		vt.logger.Error().Err(err).Str("peer_id", peerID).Msg("text translation failed")
		return
	}

	// 4. TTS: translated text -> audio
	audioData, err := vt.tts.Synthesize(ctx, translatedText, targetLang)
	if err != nil {
		vt.logger.Error().Err(err).Str("peer_id", peerID).Msg("TTS synthesis failed")
		return
	}

	// 5. Emit Wails event with base64-encoded audio
	payload := TranslatedAudioPayload{
		PeerID: peerID,
		Audio:  base64.StdEncoding.EncodeToString(audioData),
		Format: "mp3",
	}

	if vt.emitEvent != nil {
		vt.emitEvent("voice:translated-audio", payload)
	}

	latency := time.Since(start)
	vt.logger.Info().
		Str("peer_id", peerID).
		Dur("total_latency", latency).
		Int("frames", len(frames)).
		Int("text_len", len(text)).
		Int("translated_len", len(translatedText)).
		Int("audio_bytes", len(audioData)).
		Msg("voice translation segment completed")
}

// framesToOGG packs Opus frames into an OGG container using pion's oggwriter.
func (vt *VoiceTranslator) framesToOGG(frames [][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := oggwriter.NewWith(&buf, SampleRate, Channels)
	if err != nil {
		return nil, fmt.Errorf("create ogg writer: %w", err)
	}

	var seq uint16
	var ts uint32

	for _, frame := range frames {
		pkt := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    111,
				SequenceNumber: seq,
				Timestamp:      ts,
			},
			Payload: frame,
		}
		if err := w.WriteRTP(pkt); err != nil {
			_ = w.Close()
			return nil, fmt.Errorf("write rtp to ogg: %w", err)
		}
		seq++
		ts += uint32(FrameSize) // 960 samples per 20ms frame at 48kHz
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close ogg writer: %w", err)
	}

	return buf.Bytes(), nil
}

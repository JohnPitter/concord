package voice

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog"
)

// State represents the current voice engine state.
type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
)

// SpeakerInfo holds info about an active speaker.
type SpeakerInfo struct {
	PeerID    string  `json:"peer_id"`
	UserID    string  `json:"user_id"`
	Username  string  `json:"username"`
	Volume    float64 `json:"volume"`
	Speaking  bool    `json:"speaking"`
}

// EngineConfig holds voice engine settings.
type EngineConfig struct {
	Bitrate       int
	Jitter        JitterConfig
	VAD           VADConfig
	ICEServers    []webrtc.ICEServer
}

// DefaultEngineConfig returns sensible defaults.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		Bitrate: DefaultBitrate,
		Jitter:  DefaultJitterConfig(),
		VAD:     DefaultVADConfig(),
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun1.l.google.com:19302"}},
		},
	}
}

// Engine is the core voice chat engine that manages WebRTC connections,
// audio mixing, and voice activity detection.
type Engine struct {
	mu          sync.RWMutex
	state       State
	channelID   string
	muted       bool
	deafened    bool
	mixer       *Mixer
	vad         *VAD
	peers       map[string]*peerConnection // peerID -> connection
	speakers    map[string]*SpeakerInfo
	config      EngineConfig
	logger      zerolog.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	translator  *VoiceTranslator

	// Callbacks
	onStateChange    func(State)
	onSpeakersChange func([]SpeakerInfo)
	onICECandidate   func(peerID string, candidate webrtc.ICECandidateInit)
}

type peerConnection struct {
	pc     *webrtc.PeerConnection
	jitter *JitterBuffer
	userID string
}

// NewEngine creates a new voice engine.
func NewEngine(cfg EngineConfig, logger zerolog.Logger) *Engine {
	return &Engine{
		state:    StateDisconnected,
		mixer:    NewMixer(),
		vad:      NewVAD(cfg.VAD),
		peers:    make(map[string]*peerConnection),
		speakers: make(map[string]*SpeakerInfo),
		config:   cfg,
		logger:   logger.With().Str("component", "voice-engine").Logger(),
	}
}

// State returns the current engine state.
func (e *Engine) State() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

// ChannelID returns the current voice channel ID.
func (e *Engine) ChannelID() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.channelID
}

// IsMuted returns whether the local microphone is muted.
func (e *Engine) IsMuted() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.muted
}

// IsDeafened returns whether the local audio output is deafened.
func (e *Engine) IsDeafened() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.deafened
}

// OnStateChange registers a callback for state changes.
func (e *Engine) OnStateChange(fn func(State)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onStateChange = fn
}

// OnSpeakersChange registers a callback for active speaker updates.
func (e *Engine) OnSpeakersChange(fn func([]SpeakerInfo)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onSpeakersChange = fn
}

// SetOnICECandidate registers a callback for trickle ICE candidates.
// Called when a new ICE candidate is discovered for a peer connection.
func (e *Engine) SetOnICECandidate(fn func(peerID string, candidate webrtc.ICECandidateInit)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onICECandidate = fn
}

// JoinChannel starts the voice connection for a channel.
func (e *Engine) JoinChannel(ctx context.Context, channelID string) error {
	e.mu.Lock()
	if e.state != StateDisconnected {
		e.mu.Unlock()
		return fmt.Errorf("voice: already connected to channel %s", e.channelID)
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.channelID = channelID
	e.state = StateConnecting
	e.mu.Unlock()

	e.setState(StateConnecting)
	e.logger.Info().Str("channel_id", channelID).Msg("joining voice channel")

	e.setState(StateConnected)
	return nil
}

// LeaveChannel disconnects from the current voice channel.
func (e *Engine) LeaveChannel() error {
	e.mu.Lock()
	if e.state == StateDisconnected {
		e.mu.Unlock()
		return nil
	}

	channelID := e.channelID
	e.mu.Unlock()

	e.logger.Info().Str("channel_id", channelID).Msg("leaving voice channel")

	// Close all peer connections
	e.mu.Lock()
	for peerID, pc := range e.peers {
		if err := pc.pc.Close(); err != nil {
			e.logger.Warn().Err(err).Str("peer_id", peerID).Msg("failed to close peer connection")
		}
		e.mixer.RemoveStream(peerID)
	}
	e.peers = make(map[string]*peerConnection)
	e.speakers = make(map[string]*SpeakerInfo)
	e.channelID = ""

	if e.cancel != nil {
		e.cancel()
	}
	e.mu.Unlock()

	e.setState(StateDisconnected)
	return nil
}

// Mute toggles the microphone mute state.
func (e *Engine) Mute() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.muted = !e.muted
	e.logger.Info().Bool("muted", e.muted).Msg("mute toggled")
}

// SetMuted sets the mute state explicitly.
func (e *Engine) SetMuted(muted bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.muted = muted
}

// Deafen toggles the audio output deafen state.
func (e *Engine) Deafen() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deafened = !e.deafened
	if e.deafened {
		e.muted = true // deafen implies mute
	}
	e.logger.Info().Bool("deafened", e.deafened).Bool("muted", e.muted).Msg("deafen toggled")
}

// SetDeafened sets the deafen state explicitly.
func (e *Engine) SetDeafened(deafened bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.deafened = deafened
	if deafened {
		e.muted = true
	}
}

// AddPeer creates a WebRTC peer connection for a remote user.
func (e *Engine) AddPeer(peerID, userID, username string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.peers[peerID]; exists {
		return nil // already connected
	}

	config := webrtc.Configuration{
		ICEServers: e.config.ICEServers,
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("voice: create peer connection: %w", err)
	}

	// Add Opus audio track
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"concord-voice",
	)
	if err != nil {
		_ = pc.Close()
		return fmt.Errorf("voice: create audio track: %w", err)
	}

	if _, err := pc.AddTrack(audioTrack); err != nil {
		_ = pc.Close()
		return fmt.Errorf("voice: add track: %w", err)
	}

	// Handle incoming audio from peer
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		e.logger.Info().
			Str("peer_id", peerID).
			Str("codec", track.Codec().MimeType).
			Msg("received remote audio track")

		e.handleRemoteTrack(peerID, track)
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		e.logger.Info().
			Str("peer_id", peerID).
			Str("state", state.String()).
			Msg("ICE connection state changed")

		if state == webrtc.ICEConnectionStateFailed || state == webrtc.ICEConnectionStateDisconnected {
			e.removePeer(peerID)
		}
	})

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return // gathering complete
		}
		e.mu.RLock()
		cb := e.onICECandidate
		e.mu.RUnlock()
		if cb != nil {
			cb(peerID, c.ToJSON())
		}
	})

	e.peers[peerID] = &peerConnection{
		pc:     pc,
		jitter: NewJitterBuffer(e.config.Jitter),
		userID: userID,
	}

	e.mixer.AddStream(peerID)
	e.speakers[peerID] = &SpeakerInfo{
		PeerID:   peerID,
		UserID:   userID,
		Username: username,
	}

	e.logger.Info().
		Str("peer_id", peerID).
		Str("user_id", userID).
		Msg("peer added to voice")

	return nil
}

// RemovePeer disconnects a remote peer.
func (e *Engine) RemovePeer(peerID string) {
	e.removePeer(peerID)
}

func (e *Engine) removePeer(peerID string) {
	e.mu.Lock()
	pc, ok := e.peers[peerID]
	if ok {
		delete(e.peers, peerID)
		delete(e.speakers, peerID)
	}
	e.mu.Unlock()

	if ok {
		e.mixer.RemoveStream(peerID)
		if err := pc.pc.Close(); err != nil {
			e.logger.Warn().Err(err).Str("peer_id", peerID).Msg("failed to close peer connection")
		}
		e.logger.Info().Str("peer_id", peerID).Msg("peer removed from voice")
	}
}

// AddSelfSpeaker adds the local user as a speaker so they appear in the
// speakers list without needing a real peer connection.
func (e *Engine) AddSelfSpeaker(peerID, userID, username string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.speakers[peerID] = &SpeakerInfo{
		PeerID:   peerID,
		UserID:   userID,
		Username: username,
	}
}

// GetActiveSpeakers returns the list of currently active speakers.
func (e *Engine) GetActiveSpeakers() []SpeakerInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	speakers := make([]SpeakerInfo, 0, len(e.speakers))
	for _, s := range e.speakers {
		speakers = append(speakers, *s)
	}
	return speakers
}

// PeerCount returns the number of connected peers.
func (e *Engine) PeerCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.peers)
}

// CreateOffer creates a WebRTC offer for a peer (SDP exchange).
func (e *Engine) CreateOffer(peerID string) (string, error) {
	e.mu.RLock()
	pc, ok := e.peers[peerID]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("voice: peer %s not found", peerID)
	}

	offer, err := pc.pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("voice: create offer: %w", err)
	}

	if err := pc.pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("voice: set local description: %w", err)
	}

	return offer.SDP, nil
}

// HandleAnswer processes a WebRTC answer from a peer.
func (e *Engine) HandleAnswer(peerID, sdp string) error {
	e.mu.RLock()
	pc, ok := e.peers[peerID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("voice: peer %s not found", peerID)
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdp,
	}

	if err := pc.pc.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("voice: set remote description: %w", err)
	}

	return nil
}

// HandleOffer processes a WebRTC offer from a peer and returns an answer SDP.
func (e *Engine) HandleOffer(peerID, sdp string) (string, error) {
	e.mu.RLock()
	pc, ok := e.peers[peerID]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("voice: peer %s not found", peerID)
	}

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}

	if err := pc.pc.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("voice: set remote description: %w", err)
	}

	answer, err := pc.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("voice: create answer: %w", err)
	}

	if err := pc.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("voice: set local description: %w", err)
	}

	return answer.SDP, nil
}

// AddICECandidate adds an ICE candidate from a peer.
func (e *Engine) AddICECandidate(peerID string, candidate webrtc.ICECandidateInit) error {
	e.mu.RLock()
	pc, ok := e.peers[peerID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("voice: peer %s not found", peerID)
	}

	return pc.pc.AddICECandidate(candidate)
}

// SetTranslator sets the voice translator for real-time voice translation.
func (e *Engine) SetTranslator(vt *VoiceTranslator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.translator = vt
}

// handleRemoteTrack processes incoming audio from a peer.
func (e *Engine) handleRemoteTrack(peerID string, track *webrtc.TrackRemote) {
	buf := make([]byte, 1500)

	for {
		n, _, err := track.Read(buf)
		if err != nil {
			e.logger.Debug().Err(err).Str("peer_id", peerID).Msg("remote track read ended")
			return
		}

		e.mu.RLock()
		pc, ok := e.peers[peerID]
		translator := e.translator
		e.mu.RUnlock()

		if !ok || e.deafened {
			continue
		}

		packet := buf[:n]

		// Push to jitter buffer
		pc.jitter.Push(packet, 0, 0)

		// Feed voice translator if enabled
		if translator != nil && translator.IsEnabled() {
			var rtpPkt rtp.Packet
			if err := rtpPkt.Unmarshal(packet); err == nil {
				translator.PushOpusFrame(peerID, rtpPkt.Payload)
			}
		}
	}
}

// setState updates the engine state and fires the callback.
func (e *Engine) setState(s State) {
	e.mu.Lock()
	e.state = s
	cb := e.onStateChange
	e.mu.Unlock()

	if cb != nil {
		cb(s)
	}
}

// GetVoiceState returns the full voice state for the frontend.
func (e *Engine) GetVoiceState() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	speakers := make([]SpeakerInfo, 0, len(e.speakers))
	for _, s := range e.speakers {
		speakers = append(speakers, *s)
	}

	return map[string]interface{}{
		"state":     string(e.state),
		"channelId": e.channelID,
		"muted":     e.muted,
		"deafened":  e.deafened,
		"peerCount": len(e.peers),
		"speakers":  speakers,
	}
}

// VoiceStatus is a simplified status for Wails bindings.
type VoiceStatus struct {
	State     string        `json:"state"`
	ChannelID string        `json:"channel_id"`
	Muted     bool          `json:"muted"`
	Deafened  bool          `json:"deafened"`
	PeerCount int           `json:"peer_count"`
	Speakers  []SpeakerInfo `json:"speakers"`
}

// GetStatus returns the current voice status.
func (e *Engine) GetStatus() VoiceStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	speakers := make([]SpeakerInfo, 0, len(e.speakers))
	for _, s := range e.speakers {
		speakers = append(speakers, *s)
	}

	return VoiceStatus{
		State:     string(e.state),
		ChannelID: e.channelID,
		Muted:     e.muted,
		Deafened:  e.deafened,
		PeerCount: len(e.peers),
		Speakers:  speakers,
	}
}

// Cleanup releases all resources. Used to satisfy a 5-minute periodic cleanup.
func (e *Engine) Cleanup() {
	e.mu.RLock()
	state := e.state
	e.mu.RUnlock()

	if state != StateDisconnected {
		return
	}

	// Clear any stale data
	e.mu.Lock()
	e.peers = make(map[string]*peerConnection)
	e.speakers = make(map[string]*SpeakerInfo)
	e.mu.Unlock()
}

// ensure time import is used for future ticker-based mixing
var _ = time.Second

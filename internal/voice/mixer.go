package voice

import (
	"sync"
)

// Mixer combines multiple audio streams into one output.
// Uses simple additive mixing with soft clipping.
type Mixer struct {
	mu      sync.RWMutex
	streams map[string]*audioStream // peerID -> stream
}

type audioStream struct {
	buffer []float32
	volume float32
}

// NewMixer creates a new audio mixer.
func NewMixer() *Mixer {
	return &Mixer{
		streams: make(map[string]*audioStream),
	}
}

// AddStream registers a new audio stream for mixing.
func (m *Mixer) AddStream(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streams[peerID] = &audioStream{
		buffer: make([]float32, FrameSize),
		volume: 1.0,
	}
}

// RemoveStream removes an audio stream.
func (m *Mixer) RemoveStream(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.streams, peerID)
}

// SetVolume sets the volume for a specific stream (0.0 to 1.0).
func (m *Mixer) SetVolume(peerID string, volume float32) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.streams[peerID]; ok {
		if volume < 0 {
			volume = 0
		}
		if volume > 1 {
			volume = 1
		}
		s.volume = volume
	}
}

// PushSamples adds decoded audio samples from a peer.
func (m *Mixer) PushSamples(peerID string, samples []float32) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.streams[peerID]
	if !ok {
		return
	}

	// Copy into stream buffer (truncate if needed)
	n := len(samples)
	if n > len(s.buffer) {
		n = len(s.buffer)
	}
	copy(s.buffer[:n], samples[:n])
}

// Mix combines all active streams into a single output frame.
// Returns FrameSize float32 samples. O(k) where k = number of streams.
func (m *Mixer) Mix() []float32 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	output := make([]float32, FrameSize)

	for _, s := range m.streams {
		for i := 0; i < FrameSize; i++ {
			output[i] += s.buffer[i] * s.volume
		}
	}

	// Soft clipping (tanh-like) to prevent distortion
	for i := range output {
		output[i] = softClip(output[i])
	}

	return output
}

// StreamCount returns the number of active streams.
func (m *Mixer) StreamCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.streams)
}

// softClip applies soft saturation to keep audio in [-1.0, 1.0].
// For small values it's nearly linear; for large values it compresses.
func softClip(x float32) float32 {
	if x > 1.0 {
		return 1.0 - 1.0/(x*x+1.0)
	}
	if x < -1.0 {
		return -(1.0 - 1.0/(x*x+1.0))
	}
	return x
}

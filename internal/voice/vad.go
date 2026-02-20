package voice

import (
	"math"
	"sync"
)

// VAD (Voice Activity Detection) detects whether an audio frame
// contains speech or silence. Uses energy-based detection with
// adaptive threshold.
type VAD struct {
	mu             sync.RWMutex
	threshold      float64 // Energy threshold in dB
	noiseFloor     float64 // Estimated noise floor in dB
	hangoverFrames int     // Frames to keep active after speech stops
	hangoverCount  int     // Current hangover counter
	active         bool    // Currently detecting speech
	adaptRate      float64 // Noise floor adaptation rate
}

// VADConfig holds VAD settings.
type VADConfig struct {
	Threshold      float64 // Initial energy threshold in dB (default: -40)
	HangoverFrames int     // Frames to keep active after speech (default: 15 = 300ms at 20ms frames)
	AdaptRate      float64 // Noise floor adaptation rate (default: 0.01)
}

// DefaultVADConfig returns default VAD settings.
func DefaultVADConfig() VADConfig {
	return VADConfig{
		Threshold:      -40.0,
		HangoverFrames: 15,
		AdaptRate:      0.01,
	}
}

// NewVAD creates a new Voice Activity Detector.
func NewVAD(cfg VADConfig) *VAD {
	if cfg.HangoverFrames == 0 {
		cfg = DefaultVADConfig()
	}
	return &VAD{
		threshold:      cfg.Threshold,
		noiseFloor:     -60.0,
		hangoverFrames: cfg.HangoverFrames,
		adaptRate:      cfg.AdaptRate,
	}
}

// Process analyzes an audio frame and returns true if speech is detected.
// Expects float32 PCM samples in [-1.0, 1.0].
func (v *VAD) Process(samples []float32) bool {
	energy := calculateEnergy(samples)
	energyDB := energyToDB(energy)

	v.mu.Lock()
	defer v.mu.Unlock()

	// Adapt noise floor during silence
	if !v.active {
		v.noiseFloor = v.noiseFloor*(1-v.adaptRate) + energyDB*v.adaptRate
	}

	// Dynamic threshold: noise floor + margin
	dynamicThreshold := v.noiseFloor + 15.0 // 15 dB above noise floor
	if dynamicThreshold < v.threshold {
		dynamicThreshold = v.threshold
	}

	if energyDB > dynamicThreshold {
		v.active = true
		v.hangoverCount = v.hangoverFrames
	} else if v.hangoverCount > 0 {
		v.hangoverCount--
	} else {
		v.active = false
	}

	return v.active
}

// IsActive returns the current VAD state.
func (v *VAD) IsActive() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.active
}

// Reset resets the VAD state.
func (v *VAD) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.active = false
	v.hangoverCount = 0
	v.noiseFloor = -60.0
}

// SetThreshold adjusts the energy threshold in dB.
func (v *VAD) SetThreshold(db float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = db
}

// calculateEnergy computes the RMS energy of a frame.
func calculateEnergy(samples []float32) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

// energyToDB converts RMS energy to decibels.
func energyToDB(energy float64) float64 {
	if energy <= 0 {
		return -100.0
	}
	return 20.0 * math.Log10(energy)
}

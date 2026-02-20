// Package voice implements the voice chat engine for Concord.
package voice

// Audio constants matching the ARCHITECTURE.md spec.
const (
	SampleRate     = 48000                              // 48kHz
	Channels       = 1                                  // Mono
	FrameDuration  = 20                                 // 20ms per frame
	FrameSize      = SampleRate * FrameDuration / 1000  // 960 samples at 48kHz
	DefaultBitrate = 64000                              // 64 kbps
	MinBitrate     = 32000
	MaxBitrate     = 128000
)

// int16ToFloat32 converts PCM int16 samples to float32 in [-1.0, 1.0].
func int16ToFloat32(pcm []int16) []float32 {
	out := make([]float32, len(pcm))
	for i, s := range pcm {
		out[i] = float32(s) / 32768.0
	}
	return out
}

// float32ToInt16 converts float32 samples back to int16 with clamping.
func float32ToInt16(pcm []float32) []int16 {
	out := make([]int16, len(pcm))
	for i, s := range pcm {
		v := s * 32768.0
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		out[i] = int16(v)
	}
	return out
}

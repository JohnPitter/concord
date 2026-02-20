# Concord Voice Pipeline Architecture

This document describes the audio pipeline used for real-time voice chat in Concord, including capture, encoding, transport, mixing, and optional translation.

---

## Table of Contents

- [Overview](#overview)
- [Pipeline Stages](#pipeline-stages)
- [Audio Capture](#audio-capture)
- [Opus Encoding](#opus-encoding)
- [WebRTC Transport](#webrtc-transport)
- [Audio Mixing](#audio-mixing)
- [Voice Activity Detection](#voice-activity-detection)
- [Translation Pipeline](#translation-pipeline)
- [Configuration](#configuration)
- [Performance](#performance)

---

## Overview

```
Microphone → VAD → Opus Encoder → WebRTC (DTLS-SRTP) → Opus Decoder → Mixer → Speaker
                                                                          ↑
                                                              [Translation Pipeline]
                                                              (PersonaPlex API)
```

The voice engine (`internal/voice/engine.go`) manages the full lifecycle of audio capture, encoding, transport, and playback.

---

## Pipeline Stages

| Stage | Component | Latency Budget |
|-------|-----------|---------------|
| Capture | OS audio API | ~5ms |
| VAD | Energy-based detector | ~1ms |
| Encoding | Opus codec | ~2ms |
| Network | WebRTC (DTLS-SRTP) | 20-100ms |
| Jitter Buffer | Adaptive | 50-200ms |
| Decoding | Opus codec | ~2ms |
| Mixing | Linear mixer | ~1ms |
| Playback | OS audio API | ~5ms |
| **Total** | | **~90-320ms** |

---

## Audio Capture

**Format:** PCM S16LE (signed 16-bit little-endian)

| Parameter | Value |
|-----------|-------|
| Sample rate | 48,000 Hz |
| Channels | 1 (mono) |
| Frame size | 960 samples (20ms) |
| Bit depth | 16 bits |

Capture happens via the OS WebView's WebAudio API (Wails bridge) on the frontend, or directly via platform audio APIs on the backend.

---

## Opus Encoding

Opus is used for all voice encoding/decoding:

| Parameter | Value |
|-----------|-------|
| Application | VOIP |
| Bitrate | 64,000 bps |
| Frame duration | 20ms |
| Complexity | 10 |
| FEC | Enabled |
| DTX | Enabled |

**Forward Error Correction (FEC):** Opus includes redundancy data from the previous frame, allowing recovery from single packet loss without retransmission.

**Discontinuous Transmission (DTX):** When VAD detects silence, the encoder sends comfort noise parameters at reduced rate (~1 packet/400ms), saving bandwidth.

---

## WebRTC Transport

### ICE Connectivity

1. Gather local candidates (host, srflx via STUN, relay via TURN)
2. Exchange candidates via signaling server
3. ICE connectivity checks determine best path
4. DTLS handshake establishes SRTP keys

### RTP/RTCP

| Parameter | Value |
|-----------|-------|
| Payload type | 111 (Opus) |
| Clock rate | 48,000 |
| SSRC | Random per session |
| RTCP interval | 5 seconds |

### Jitter Buffer

Adaptive jitter buffer (`internal/voice/jitter.go`):

- **Minimum:** 50ms (configured via `VoiceConfig.JitterBufferSize`)
- **Maximum:** 200ms (configured via `VoiceConfig.MaxJitterBuffer`)
- **Algorithm:** Adapts based on inter-arrival jitter (RFC 3550)
- **Behavior on underrun:** Plays silence (no audio stretching)
- **Behavior on overflow:** Drops oldest packets

---

## Audio Mixing

When multiple speakers are active, the mixer (`internal/voice/mixer.go`) combines audio:

```
Speaker A PCM ──┐
                ├──→ Linear Mix → Clamp → Output PCM
Speaker B PCM ──┘
```

**Algorithm:**
1. Sum all PCM samples (int32 accumulator to avoid overflow)
2. Divide by number of active speakers (normalization)
3. Clamp to int16 range [-32768, 32767]

**Complexity:** O(s * f) where s = number of speakers, f = frame size (960)

The mixer skips the local user's audio to prevent echo.

---

## Voice Activity Detection

Energy-based VAD (`internal/voice/vad.go`):

1. Compute RMS energy of the frame: `sqrt(sum(sample^2) / frame_size)`
2. Compare against threshold (`VoiceConfig.VADThreshold`, default 0.01)
3. Apply hold time (200ms) to prevent clipping on word boundaries

When VAD is inactive:
- Opus DTX kicks in (reduced packet rate)
- Speaker indicator in UI turns off
- Translation pipeline skips the frame

---

## Translation Pipeline

Optional real-time voice translation via PersonaPlex API (`internal/translation/`):

```
Opus Decoder → PCM → PersonaPlex Stream → Translated PCM → Opus Encoder → Mixer
```

### Flow

1. User enables translation in settings (source + target language)
2. `TranslationService.Enable()` starts the pipeline
3. Decoded PCM frames from remote speakers are sent to PersonaPlex WebSocket
4. Translated audio frames are received and injected into the mixer
5. If PersonaPlex fails, original audio passes through (graceful degradation)

### Circuit Breaker

- Monitors PersonaPlex response latency
- If latency > `MaxLatency` (default 500ms) for `FailureThreshold` (default 3) consecutive requests:
  - Circuit opens → translation bypassed
  - Log warning
  - Retry after 30 seconds

### Translation Cache

- Caches text translations using `internal/cache/LRU`
- Key: `translate:{src}:{tgt}:{sha256(text)}`
- TTL: 1 hour
- Max entries: configurable (default 1000)
- Persistent cache in SQLite (`translation_cache` table)

---

## Configuration

All voice parameters are in `internal/config/config.go` → `VoiceConfig`:

```json
{
  "voice": {
    "sample_rate": 48000,
    "channels": 1,
    "frame_size": 960,
    "bitrate": 64000,
    "jitter_buffer_size": "50ms",
    "max_jitter_buffer": "200ms",
    "enable_vad": true,
    "vad_threshold": 0.01,
    "enable_noise_suppression": false,
    "max_channel_users": 25
  },
  "translation": {
    "enabled": false,
    "personaplex_url": "https://api.personaplex.nvidia.com/v1",
    "api_key": "",
    "default_lang": "en",
    "cache_enabled": true,
    "cache_size": 1000,
    "timeout": "5s",
    "max_latency": "500ms",
    "circuit_breaker": true,
    "failure_threshold": 3
  }
}
```

---

## Performance

### Bandwidth per User

| Mode | Bandwidth |
|------|-----------|
| Active speaking | ~8 KB/s (64 kbps Opus) |
| Silent (DTX) | ~0.2 KB/s |
| Overhead (RTP/DTLS) | ~2 KB/s |

### CPU Usage

| Operation | CPU per Frame (20ms) |
|-----------|---------------------|
| Opus encode | ~0.5ms |
| Opus decode (per speaker) | ~0.3ms |
| Mixing (5 speakers) | ~0.1ms |
| VAD | ~0.05ms |

### Scalability

- Max 25 users per voice channel (configurable)
- Each peer maintains N-1 WebRTC peer connections (full mesh)
- For channels > 10 users, consider SFU architecture (future work)

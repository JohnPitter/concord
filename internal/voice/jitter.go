package voice

import (
	"sync"
	"time"
)

// JitterBuffer smooths out network timing variations in audio delivery.
// It holds incoming audio packets and releases them at a steady rate.
type JitterBuffer struct {
	mu          sync.Mutex
	buffer      []*audioPacket
	targetDelay time.Duration // target buffering delay
	minDelay    time.Duration
	maxDelay    time.Duration
	maxPackets  int
}

type audioPacket struct {
	data      []byte
	seq       uint16
	timestamp uint32
	received  time.Time
}

// JitterConfig holds jitter buffer settings.
type JitterConfig struct {
	TargetDelay time.Duration // Default: 50ms
	MinDelay    time.Duration // Default: 20ms
	MaxDelay    time.Duration // Default: 200ms
	MaxPackets  int           // Default: 50
}

// DefaultJitterConfig returns default jitter buffer settings.
func DefaultJitterConfig() JitterConfig {
	return JitterConfig{
		TargetDelay: 50 * time.Millisecond,
		MinDelay:    20 * time.Millisecond,
		MaxDelay:    200 * time.Millisecond,
		MaxPackets:  50,
	}
}

// NewJitterBuffer creates a new adaptive jitter buffer.
func NewJitterBuffer(cfg JitterConfig) *JitterBuffer {
	if cfg.TargetDelay == 0 {
		cfg = DefaultJitterConfig()
	}
	return &JitterBuffer{
		buffer:      make([]*audioPacket, 0, cfg.MaxPackets),
		targetDelay: cfg.TargetDelay,
		minDelay:    cfg.MinDelay,
		maxDelay:    cfg.MaxDelay,
		maxPackets:  cfg.MaxPackets,
	}
}

// Push adds an audio packet to the buffer, maintaining sequence order.
func (jb *JitterBuffer) Push(data []byte, seq uint16, timestamp uint32) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	pkt := &audioPacket{
		data:      make([]byte, len(data)),
		seq:       seq,
		timestamp: timestamp,
		received:  time.Now(),
	}
	copy(pkt.data, data)

	// Drop if buffer is full (oldest packets already consumed)
	if len(jb.buffer) >= jb.maxPackets {
		jb.buffer = jb.buffer[1:]
	}

	// Insert in sequence order
	inserted := false
	for i := len(jb.buffer) - 1; i >= 0; i-- {
		if seqLessThan(jb.buffer[i].seq, seq) {
			// Insert after this position
			jb.buffer = append(jb.buffer, nil)
			copy(jb.buffer[i+2:], jb.buffer[i+1:])
			jb.buffer[i+1] = pkt
			inserted = true
			break
		}
		if jb.buffer[i].seq == seq {
			return // duplicate, drop
		}
	}
	if !inserted {
		// Insert at beginning
		jb.buffer = append([]*audioPacket{pkt}, jb.buffer...)
	}
}

// Pop retrieves the next audio packet if available and ready.
// Returns nil if no packet is ready yet.
func (jb *JitterBuffer) Pop() []byte {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.buffer) == 0 {
		return nil
	}

	pkt := jb.buffer[0]

	// Check if the packet has been buffered long enough
	elapsed := time.Since(pkt.received)
	if elapsed < jb.targetDelay && len(jb.buffer) < 3 {
		return nil // not enough delay accumulated and buffer not backed up
	}

	jb.buffer = jb.buffer[1:]
	return pkt.data
}

// Len returns the current number of buffered packets.
func (jb *JitterBuffer) Len() int {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	return len(jb.buffer)
}

// Reset clears the jitter buffer.
func (jb *JitterBuffer) Reset() {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	jb.buffer = jb.buffer[:0]
}

// SetTargetDelay adjusts the target delay (adaptive).
func (jb *JitterBuffer) SetTargetDelay(d time.Duration) {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	if d < jb.minDelay {
		d = jb.minDelay
	}
	if d > jb.maxDelay {
		d = jb.maxDelay
	}
	jb.targetDelay = d
}

// seqLessThan handles uint16 sequence number wraparound.
func seqLessThan(a, b uint16) bool {
	return (b-a) > 0 && (b-a) < 0x8000
}

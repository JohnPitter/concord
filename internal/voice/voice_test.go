package voice

import (
	"context"
	"math"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).Level(zerolog.ErrorLevel)
}

// --- Codec tests ---

func TestInt16ToFloat32(t *testing.T) {
	pcm := []int16{0, 32767, -32768, 16384}
	f := int16ToFloat32(pcm)

	assert.InDelta(t, 0.0, f[0], 0.001)
	assert.InDelta(t, 1.0, f[1], 0.001)
	assert.InDelta(t, -1.0, f[2], 0.001)
	assert.InDelta(t, 0.5, f[3], 0.001)
}

func TestFloat32ToInt16(t *testing.T) {
	pcm := []float32{0.0, 1.0, -1.0, 0.5}
	i := float32ToInt16(pcm)

	assert.Equal(t, int16(0), i[0])
	assert.Equal(t, int16(32767), i[1])      // clamped
	assert.Equal(t, int16(-32768), i[2])     // clamped
	assert.InDelta(t, 16384, float64(i[3]), 1)
}

func TestFloat32ToInt16Clamp(t *testing.T) {
	pcm := []float32{2.0, -2.0}
	i := float32ToInt16(pcm)
	assert.Equal(t, int16(32767), i[0])
	assert.Equal(t, int16(-32768), i[1])
}

func TestRoundTrip(t *testing.T) {
	original := []int16{0, 100, -100, 1000, -1000, 32767, -32768}
	f := int16ToFloat32(original)
	back := float32ToInt16(f)

	for i := range original {
		assert.InDelta(t, float64(original[i]), float64(back[i]), 1.0, "sample %d", i)
	}
}

// --- Jitter Buffer tests ---

func TestJitterBufferPushPop(t *testing.T) {
	jb := NewJitterBuffer(JitterConfig{
		TargetDelay: 10 * time.Millisecond,
		MinDelay:    5 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		MaxPackets:  10,
	})

	// Push packets
	jb.Push([]byte{0x01}, 1, 100)
	jb.Push([]byte{0x02}, 2, 200)
	jb.Push([]byte{0x03}, 3, 300)

	assert.Equal(t, 3, jb.Len())

	// Wait for target delay
	time.Sleep(15 * time.Millisecond)

	data := jb.Pop()
	require.NotNil(t, data)
	assert.Equal(t, byte(0x01), data[0])
}

func TestJitterBufferOrdering(t *testing.T) {
	jb := NewJitterBuffer(JitterConfig{
		TargetDelay: 1 * time.Millisecond,
		MinDelay:    1 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		MaxPackets:  10,
	})

	// Push out of order
	jb.Push([]byte{0x03}, 3, 300)
	jb.Push([]byte{0x01}, 1, 100)
	jb.Push([]byte{0x02}, 2, 200)

	time.Sleep(5 * time.Millisecond)

	// Should come out in order
	d1 := jb.Pop()
	d2 := jb.Pop()
	d3 := jb.Pop()

	require.NotNil(t, d1)
	require.NotNil(t, d2)
	require.NotNil(t, d3)
	assert.Equal(t, byte(0x01), d1[0])
	assert.Equal(t, byte(0x02), d2[0])
	assert.Equal(t, byte(0x03), d3[0])
}

func TestJitterBufferDuplicate(t *testing.T) {
	jb := NewJitterBuffer(DefaultJitterConfig())

	jb.Push([]byte{0x01}, 1, 100)
	jb.Push([]byte{0x01}, 1, 100) // duplicate seq
	assert.Equal(t, 1, jb.Len())
}

func TestJitterBufferReset(t *testing.T) {
	jb := NewJitterBuffer(DefaultJitterConfig())
	jb.Push([]byte{0x01}, 1, 100)
	jb.Push([]byte{0x02}, 2, 200)
	assert.Equal(t, 2, jb.Len())

	jb.Reset()
	assert.Equal(t, 0, jb.Len())
}

func TestJitterBufferMaxPackets(t *testing.T) {
	jb := NewJitterBuffer(JitterConfig{
		TargetDelay: 10 * time.Millisecond,
		MinDelay:    5 * time.Millisecond,
		MaxDelay:    100 * time.Millisecond,
		MaxPackets:  3,
	})

	jb.Push([]byte{0x01}, 1, 100)
	jb.Push([]byte{0x02}, 2, 200)
	jb.Push([]byte{0x03}, 3, 300)
	jb.Push([]byte{0x04}, 4, 400) // should evict oldest

	assert.Equal(t, 3, jb.Len())
}

// --- Mixer tests ---

func TestMixerSingleStream(t *testing.T) {
	m := NewMixer()
	m.AddStream("peer1")

	samples := make([]float32, FrameSize)
	for i := range samples {
		samples[i] = 0.5
	}
	m.PushSamples("peer1", samples)

	output := m.Mix()
	assert.Len(t, output, FrameSize)
	assert.InDelta(t, 0.5, output[0], 0.01)
}

func TestMixerMultipleStreams(t *testing.T) {
	m := NewMixer()
	m.AddStream("peer1")
	m.AddStream("peer2")

	s1 := make([]float32, FrameSize)
	s2 := make([]float32, FrameSize)
	for i := range s1 {
		s1[i] = 0.3
		s2[i] = 0.4
	}
	m.PushSamples("peer1", s1)
	m.PushSamples("peer2", s2)

	output := m.Mix()
	// 0.3 + 0.4 = 0.7 (within soft clip range, so nearly linear)
	assert.InDelta(t, 0.7, output[0], 0.01)
}

func TestMixerSoftClip(t *testing.T) {
	m := NewMixer()
	m.AddStream("peer1")
	m.AddStream("peer2")

	s1 := make([]float32, FrameSize)
	s2 := make([]float32, FrameSize)
	for i := range s1 {
		s1[i] = 0.9
		s2[i] = 0.9
	}
	m.PushSamples("peer1", s1)
	m.PushSamples("peer2", s2)

	output := m.Mix()
	// Sum would be 1.8, soft clipping should bring it down
	assert.Less(t, output[0], float32(1.0))
	assert.Greater(t, output[0], float32(0.5))
}

func TestMixerVolume(t *testing.T) {
	m := NewMixer()
	m.AddStream("peer1")
	m.SetVolume("peer1", 0.5)

	samples := make([]float32, FrameSize)
	for i := range samples {
		samples[i] = 0.8
	}
	m.PushSamples("peer1", samples)

	output := m.Mix()
	// 0.8 * 0.5 = 0.4
	assert.InDelta(t, 0.4, output[0], 0.01)
}

func TestMixerRemoveStream(t *testing.T) {
	m := NewMixer()
	m.AddStream("peer1")
	assert.Equal(t, 1, m.StreamCount())

	m.RemoveStream("peer1")
	assert.Equal(t, 0, m.StreamCount())
}

// --- VAD tests ---

func TestVADSilence(t *testing.T) {
	v := NewVAD(DefaultVADConfig())

	silence := make([]float32, FrameSize)
	active := v.Process(silence)
	assert.False(t, active)
}

func TestVADSpeech(t *testing.T) {
	v := NewVAD(VADConfig{
		Threshold:      -40.0,
		HangoverFrames: 3,
		AdaptRate:      0.01,
	})

	// Generate a loud signal (sine wave)
	speech := make([]float32, FrameSize)
	for i := range speech {
		speech[i] = 0.5 * float32(math.Sin(2*math.Pi*440*float64(i)/float64(SampleRate)))
	}

	active := v.Process(speech)
	assert.True(t, active)
}

func TestVADHangover(t *testing.T) {
	v := NewVAD(VADConfig{
		Threshold:      -40.0,
		HangoverFrames: 2,
		AdaptRate:      0.01,
	})

	// Send speech
	speech := make([]float32, FrameSize)
	for i := range speech {
		speech[i] = 0.5
	}
	v.Process(speech)
	assert.True(t, v.IsActive())

	// Send silence — should stay active due to hangover
	silence := make([]float32, FrameSize)
	v.Process(silence)
	assert.True(t, v.IsActive(), "should be active during hangover")

	v.Process(silence)
	assert.True(t, v.IsActive(), "should still be active (hangover frame 2)")

	v.Process(silence)
	assert.False(t, v.IsActive(), "should be inactive after hangover expires")
}

func TestVADReset(t *testing.T) {
	v := NewVAD(DefaultVADConfig())

	speech := make([]float32, FrameSize)
	for i := range speech {
		speech[i] = 0.5
	}
	v.Process(speech)
	assert.True(t, v.IsActive())

	v.Reset()
	assert.False(t, v.IsActive())
}

// --- Engine tests ---

func TestEngineLifecycle(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), testLogger())

	assert.Equal(t, StateDisconnected, e.State())
	assert.Equal(t, "", e.ChannelID())
	assert.False(t, e.IsMuted())
	assert.False(t, e.IsDeafened())

	// Join
	err := e.JoinChannel(context.Background(), "ch-1")
	require.NoError(t, err)
	assert.Equal(t, StateConnected, e.State())
	assert.Equal(t, "ch-1", e.ChannelID())

	// Leave
	err = e.LeaveChannel()
	require.NoError(t, err)
	assert.Equal(t, StateDisconnected, e.State())
	assert.Equal(t, "", e.ChannelID())
}

func TestEngineMuteDeafen(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), testLogger())

	e.Mute()
	assert.True(t, e.IsMuted())
	e.Mute()
	assert.False(t, e.IsMuted())

	e.Deafen()
	assert.True(t, e.IsDeafened())
	assert.True(t, e.IsMuted(), "deafen implies mute")

	e.Deafen()
	assert.False(t, e.IsDeafened())
}

func TestEngineDoubleJoin(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), testLogger())

	require.NoError(t, e.JoinChannel(context.Background(), "ch-1"))
	err := e.JoinChannel(context.Background(), "ch-2")
	assert.Error(t, err, "should fail when already connected")

	e.LeaveChannel()
}

func TestEngineStateCallback(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), testLogger())

	states := make([]State, 0)
	e.OnStateChange(func(s State) {
		states = append(states, s)
	})

	e.JoinChannel(context.Background(), "ch-1")
	e.LeaveChannel()

	assert.Contains(t, states, StateConnecting)
	assert.Contains(t, states, StateConnected)
	assert.Contains(t, states, StateDisconnected)
}

func TestEnginePeerManagement(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), testLogger())
	require.NoError(t, e.JoinChannel(context.Background(), "ch-1"))
	defer e.LeaveChannel()

	err := e.AddPeer("peer-1", "user-1", "Alice")
	require.NoError(t, err)
	assert.Equal(t, 1, e.PeerCount())

	speakers := e.GetActiveSpeakers()
	assert.Len(t, speakers, 1)
	assert.Equal(t, "Alice", speakers[0].Username)

	e.RemovePeer("peer-1")
	assert.Equal(t, 0, e.PeerCount())
}

func TestEngineGetStatus(t *testing.T) {
	e := NewEngine(DefaultEngineConfig(), testLogger())
	require.NoError(t, e.JoinChannel(context.Background(), "ch-1"))
	defer e.LeaveChannel()

	e.SetMuted(true)
	e.AddPeer("peer-1", "user-1", "Bob")

	status := e.GetStatus()
	assert.Equal(t, "connected", status.State)
	assert.Equal(t, "ch-1", status.ChannelID)
	assert.True(t, status.Muted)
	assert.False(t, status.Deafened)
	assert.Equal(t, 1, status.PeerCount)
	assert.Len(t, status.Speakers, 1)
}

func TestSoftClip(t *testing.T) {
	// Values within range should be unchanged
	assert.InDelta(t, 0.5, softClip(0.5), 0.01)
	assert.InDelta(t, -0.5, softClip(-0.5), 0.01)
	assert.InDelta(t, 0.0, softClip(0.0), 0.001)

	// Values outside range should be clipped
	assert.Less(t, softClip(2.0), float32(1.0))
	assert.Greater(t, softClip(-2.0), float32(-1.0))
}

func TestSeqLessThan(t *testing.T) {
	assert.True(t, seqLessThan(1, 2))
	assert.False(t, seqLessThan(2, 1))
	assert.False(t, seqLessThan(1, 1))
	// Wraparound
	assert.True(t, seqLessThan(65535, 0))
	assert.True(t, seqLessThan(65534, 0))
}

func TestCalculateEnergy(t *testing.T) {
	silence := make([]float32, 100)
	assert.InDelta(t, 0.0, calculateEnergy(silence), 0.001)

	loud := make([]float32, 100)
	for i := range loud {
		loud[i] = 0.5
	}
	assert.InDelta(t, 0.5, calculateEnergy(loud), 0.001)
}

func TestEnergyToDB(t *testing.T) {
	assert.InDelta(t, 0.0, energyToDB(1.0), 0.1)
	assert.InDelta(t, -6.0, energyToDB(0.5), 0.5) // ~-6 dB
	assert.Equal(t, -100.0, energyToDB(0.0))
}

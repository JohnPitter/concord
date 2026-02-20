package p2p

import (
	"context"
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

func TestNewHostAndStop(t *testing.T) {
	cfg := Config{
		ListenPort: 0,
		EnableMDNS: false,
		EnableDHT:  false,
	}

	h, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h.Stop()

	assert.NotEmpty(t, h.ID())
	assert.NotEmpty(t, h.Addrs())
	assert.Equal(t, 0, h.PeerCount())
}

func TestTwoPeersConnect(t *testing.T) {
	cfg := Config{
		ListenPort: 0,
		EnableMDNS: false,
		EnableDHT:  false,
	}

	h1, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h1.Stop()

	h2, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h2.Stop()

	// Connect h2 to h1
	h1Addrs := h1.Addrs()
	require.NotEmpty(t, h1Addrs)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h2.Connect(ctx, h1Addrs[0])
	require.NoError(t, err)

	// Verify connectivity
	assert.GreaterOrEqual(t, h2.PeerCount(), 1)
}

func TestSendAndReceiveData(t *testing.T) {
	cfg := Config{
		ListenPort: 0,
		EnableMDNS: false,
		EnableDHT:  false,
	}

	h1, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h1.Stop()

	h2, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h2.Stop()

	// Set up message handler on h2
	received := make(chan []byte, 1)
	h2.OnMessage(func(peerID string, data []byte) {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		received <- dataCopy
	})

	// Connect h1 to h2
	h2Addrs := h2.Addrs()
	require.NotEmpty(t, h2Addrs)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h1.Connect(ctx, h2Addrs[0])
	require.NoError(t, err)

	// Give the connection a moment to stabilize
	time.Sleep(100 * time.Millisecond)

	// Send data from h1 to h2
	err = h1.SendData(ctx, h2.ID(), []byte("hello p2p"))
	require.NoError(t, err)

	// Wait for receipt
	select {
	case data := <-received:
		assert.Equal(t, "hello p2p", string(data))
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestPeersInfo(t *testing.T) {
	cfg := Config{
		ListenPort: 0,
		EnableMDNS: false,
		EnableDHT:  false,
	}

	h1, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h1.Stop()

	h2, err := New(cfg, testLogger())
	require.NoError(t, err)
	defer h2.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h1.Connect(ctx, h2.Addrs()[0])
	require.NoError(t, err)

	peers := h1.Peers()
	require.GreaterOrEqual(t, len(peers), 1)
	assert.Equal(t, h2.ID(), peers[0].ID)
	assert.True(t, peers[0].Connected)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.True(t, cfg.EnableMDNS)
	assert.True(t, cfg.EnableDHT)
	assert.Equal(t, 0, cfg.ListenPort)
}

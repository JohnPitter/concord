package signaling

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() zerolog.Logger {
	return zerolog.New(os.Stderr).Level(zerolog.ErrorLevel)
}

func setupServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := NewServer(testLogger())
	httpSrv := httptest.NewServer(srv.Handler())
	t.Cleanup(func() { httpSrv.Close() })
	return srv, httpSrv
}

func wsURL(httpSrv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(httpSrv.URL, "http")
}

func TestNewSignalAndDecodePayload(t *testing.T) {
	payload := JoinPayload{
		UserID:    "user-1",
		PeerID:    "peer-1",
		Addresses: []string{"/ip4/127.0.0.1/tcp/4001"},
	}

	sig, err := NewSignal(SignalJoin, "user-1", payload)
	require.NoError(t, err)
	assert.Equal(t, SignalJoin, sig.Type)
	assert.Equal(t, "user-1", sig.From)

	var decoded JoinPayload
	require.NoError(t, sig.DecodePayload(&decoded))
	assert.Equal(t, "user-1", decoded.UserID)
	assert.Equal(t, "peer-1", decoded.PeerID)
}

func TestSignalNilPayload(t *testing.T) {
	sig, err := NewSignal(SignalLeave, "user-1", nil)
	require.NoError(t, err)
	assert.Nil(t, sig.Payload)

	var dummy struct{}
	err = sig.DecodePayload(&dummy)
	assert.ErrorIs(t, err, ErrInvalidMsg)
}

func TestServerClientIntegration(t *testing.T) {
	srv, httpSrv := setupServer(t)
	url := wsURL(httpSrv)

	// Create two clients
	client1 := NewClient(url, testLogger())
	client2 := NewClient(url, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, client1.Connect(ctx))
	defer client1.Close()
	require.NoError(t, client2.Connect(ctx))
	defer client2.Close()

	assert.True(t, client1.Connected())
	assert.True(t, client2.Connected())

	// Set up handler for peer_joined on client2
	peerJoined := make(chan string, 1)
	client2.On(SignalPeerJoined, func(sig *Signal) {
		peerJoined <- sig.From
	})

	// Client2 joins first
	err := client2.JoinChannel("server-1", "channel-1", JoinPayload{
		UserID:    "user-2",
		PeerID:    "peer-2",
		Addresses: []string{"/ip4/127.0.0.1/tcp/4002"},
	})
	require.NoError(t, err)

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	// Set up peer list handler for client1
	peerList := make(chan int, 1)
	client1.On(SignalPeerList, func(sig *Signal) {
		var pl PeerListPayload
		sig.DecodePayload(&pl)
		peerList <- len(pl.Peers)
	})

	// Client1 joins
	err = client1.JoinChannel("server-1", "channel-1", JoinPayload{
		UserID:    "user-1",
		PeerID:    "peer-1",
		Addresses: []string{"/ip4/127.0.0.1/tcp/4001"},
	})
	require.NoError(t, err)

	// Client1 should receive peer list with client2
	select {
	case count := <-peerList:
		assert.Equal(t, 1, count, "should see 1 existing peer")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for peer list")
	}

	// Client2 should be notified about client1
	select {
	case from := <-peerJoined:
		assert.Equal(t, "peer-1", from)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for peer joined")
	}

	// Verify server state
	assert.Equal(t, 1, srv.ChannelCount())
	assert.Equal(t, 2, srv.PeerCount())
}

func TestClientDisconnect(t *testing.T) {
	_, httpSrv := setupServer(t)
	url := wsURL(httpSrv)

	client := NewClient(url, testLogger())
	ctx := context.Background()

	require.NoError(t, client.Connect(ctx))
	assert.True(t, client.Connected())

	require.NoError(t, client.Close())
	// After close, Connected should return false
	assert.False(t, client.Connected())
}

func TestClientSendWithoutConnection(t *testing.T) {
	client := NewClient("ws://localhost:0", testLogger())
	err := client.Send(&Signal{Type: SignalJoin})
	assert.ErrorIs(t, err, ErrNotConnected)
}

func TestServerPeerLeave(t *testing.T) {
	srv, httpSrv := setupServer(t)
	url := wsURL(httpSrv)

	client1 := NewClient(url, testLogger())
	client2 := NewClient(url, testLogger())

	ctx := context.Background()
	require.NoError(t, client1.Connect(ctx))
	defer client1.Close()
	require.NoError(t, client2.Connect(ctx))

	// Both join
	client1.JoinChannel("s1", "c1", JoinPayload{UserID: "u1", PeerID: "p1"})
	client2.JoinChannel("s1", "c1", JoinPayload{UserID: "u2", PeerID: "p2"})
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, srv.PeerCount())

	// Set up leave handler on client1
	peerLeft := make(chan string, 1)
	client1.On(SignalPeerLeft, func(sig *Signal) {
		peerLeft <- sig.From
	})

	// Client2 leaves
	client2.LeaveChannel("s1", "c1", "u2")
	time.Sleep(100 * time.Millisecond)

	// Client2 disconnects
	client2.Close()
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, srv.PeerCount())
}

func TestServerHandler(t *testing.T) {
	srv := NewServer(testLogger())
	handler := srv.Handler()

	// Test that handler returns a valid HTTP handler func
	assert.NotNil(t, handler)

	// Non-WebSocket request should fail gracefully
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestErrorPayload(t *testing.T) {
	ep := ErrorPayload{Code: 400, Message: "bad request"}
	sig, err := NewSignal(SignalError, "", ep)
	require.NoError(t, err)

	var decoded ErrorPayload
	require.NoError(t, sig.DecodePayload(&decoded))
	assert.Equal(t, 400, decoded.Code)
	assert.Equal(t, "bad request", decoded.Message)
}

func TestOfferPayload(t *testing.T) {
	offer := OfferPayload{
		PeerID:    "peer-1",
		Addresses: []string{"/ip4/1.2.3.4/tcp/4001"},
		PublicKey: []byte{0x01, 0x02, 0x03},
	}

	sig, err := NewSignal(SignalOffer, "peer-1", offer)
	require.NoError(t, err)

	var decoded OfferPayload
	require.NoError(t, sig.DecodePayload(&decoded))
	assert.Equal(t, "peer-1", decoded.PeerID)
	assert.Equal(t, []byte{0x01, 0x02, 0x03}, decoded.PublicKey)
}

func TestOfferForwarding(t *testing.T) {
	_, httpSrv := setupServer(t)
	url := wsURL(httpSrv)

	client1 := NewClient(url, testLogger())
	client2 := NewClient(url, testLogger())

	ctx := context.Background()
	require.NoError(t, client1.Connect(ctx))
	defer client1.Close()
	require.NoError(t, client2.Connect(ctx))
	defer client2.Close()

	// Both join the same channel
	client1.JoinChannel("s1", "c1", JoinPayload{UserID: "u1", PeerID: "p1"})
	client2.JoinChannel("s1", "c1", JoinPayload{UserID: "u2", PeerID: "p2"})
	time.Sleep(100 * time.Millisecond)

	// Set up offer handler on client2
	offerReceived := make(chan OfferPayload, 1)
	client2.On(SignalOffer, func(sig *Signal) {
		var offer OfferPayload
		if err := sig.DecodePayload(&offer); err == nil {
			offerReceived <- offer
		}
	})

	// Client1 sends offer to client2
	sig, err := NewSignal(SignalOffer, "p1", OfferPayload{
		PeerID:    "p1",
		Addresses: []string{"/ip4/1.2.3.4/tcp/9000"},
		PublicKey: []byte{0xAA, 0xBB},
	})
	require.NoError(t, err)
	sig.To = "p2"
	sig.ServerID = "s1"
	sig.ChannelID = "c1"
	require.NoError(t, client1.Send(sig))

	select {
	case offer := <-offerReceived:
		assert.Equal(t, "p1", offer.PeerID)
		assert.Equal(t, []byte{0xAA, 0xBB}, offer.PublicKey)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for offer")
	}
}

func TestMultipleChannels(t *testing.T) {
	srv, httpSrv := setupServer(t)
	url := wsURL(httpSrv)

	clients := make([]*Client, 3)
	for i := range clients {
		clients[i] = NewClient(url, testLogger())
		require.NoError(t, clients[i].Connect(context.Background()))
		defer clients[i].Close()
	}

	// Client 0 and 1 join channel A
	clients[0].JoinChannel("s1", "chA", JoinPayload{UserID: "u0", PeerID: "p0"})
	clients[1].JoinChannel("s1", "chA", JoinPayload{UserID: "u1", PeerID: "p1"})

	// Client 2 joins channel B
	clients[2].JoinChannel("s1", "chB", JoinPayload{UserID: "u2", PeerID: "p2"})

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, srv.ChannelCount())
	assert.Equal(t, 3, srv.PeerCount())

	// Get the per-channel breakdown
	assert.Equal(t, fmt.Sprintf("%d channels, %d peers", srv.ChannelCount(), srv.PeerCount()), "2 channels, 3 peers")
}

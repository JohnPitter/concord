package p2p_test

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/concord-chat/concord/internal/network/p2p"
)

func TestRoomCode_Deterministic(t *testing.T) {
	cfg := p2p.DefaultConfig()
	cfg.EnableMDNS = false
	cfg.EnableDHT = false
	h, err := p2p.New(cfg, zerolog.Nop())
	require.NoError(t, err)
	defer h.Stop()

	code1 := h.RoomCode()
	code2 := h.RoomCode()
	assert.Equal(t, code1, code2, "same host must generate same code")
	assert.True(t, strings.Contains(code1, "-"), "code must contain separator")
	parts := strings.SplitN(code1, "-", 2)
	assert.Len(t, parts, 2, "code must have word and number")
	assert.Len(t, parts[1], 4, "number part must be 4 digits")
}

func TestRoomRendezvous(t *testing.T) {
	rv := p2p.RoomRendezvous("alpha-1234")
	assert.Equal(t, "concord-room/alpha-1234", rv)
}

package protocol

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeTextMessage(t *testing.T) {
	msg := TextMessage{
		ID:        "msg-1",
		ChannelID: "ch-1",
		AuthorID:  "usr-1",
		Content:   "hello world",
		Timestamp: 1700000000,
	}

	data, err := Encode(TypeTextMessage, msg)
	require.NoError(t, err)

	// Verify header
	assert.Equal(t, byte(TypeTextMessage), data[0])

	// Decode
	env, err := Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, TypeTextMessage, env.Type)

	var decoded TextMessage
	require.NoError(t, env.DecodePayload(&decoded))
	assert.Equal(t, msg.ID, decoded.ID)
	assert.Equal(t, msg.Content, decoded.Content)
	assert.Equal(t, msg.Timestamp, decoded.Timestamp)
}

func TestEncodePingPong(t *testing.T) {
	ping := PingPong{Nonce: 42}
	data, err := Encode(TypePing, ping)
	require.NoError(t, err)

	env, err := Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, TypePing, env.Type)

	var decoded PingPong
	require.NoError(t, env.DecodePayload(&decoded))
	assert.Equal(t, uint64(42), decoded.Nonce)
}

func TestPayloadTooLarge(t *testing.T) {
	bigContent := make([]byte, MaxPayloadSize+1)
	msg := TextMessage{Content: string(bigContent)}
	_, err := Encode(TypeTextMessage, msg)
	assert.ErrorIs(t, err, ErrPayloadTooLarge)
}

func TestDecodeInvalidReader(t *testing.T) {
	// Empty reader
	_, err := Decode(bytes.NewReader(nil))
	assert.Error(t, err)
}

func TestDecodePartialPayload(t *testing.T) {
	// Valid header but truncated payload
	data := make([]byte, HeaderSize)
	data[0] = byte(TypePing)
	data[1] = 0
	data[2] = 0
	data[3] = 0
	data[4] = 10 // claims 10 bytes payload

	_, err := Decode(bytes.NewReader(data))
	assert.Error(t, err)
}

func TestEnvelopeEncodeRaw(t *testing.T) {
	env := &Envelope{
		Type:    TypePresenceUpdate,
		Payload: []byte{0x01, 0x02, 0x03},
	}
	data, err := env.EncodeRaw()
	require.NoError(t, err)
	assert.Equal(t, byte(TypePresenceUpdate), data[0])
	assert.Equal(t, 3+HeaderSize, len(data))

	// Round-trip
	decoded, err := Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, env.Type, decoded.Type)
	assert.Equal(t, env.Payload, decoded.Payload)
}

func TestAllMessageTypes(t *testing.T) {
	types := []MessageType{
		TypeTextMessage, TypeTextEdit, TypeTextDelete,
		TypeVoiceJoin, TypeVoiceLeave, TypeVoiceData, TypeVoiceMute,
		TypeFileOffer, TypeFileAccept, TypeFileChunk, TypeFileComplete,
		TypeServerSync, TypePresenceUpdate, TypeTypingStart, TypeTypingStop,
		TypePing, TypePong,
	}

	for _, mt := range types {
		data, err := Encode(mt, PingPong{Nonce: uint64(mt)})
		require.NoError(t, err)
		env, err := Decode(bytes.NewReader(data))
		require.NoError(t, err)
		assert.Equal(t, mt, env.Type, "message type mismatch for 0x%02x", mt)
	}
}

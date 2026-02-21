package p2p

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeEnvelope_Chat(t *testing.T) {
	payload := ChatPayload{Content: "oi", SentAt: "2026-02-21T10:00:00Z"}
	data, err := EncodeEnvelope(TypeChat, "peer-123", payload)
	require.NoError(t, err)

	env, err := DecodeEnvelope(data)
	require.NoError(t, err)
	assert.Equal(t, TypeChat, env.Type)
	assert.Equal(t, "peer-123", env.SenderID)

	var decoded ChatPayload
	require.NoError(t, json.Unmarshal(env.Payload, &decoded))
	assert.Equal(t, "oi", decoded.Content)
	assert.Equal(t, "2026-02-21T10:00:00Z", decoded.SentAt)
}

func TestEncodeDecodeEnvelope_Profile(t *testing.T) {
	payload := ProfilePayload{DisplayName: "Alice", AvatarDataURL: "data:image/png;base64,abc"}
	data, err := EncodeEnvelope(TypeProfile, "peer-456", payload)
	require.NoError(t, err)

	env, err := DecodeEnvelope(data)
	require.NoError(t, err)
	assert.Equal(t, TypeProfile, env.Type)
	assert.Equal(t, "peer-456", env.SenderID)

	var decoded ProfilePayload
	require.NoError(t, json.Unmarshal(env.Payload, &decoded))
	assert.Equal(t, "Alice", decoded.DisplayName)
	assert.Equal(t, "data:image/png;base64,abc", decoded.AvatarDataURL)
}

func TestDecodeEnvelope_InvalidJSON(t *testing.T) {
	_, err := DecodeEnvelope([]byte("not json"))
	assert.Error(t, err)
}

func TestEncodeEnvelope_NilPayload(t *testing.T) {
	data, err := EncodeEnvelope(TypeChat, "peer-789", nil)
	require.NoError(t, err)

	env, err := DecodeEnvelope(data)
	require.NoError(t, err)
	assert.Equal(t, TypeChat, env.Type)
}

package voice

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestICECredentialsProvider_Disabled(t *testing.T) {
	p := NewICECredentialsProvider("", 0, 0, "", 0)
	resp := p.BuildConfig("user-1", "")

	assert.False(t, p.Enabled())
	require.Len(t, resp.Servers, 2)
	assert.Equal(t, int64(0), resp.TTLSeconds)
	assert.Equal(t, int64(0), resp.ExpiresAt)
}

func TestICECredentialsProvider_Enabled(t *testing.T) {
	p := NewICECredentialsProvider("turn.example.com", 3478, 5349, "my-secret", 10*time.Minute)
	resp := p.BuildConfig("user:42", "")

	assert.True(t, p.Enabled())
	require.Len(t, resp.Servers, 4)
	assert.Greater(t, resp.TTLSeconds, int64(0))
	assert.Greater(t, resp.ExpiresAt, time.Now().UTC().Unix())

	turn := resp.Servers[2]
	require.NotEmpty(t, turn.URLs)
	assert.Contains(t, turn.URLs, "stun:turn.example.com:3478")
	assert.Contains(t, turn.URLs, "turn:turn.example.com:3478?transport=udp")
	assert.Contains(t, turn.URLs, "turn:turn.example.com:3478?transport=tcp")
	assert.Contains(t, turn.URLs, "turns:turn.example.com:5349?transport=tcp")

	parts := strings.SplitN(turn.Username, ":", 2)
	require.Len(t, parts, 2)
	_, err := strconv.ParseInt(parts[0], 10, 64)
	require.NoError(t, err)
	assert.Equal(t, "user_42", parts[1])

	mac := hmac.New(sha1.New, []byte("my-secret"))
	_, _ = mac.Write([]byte(turn.Username))
	expectedCredential := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expectedCredential, turn.Credential)

	fallback := resp.Servers[3]
	assert.Contains(t, fallback.URLs, "turn:openrelay.metered.ca:80")
	assert.Contains(t, fallback.URLs, "turn:openrelay.metered.ca:443")
	assert.Contains(t, fallback.URLs, "turns:openrelay.metered.ca:443")
	assert.Equal(t, "openrelayproject", fallback.Username)
	assert.Equal(t, "openrelayproject", fallback.Credential)
}

func TestICECredentialsProvider_UsesRequestHostFallback(t *testing.T) {
	p := NewICECredentialsProvider("", 3478, 0, "my-secret", 10*time.Minute)
	resp := p.BuildConfig("user", "voice.example.com:443")

	require.Len(t, resp.Servers, 4)
	turn := resp.Servers[2]
	assert.Contains(t, turn.URLs, "stun:voice.example.com:3478")
	assert.Contains(t, turn.URLs, "turn:voice.example.com:3478?transport=udp")
	assert.Contains(t, turn.URLs, "turn:voice.example.com:3478?transport=tcp")

	fallback := resp.Servers[3]
	assert.Contains(t, fallback.URLs, "turn:openrelay.metered.ca:80")
}

package voice

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTurnPort      = 3478
	defaultTurnTLSPort   = 5349
	minCredentialTTL     = 5 * time.Minute
	defaultCredentialTTL = 12 * time.Hour
	openRelayUsername    = "openrelayproject"
	openRelayCredential  = "openrelayproject"
)

// ICEServer represents a single ICE server entry for WebRTC peers.
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// ICEConfigResponse is returned to the frontend with STUN/TURN configuration.
type ICEConfigResponse struct {
	Servers    []ICEServer `json:"servers"`
	TTLSeconds int64       `json:"ttl_seconds"`
	ExpiresAt  int64       `json:"expires_at"`
}

// ICECredentialsProvider generates TURN REST credentials for browser peers.
type ICECredentialsProvider struct {
	turnHost      string
	turnPort      int
	turnTLSPort   int
	turnSecret    string
	credentialTTL time.Duration
}

// NewICECredentialsProvider creates a provider using TURN server settings.
func NewICECredentialsProvider(turnHost string, turnPort, turnTLSPort int, turnSecret string, credentialTTL time.Duration) *ICECredentialsProvider {
	turnHost = strings.TrimSpace(turnHost)
	turnSecret = strings.TrimSpace(turnSecret)
	if turnPort <= 0 {
		turnPort = defaultTurnPort
	}
	if turnTLSPort <= 0 {
		turnTLSPort = defaultTurnTLSPort
	}
	if credentialTTL < minCredentialTTL {
		credentialTTL = defaultCredentialTTL
	}

	return &ICECredentialsProvider{
		turnHost:      turnHost,
		turnPort:      turnPort,
		turnTLSPort:   turnTLSPort,
		turnSecret:    turnSecret,
		credentialTTL: credentialTTL,
	}
}

// Enabled returns true when TURN credentials can be generated.
func (p *ICECredentialsProvider) Enabled() bool {
	return p != nil && p.turnSecret != ""
}

// BuildConfig returns STUN defaults and TURN credentials when configured.
func (p *ICECredentialsProvider) BuildConfig(userID, publicHost string) ICEConfigResponse {
	resp := ICEConfigResponse{
		Servers: []ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun1.l.google.com:19302"}},
		},
	}

	if !p.Enabled() {
		return resp
	}

	turnHost := normalizeHost(p.turnHost)
	if turnHost == "" {
		turnHost = normalizeHost(publicHost)
	}
	if turnHost == "" {
		return resp
	}

	ttl := p.credentialTTL
	if ttl < minCredentialTTL {
		ttl = minCredentialTTL
	}

	now := time.Now().UTC()
	expiresAt := now.Add(ttl).Unix()
	cleanUserID := sanitizeUserID(userID)
	username := fmt.Sprintf("%d:%s", expiresAt, cleanUserID)

	mac := hmac.New(sha1.New, []byte(p.turnSecret))
	_, _ = mac.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	turnURLs := []string{
		"stun:" + turnHost + ":" + strconv.Itoa(p.turnPort),
		"turn:" + turnHost + ":" + strconv.Itoa(p.turnPort) + "?transport=udp",
		"turn:" + turnHost + ":" + strconv.Itoa(p.turnPort) + "?transport=tcp",
	}
	if p.turnTLSPort > 0 {
		turnURLs = append(turnURLs, "turns:"+turnHost+":"+strconv.Itoa(p.turnTLSPort)+"?transport=tcp")
	}

	resp.Servers = append(resp.Servers, ICEServer{
		URLs:       turnURLs,
		Username:   username,
		Credential: credential,
	})

	// Keep a public relay fallback available for server-mode browser clients.
	// This mirrors the fallback already used in the P2P voice engine and helps
	// when self-hosted TURN is unreachable from remote NATs.
	resp.Servers = append(resp.Servers, ICEServer{
		URLs: []string{
			"turn:openrelay.metered.ca:80",
			"turn:openrelay.metered.ca:443",
			"turns:openrelay.metered.ca:443",
		},
		Username:   openRelayUsername,
		Credential: openRelayCredential,
	})
	resp.TTLSeconds = int64(ttl.Seconds())
	resp.ExpiresAt = expiresAt
	return resp
}

func sanitizeUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "anonymous"
	}
	userID = strings.ReplaceAll(userID, ":", "_")
	return userID
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		host = strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://")
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}

	host = strings.Trim(host, "[]")
	return strings.TrimSpace(host)
}

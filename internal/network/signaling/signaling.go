// Package signaling provides WebSocket-based signaling for peer coordination.
// The signaling server helps peers discover each other and exchange connection
// metadata (addresses, public keys) before establishing direct P2P connections.
package signaling

import (
	"encoding/json"
	"errors"
)

// SignalType identifies the kind of signaling message.
type SignalType string

const (
	SignalJoin       SignalType = "join"        // Peer joins a server/channel
	SignalLeave      SignalType = "leave"       // Peer leaves
	SignalOffer      SignalType = "offer"       // Connection offer (addrs + public key)
	SignalAnswer     SignalType = "answer"      // Connection answer
	SignalPeerList   SignalType = "peer_list"   // Current peers in channel
	SignalPeerJoined SignalType = "peer_joined" // New peer notification
	SignalPeerLeft      SignalType = "peer_left"      // Peer departed notification
	SignalError         SignalType = "error"         // Error message
	SignalSDPOffer      SignalType = "sdp_offer"     // WebRTC SDP offer
	SignalSDPAnswer     SignalType = "sdp_answer"    // WebRTC SDP answer
	SignalICECandidate  SignalType = "ice_candidate" // WebRTC ICE candidate
	SignalPeerState     SignalType = "peer_state"    // Peer mute/deafen state update
)

var (
	ErrNotConnected = errors.New("signaling: not connected to server")
	ErrInvalidMsg   = errors.New("signaling: invalid message format")
)

// Signal is the envelope for all signaling messages.
type Signal struct {
	Type      SignalType       `json:"type"`
	From      string           `json:"from,omitempty"`
	To        string           `json:"to,omitempty"`
	ServerID  string           `json:"server_id,omitempty"`
	ChannelID string           `json:"channel_id,omitempty"`
	Payload   json.RawMessage  `json:"payload,omitempty"`
}

// JoinPayload is sent when joining a server/channel.
type JoinPayload struct {
	UserID    string   `json:"user_id"`
	PeerID    string   `json:"peer_id"`
	Username  string   `json:"username,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Addresses []string `json:"addresses"`
	PublicKey []byte   `json:"public_key,omitempty"`
	Muted     bool     `json:"muted,omitempty"`
	Deafened  bool     `json:"deafened,omitempty"`
}

// OfferPayload carries connection details for P2P establishment.
type OfferPayload struct {
	PeerID    string   `json:"peer_id"`
	Addresses []string `json:"addresses"`
	PublicKey []byte   `json:"public_key"`
}

// PeerListPayload is the list of peers currently in a channel.
type PeerListPayload struct {
	Peers []PeerEntry `json:"peers"`
	// ChannelStartedAt is unix milliseconds when the channel became active.
	ChannelStartedAt int64 `json:"channel_started_at,omitempty"`
}

// PeerEntry represents a single peer in the peer list.
type PeerEntry struct {
	UserID    string   `json:"user_id"`
	PeerID    string   `json:"peer_id"`
	Username  string   `json:"username,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Addresses []string `json:"addresses"`
	PublicKey []byte   `json:"public_key,omitempty"`
	Muted     bool     `json:"muted,omitempty"`
	Deafened  bool     `json:"deafened,omitempty"`
}

// PeerStatePayload carries mute/deafen state updates.
type PeerStatePayload struct {
	PeerID   string `json:"peer_id"`
	Muted    bool   `json:"muted"`
	Deafened bool   `json:"deafened"`
}

// SDPPayload carries a WebRTC session description (offer or answer).
type SDPPayload struct {
	SDP string `json:"sdp"`
}

// ICECandidatePayload carries a WebRTC ICE candidate.
type ICECandidatePayload struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdp_mid"`
	SDPMLineIndex uint16 `json:"sdp_mline_index"`
}

// ErrorPayload carries error details.
type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewSignal creates a signal with a JSON-marshaled payload.
func NewSignal(sigType SignalType, from string, payload interface{}) (*Signal, error) {
	var raw json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = data
	}
	return &Signal{
		Type:    sigType,
		From:    from,
		Payload: raw,
	}, nil
}

// DecodePayload unmarshals the signal payload into the target struct.
func (s *Signal) DecodePayload(v interface{}) error {
	if s.Payload == nil {
		return ErrInvalidMsg
	}
	return json.Unmarshal(s.Payload, v)
}

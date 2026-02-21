package p2p

import (
	"encoding/json"
	"fmt"
)

// MessageType identifica o tipo do envelope P2P.
type MessageType string

const (
	TypeProfile MessageType = "profile"
	TypeChat    MessageType = "chat"
)

// Envelope é o wrapper JSON trafegado pelo stream libp2p.
type Envelope struct {
	Type     MessageType     `json:"type"`
	SenderID string          `json:"sender_id"`
	Payload  json.RawMessage `json:"payload"`
}

// ProfilePayload é o payload do envelope de perfil.
type ProfilePayload struct {
	DisplayName   string `json:"display_name"`
	AvatarDataURL string `json:"avatar_data_url,omitempty"`
}

// ChatPayload é o payload do envelope de chat.
type ChatPayload struct {
	Content string `json:"content"`
	SentAt  string `json:"sent_at"`
}

// EncodeEnvelope serializa um envelope para bytes JSON.
// Complexity: O(n) onde n é o tamanho do payload.
func EncodeEnvelope(msgType MessageType, senderID string, payload any) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}
	env := Envelope{Type: msgType, SenderID: senderID, Payload: payloadBytes}
	data, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("encode envelope: %w", err)
	}
	return data, nil
}

// DecodeEnvelope desserializa bytes JSON para um Envelope.
func DecodeEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	return &env, nil
}

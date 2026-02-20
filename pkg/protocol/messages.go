// Package protocol defines the wire protocol for Concord P2P messaging.
// Wire format: [1 byte type][4 bytes length (big-endian)][payload (msgpack)]
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

// MessageType identifies the kind of protocol message.
type MessageType uint8

const (
	TypeTextMessage    MessageType = 0x01
	TypeTextEdit       MessageType = 0x02
	TypeTextDelete     MessageType = 0x03
	TypeVoiceJoin      MessageType = 0x10
	TypeVoiceLeave     MessageType = 0x11
	TypeVoiceData      MessageType = 0x12
	TypeVoiceMute      MessageType = 0x13
	TypeFileOffer      MessageType = 0x20
	TypeFileAccept     MessageType = 0x21
	TypeFileChunk      MessageType = 0x22
	TypeFileComplete   MessageType = 0x23
	TypeServerSync     MessageType = 0x30
	TypePresenceUpdate MessageType = 0x31
	TypeTypingStart    MessageType = 0x32
	TypeTypingStop     MessageType = 0x33
	TypePing           MessageType = 0xFE
	TypePong           MessageType = 0xFF
)

// MaxPayloadSize is the maximum allowed payload size (1 MB).
const MaxPayloadSize = 1 << 20

// HeaderSize is type (1) + length (4).
const HeaderSize = 5

var (
	ErrPayloadTooLarge = errors.New("protocol: payload exceeds max size")
	ErrInvalidHeader   = errors.New("protocol: invalid header")
)

// Envelope wraps a typed message for wire transport.
type Envelope struct {
	Type    MessageType `msgpack:"-"`
	Payload []byte      `msgpack:"-"`
}

// TextMessage is sent when a user posts a chat message.
type TextMessage struct {
	ID        string `msgpack:"id"`
	ChannelID string `msgpack:"channel_id"`
	AuthorID  string `msgpack:"author_id"`
	Content   string `msgpack:"content"`
	Timestamp int64  `msgpack:"ts"`
}

// TextEdit is sent when a user edits a message.
type TextEdit struct {
	MessageID string `msgpack:"message_id"`
	AuthorID  string `msgpack:"author_id"`
	Content   string `msgpack:"content"`
	Timestamp int64  `msgpack:"ts"`
}

// TextDelete is sent when a user deletes a message.
type TextDelete struct {
	MessageID string `msgpack:"message_id"`
	ActorID   string `msgpack:"actor_id"`
	Timestamp int64  `msgpack:"ts"`
}

// PresenceUpdate announces a user's online status.
type PresenceUpdate struct {
	UserID string `msgpack:"user_id"`
	Status string `msgpack:"status"` // "online", "idle", "dnd", "offline"
}

// TypingEvent signals typing start/stop.
type TypingEvent struct {
	ChannelID string `msgpack:"channel_id"`
	UserID    string `msgpack:"user_id"`
}

// PingPong is used for keepalive.
type PingPong struct {
	Nonce uint64 `msgpack:"nonce"`
}

// Encode serializes a message type and payload into wire format.
func Encode(msgType MessageType, v interface{}) ([]byte, error) {
	payload, err := msgpack.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("protocol: marshal failed: %w", err)
	}
	if len(payload) > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	buf := make([]byte, HeaderSize+len(payload))
	buf[0] = byte(msgType)
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(payload)))
	copy(buf[5:], payload)
	return buf, nil
}

// Decode reads one message from a reader and returns the envelope.
func Decode(r io.Reader) (*Envelope, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("protocol: read header: %w", err)
	}

	msgType := MessageType(header[0])
	length := binary.BigEndian.Uint32(header[1:5])

	if length > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("protocol: read payload: %w", err)
	}

	return &Envelope{Type: msgType, Payload: payload}, nil
}

// DecodePayload unmarshals the envelope payload into the target struct.
func (e *Envelope) DecodePayload(v interface{}) error {
	return msgpack.Unmarshal(e.Payload, v)
}

// EncodeRaw creates wire bytes from a pre-built envelope.
func (e *Envelope) EncodeRaw() ([]byte, error) {
	if len(e.Payload) > MaxPayloadSize {
		return nil, ErrPayloadTooLarge
	}
	buf := make([]byte, HeaderSize+len(e.Payload))
	buf[0] = byte(e.Type)
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(e.Payload)))
	copy(buf[5:], e.Payload)
	return buf, nil
}

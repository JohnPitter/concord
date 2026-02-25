package signaling

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	clientWriteWait      = 10 * time.Second
	clientPongWait       = 30 * time.Second
	clientPingPeriod     = 15 * time.Second
	reconnectInitialWait = 1 * time.Second
	reconnectMaxWait     = 30 * time.Second
)

// Client connects to a signaling server via WebSocket.
type Client struct {
	mu       sync.RWMutex
	conn     *websocket.Conn
	url      string
	handlers map[SignalType]SignalHandler
	logger   zerolog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}

	joinState *joinState
}

type joinState struct {
	serverID  string
	channelID string
	payload   JoinPayload
}

// SignalHandler is called when a signal of a given type is received.
type SignalHandler func(signal *Signal)

// NewClient creates a new signaling client (not yet connected).
func NewClient(url string, logger zerolog.Logger) *Client {
	return &Client{
		url:      url,
		handlers: make(map[SignalType]SignalHandler),
		logger:   logger.With().Str("component", "signaling-client").Logger(),
		done:     make(chan struct{}),
	}
}

// On registers a handler for a specific signal type.
func (c *Client) On(sigType SignalType, handler SignalHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[sigType] = handler
}

// Connect establishes the WebSocket connection and starts reading messages.
func (c *Client) Connect(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	conn, err := c.dial(c.ctx)
	if err != nil {
		return fmt.Errorf("signaling: connect to %s: %w", c.url, err)
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.configureConn(conn)
	c.logger.Info().Str("url", c.url).Msg("connected to signaling server")

	go c.pingLoop()
	go c.readLoop()
	return nil
}

func (c *Client) dial(ctx context.Context) (*websocket.Conn, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *Client) configureConn(conn *websocket.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(clientPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(clientPongWait))
	})
}

// Send sends a signal to the server.
func (c *Client) Send(signal *Signal) error {
	data, err := json.Marshal(signal)
	if err != nil {
		return fmt.Errorf("signaling: marshal signal: %w", err)
	}

	if err := c.writeMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("signaling: write: %w", err)
	}

	return nil
}

func (c *Client) writeMessage(messageType int, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return ErrNotConnected
	}

	_ = c.conn.SetWriteDeadline(time.Now().Add(clientWriteWait))
	return c.conn.WriteMessage(messageType, payload)
}

// JoinChannel sends a join signal for a server channel.
func (c *Client) JoinChannel(serverID, channelID string, payload JoinPayload) error {
	c.mu.Lock()
	c.joinState = &joinState{serverID: serverID, channelID: channelID, payload: payload}
	c.mu.Unlock()

	sig, err := NewSignal(SignalJoin, payload.UserID, payload)
	if err != nil {
		return err
	}
	sig.ServerID = serverID
	sig.ChannelID = channelID
	return c.Send(sig)
}

// LeaveChannel sends a leave signal.
func (c *Client) LeaveChannel(serverID, channelID, userID string) error {
	c.mu.Lock()
	c.joinState = nil
	c.mu.Unlock()

	sig, err := NewSignal(SignalLeave, userID, nil)
	if err != nil {
		return err
	}
	sig.ServerID = serverID
	sig.ChannelID = channelID
	return c.Send(sig)
}

// SendOffer sends a connection offer to a specific peer.
func (c *Client) SendOffer(toPeerID string, offer OfferPayload) error {
	sig, err := NewSignal(SignalOffer, offer.PeerID, offer)
	if err != nil {
		return err
	}
	sig.To = toPeerID
	return c.Send(sig)
}

// SendSDPOffer sends a WebRTC SDP offer to a specific peer.
func (c *Client) SendSDPOffer(serverID, channelID, toPeerID, sdp string) error {
	sig, err := NewSignal(SignalSDPOffer, "", SDPPayload{SDP: sdp})
	if err != nil {
		return err
	}
	sig.To = toPeerID
	sig.ServerID = serverID
	sig.ChannelID = channelID
	return c.Send(sig)
}

// SendSDPAnswer sends a WebRTC SDP answer to a specific peer.
func (c *Client) SendSDPAnswer(serverID, channelID, toPeerID, sdp string) error {
	sig, err := NewSignal(SignalSDPAnswer, "", SDPPayload{SDP: sdp})
	if err != nil {
		return err
	}
	sig.To = toPeerID
	sig.ServerID = serverID
	sig.ChannelID = channelID
	return c.Send(sig)
}

// SendICECandidate sends a WebRTC ICE candidate to a specific peer.
func (c *Client) SendICECandidate(serverID, channelID, toPeerID string, candidate ICECandidatePayload) error {
	sig, err := NewSignal(SignalICECandidate, "", candidate)
	if err != nil {
		return err
	}
	sig.To = toPeerID
	sig.ServerID = serverID
	sig.ChannelID = channelID
	return c.Send(sig)
}

// SendPeerState broadcasts the local peer mute/deafen state.
func (c *Client) SendPeerState(serverID, channelID string, payload PeerStatePayload) error {
	sig, err := NewSignal(SignalPeerState, "", payload)
	if err != nil {
		return err
	}
	sig.ServerID = serverID
	sig.ChannelID = channelID
	return c.Send(sig)
}

// Close disconnects from the signaling server.
func (c *Client) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn != nil {
		_ = conn.SetWriteDeadline(time.Now().Add(clientWriteWait))
		err := conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			c.logger.Debug().Err(err).Msg("close write failed")
		}
		_ = conn.Close()
	}

	c.logger.Info().Msg("signaling client closed")
	return nil
}

// Connected returns whether the client has an active connection.
func (c *Client) Connected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(clientPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.writeMessage(websocket.PingMessage, nil); err != nil && err != ErrNotConnected {
				c.logger.Debug().Err(err).Msg("signaling ping failed")
			}
		}
	}
}

// readLoop reads messages from the WebSocket.
func (c *Client) readLoop() {
	defer close(c.done)

	for {
		if c.ctx.Err() != nil {
			return
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			if !c.reconnect() {
				return
			}
			continue
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.Info().Msg("signaling connection closed")
			} else {
				c.logger.Warn().Err(err).Msg("signaling read error")
			}

			c.clearConn(conn)

			if !c.reconnect() {
				return
			}
			continue
		}

		var signal Signal
		if err := json.Unmarshal(msg, &signal); err != nil {
			c.logger.Warn().Err(err).Msg("invalid signal message")
			continue
		}

		c.mu.RLock()
		handler, ok := c.handlers[signal.Type]
		c.mu.RUnlock()

		if ok {
			handler(&signal)
		} else {
			c.logger.Debug().Str("type", string(signal.Type)).Msg("unhandled signal type")
		}
	}
}

func (c *Client) clearConn(conn *websocket.Conn) {
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.mu.Unlock()
	_ = conn.Close()
}

func (c *Client) reconnect() bool {
	backoff := reconnectInitialWait

	for {
		if c.ctx.Err() != nil {
			return false
		}

		select {
		case <-c.ctx.Done():
			return false
		case <-time.After(backoff):
		}

		conn, err := c.dial(c.ctx)
		if err != nil {
			c.logger.Warn().Err(err).Dur("backoff", backoff).Msg("signaling reconnect failed")
			backoff *= 2
			if backoff > reconnectMaxWait {
				backoff = reconnectMaxWait
			}
			continue
		}

		c.configureConn(conn)

		c.mu.Lock()
		c.conn = conn
		joinState := c.joinState
		c.mu.Unlock()

		c.logger.Info().Msg("signaling reconnected")

		if joinState != nil {
			sig, err := NewSignal(SignalJoin, joinState.payload.UserID, joinState.payload)
			if err != nil {
				c.logger.Warn().Err(err).Msg("failed to rebuild join signal after reconnect")
				return true
			}
			sig.ServerID = joinState.serverID
			sig.ChannelID = joinState.channelID
			if err := c.Send(sig); err != nil {
				c.logger.Warn().Err(err).Msg("failed to rejoin channel after reconnect")
			}
		}

		return true
	}
}

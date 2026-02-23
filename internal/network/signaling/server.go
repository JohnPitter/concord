package signaling

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 50 * time.Second
	maxMessageSize = 64 * 1024
	peerSendBuffer = 128
)

var (
	errPeerBackpressure = errors.New("signaling: peer send buffer full")
	upgrader            = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for dev
	}
)

// Server is a WebSocket signaling server that coordinates P2P connections.
type Server struct {
	mu sync.RWMutex
	// channels maps "serverID:channelID" -> map of peerID -> connection
	channels map[string]map[string]*peerConn
	logger   zerolog.Logger
}

type peerConn struct {
	conn      *websocket.Conn
	userID    string
	peerID    string
	username  string
	avatarURL string
	send      chan []byte
	closeOnce sync.Once
}

// enqueueJSON serializes and enqueues a message without blocking.
func (pc *peerConn) enqueueJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	select {
	case pc.send <- data:
		return nil
	default:
		return errPeerBackpressure
	}
}

func (pc *peerConn) startWritePump(logger zerolog.Logger) {
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			_ = pc.conn.Close()
		}()

		for {
			select {
			case data, ok := <-pc.send:
				_ = pc.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					_ = pc.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := pc.conn.WriteMessage(websocket.TextMessage, data); err != nil {
					logger.Debug().Err(err).Str("peer_id", pc.peerID).Msg("write to peer failed")
					return
				}
			case <-ticker.C:
				_ = pc.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := pc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.Debug().Err(err).Str("peer_id", pc.peerID).Msg("ping to peer failed")
					return
				}
			}
		}
	}()
}

func (pc *peerConn) close() {
	pc.closeOnce.Do(func() {
		close(pc.send)
	})
}

// NewServer creates a new signaling server.
func NewServer(logger zerolog.Logger) *Server {
	return &Server{
		channels: make(map[string]map[string]*peerConn),
		logger:   logger.With().Str("component", "signaling-server").Logger(),
	}
}

// Handler returns an HTTP handler for WebSocket connections.
func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error().Err(err).Msg("websocket upgrade failed")
			return
		}
		s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	var currentChannel string
	var currentPeerID string
	var currentPC *peerConn

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				s.logger.Debug().Msg("client disconnected")
			} else {
				s.logger.Warn().Err(err).Msg("read error")
			}
			if currentChannel != "" && currentPeerID != "" {
				s.removePeer(currentChannel, currentPeerID)
				// Broadcast peer_left so remaining peers clean up
				parts := splitChannelKey(currentChannel)
				s.broadcast(currentChannel, currentPeerID, &Signal{
					Type:      SignalPeerLeft,
					From:      currentPeerID,
					ServerID:  parts[0],
					ChannelID: parts[1],
				})
			}
			return
		}

		var signal Signal
		if err := json.Unmarshal(msg, &signal); err != nil {
			s.logger.Warn().Err(err).Msg("invalid signal")
			continue
		}

		channelKey := signal.ServerID + ":" + signal.ChannelID

		switch signal.Type {
		case SignalJoin:
			var payload JoinPayload
			if err := signal.DecodePayload(&payload); err != nil {
				if currentPC != nil {
					_ = currentPC.enqueueJSON(s.makeErrorSignal(400, "invalid join payload"))
				}
				continue
			}

			currentChannel = channelKey
			currentPeerID = payload.PeerID
			currentPC = &peerConn{
				conn:      conn,
				userID:    payload.UserID,
				peerID:    payload.PeerID,
				username:  payload.Username,
				avatarURL: payload.AvatarURL,
				send:      make(chan []byte, peerSendBuffer),
			}

			s.addPeer(channelKey, currentPC)
			currentPC.startWritePump(s.logger)

			// Send current peer list to the joiner
			s.sendPeerList(currentPC, channelKey, payload.PeerID)

			// Notify others about the new peer
			s.broadcast(channelKey, payload.PeerID, &Signal{
				Type:      SignalPeerJoined,
				From:      payload.PeerID,
				ServerID:  signal.ServerID,
				ChannelID: signal.ChannelID,
				Payload:   signal.Payload,
			})

			s.logger.Info().
				Str("user", payload.UserID).
				Str("peer", payload.PeerID).
				Str("channel", channelKey).
				Msg("peer joined channel")

		case SignalLeave:
			if currentChannel != "" && currentPeerID != "" {
				s.removePeer(currentChannel, currentPeerID)
				s.broadcast(currentChannel, currentPeerID, &Signal{
					Type:      SignalPeerLeft,
					From:      currentPeerID,
					ServerID:  signal.ServerID,
					ChannelID: signal.ChannelID,
				})
				currentChannel = ""
				currentPeerID = ""
				currentPC = nil
			}

		case SignalOffer, SignalAnswer, SignalSDPOffer, SignalSDPAnswer, SignalICECandidate:
			// Forward to specific peer
			if signal.To != "" {
				s.forwardToPeer(channelKey, signal.To, &signal)
			}

		default:
			s.logger.Debug().Str("type", string(signal.Type)).Msg("unknown signal type")
		}
	}
}

func (s *Server) addPeer(channelKey string, pc *peerConn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.channels[channelKey]; !ok {
		s.channels[channelKey] = make(map[string]*peerConn)
	}
	s.channels[channelKey][pc.peerID] = pc
}

func (s *Server) removePeer(channelKey, peerID string) {
	var pc *peerConn

	s.mu.Lock()
	if ch, ok := s.channels[channelKey]; ok {
		pc = ch[peerID]
		delete(ch, peerID)
		if len(ch) == 0 {
			delete(s.channels, channelKey)
		}
	}
	s.mu.Unlock()

	if pc != nil {
		pc.close()
	}
}

func (s *Server) sendPeerList(pc *peerConn, channelKey, peerID string) {
	s.mu.RLock()
	ch, ok := s.channels[channelKey]
	if !ok {
		s.mu.RUnlock()
		return
	}

	peers := make([]PeerEntry, 0, len(ch))
	for pid, p := range ch {
		if pid == peerID {
			continue
		}
		peers = append(peers, PeerEntry{
			UserID:    p.userID,
			PeerID:    p.peerID,
			Username:  p.username,
			AvatarURL: p.avatarURL,
		})
	}
	s.mu.RUnlock()

	sig, err := NewSignal(SignalPeerList, "", PeerListPayload{Peers: peers})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create peer list signal")
		return
	}

	if err := pc.enqueueJSON(sig); err != nil {
		s.dropPeer(channelKey, peerID)
	}
}

func (s *Server) broadcast(channelKey, excludePeerID string, signal *Signal) {
	s.mu.RLock()
	ch, ok := s.channels[channelKey]
	if !ok {
		s.mu.RUnlock()
		return
	}

	// Copy peer list to avoid holding lock during writes
	peers := make(map[string]*peerConn, len(ch))
	for pid, pc := range ch {
		if pid != excludePeerID {
			peers[pid] = pc
		}
	}
	s.mu.RUnlock()

	for pid, pc := range peers {
		if err := pc.enqueueJSON(signal); err != nil {
			s.dropPeer(channelKey, pid)
		}
	}
}

func (s *Server) forwardToPeer(channelKey, toPeerID string, signal *Signal) {
	s.mu.RLock()
	ch, ok := s.channels[channelKey]
	if !ok {
		s.mu.RUnlock()
		return
	}
	pc, ok := ch[toPeerID]
	s.mu.RUnlock()

	if !ok {
		s.logger.Debug().Str("peer", toPeerID).Msg("target peer not found")
		return
	}

	if err := pc.enqueueJSON(signal); err != nil {
		s.dropPeer(channelKey, toPeerID)
	}
}

func (s *Server) makeErrorSignal(code int, message string) *Signal {
	sig, _ := NewSignal(SignalError, "", ErrorPayload{Code: code, Message: message})
	return sig
}

// ChannelCount returns the number of active channels.
func (s *Server) ChannelCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.channels)
}

// PeerCount returns the total number of connected peers across all channels.
func (s *Server) PeerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, ch := range s.channels {
		count += len(ch)
	}
	return count
}

// GetChannelPeers returns the list of peers currently in a voice channel.
func (s *Server) GetChannelPeers(serverID, channelID string) []PeerEntry {
	key := serverID + ":" + channelID
	s.mu.RLock()
	ch, ok := s.channels[key]
	if !ok {
		s.mu.RUnlock()
		return nil
	}

	peers := make([]PeerEntry, 0, len(ch))
	for _, pc := range ch {
		peers = append(peers, PeerEntry{
			UserID:    pc.userID,
			PeerID:    pc.peerID,
			Username:  pc.username,
			AvatarURL: pc.avatarURL,
		})
	}
	s.mu.RUnlock()

	return peers
}

// splitChannelKey splits a "serverID:channelID" key into its parts.
func splitChannelKey(key string) [2]string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return [2]string{parts[0], parts[1]}
	}
	return [2]string{key, ""}
}

func (s *Server) dropPeer(channelKey, peerID string) {
	s.removePeer(channelKey, peerID)
	parts := splitChannelKey(channelKey)
	s.broadcast(channelKey, peerID, &Signal{
		Type:      SignalPeerLeft,
		From:      peerID,
		ServerID:  parts[0],
		ChannelID: parts[1],
	})

	s.logger.Warn().
		Str("peer_id", peerID).
		Str("channel", channelKey).
		Msg("peer dropped due to backpressure")
}

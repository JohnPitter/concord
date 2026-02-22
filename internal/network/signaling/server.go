package signaling

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for dev
}

// Server is a WebSocket signaling server that coordinates P2P connections.
type Server struct {
	mu       sync.RWMutex
	// channels maps "serverID:channelID" -> map of peerID -> connection
	channels map[string]map[string]*peerConn
	logger   zerolog.Logger
}

type peerConn struct {
	conn    *websocket.Conn
	userID  string
	peerID  string
	mu      sync.Mutex
}

// writeJSON serializes and writes a message, holding the per-connection mutex.
func (pc *peerConn) writeJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.conn.WriteMessage(websocket.TextMessage, data)
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
					_ = currentPC.writeJSON(s.makeErrorSignal(400, "invalid join payload"))
				}
				continue
			}

			currentChannel = channelKey
			currentPeerID = payload.PeerID
			currentPC = &peerConn{
				conn:   conn,
				userID: payload.UserID,
				peerID: payload.PeerID,
			}

			s.addPeer(channelKey, currentPC)

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

		case SignalOffer, SignalAnswer:
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, ok := s.channels[channelKey]; ok {
		delete(ch, peerID)
		if len(ch) == 0 {
			delete(s.channels, channelKey)
		}
	}
}

func (s *Server) sendPeerList(pc *peerConn, channelKey, excludePeerID string) {
	s.mu.RLock()
	ch, ok := s.channels[channelKey]
	if !ok {
		s.mu.RUnlock()
		return
	}

	peers := make([]PeerEntry, 0, len(ch))
	for pid, p := range ch {
		if pid == excludePeerID {
			continue
		}
		peers = append(peers, PeerEntry{
			UserID: p.userID,
			PeerID: p.peerID,
		})
	}
	s.mu.RUnlock()

	sig, err := NewSignal(SignalPeerList, "", PeerListPayload{Peers: peers})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create peer list signal")
		return
	}

	_ = pc.writeJSON(sig)
}

func (s *Server) broadcast(channelKey, excludePeerID string, signal *Signal) {
	s.mu.RLock()
	ch, ok := s.channels[channelKey]
	if !ok {
		s.mu.RUnlock()
		return
	}

	// Copy peer list to avoid holding lock during writes
	peers := make([]*peerConn, 0, len(ch))
	for pid, pc := range ch {
		if pid != excludePeerID {
			peers = append(peers, pc)
		}
	}
	s.mu.RUnlock()

	for _, pc := range peers {
		_ = pc.writeJSON(signal)
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

	_ = pc.writeJSON(signal)
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

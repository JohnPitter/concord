package voice

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog"

	"github.com/concord-chat/concord/internal/network/signaling"
)

// Orchestrator bridges the signaling client and voice engine.
// It manages the full lifecycle: WS connect → join → SDP/ICE negotiation → leave.
type Orchestrator struct {
	engine    *Engine
	sigClient *signaling.Client
	serverID  string
	channelID string
	userID    string
	peerID    string
	logger    zerolog.Logger
	mu        sync.Mutex
}

// NewOrchestrator creates a new voice orchestrator.
func NewOrchestrator(engine *Engine, logger zerolog.Logger) *Orchestrator {
	return &Orchestrator{
		engine: engine,
		logger: logger.With().Str("component", "voice-orchestrator").Logger(),
	}
}

// Join connects to the signaling server and begins peer negotiation.
func (o *Orchestrator) Join(ctx context.Context, wsURL, serverID, channelID, userID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sigClient != nil {
		return fmt.Errorf("voice: already in a voice channel")
	}

	o.serverID = serverID
	o.channelID = channelID
	o.userID = userID
	o.peerID = uuid.New().String()

	// Build WebSocket URL for signaling
	sigURL := buildSignalingURL(wsURL)

	o.logger.Info().
		Str("server_id", serverID).
		Str("channel_id", channelID).
		Str("peer_id", o.peerID).
		Str("ws_url", sigURL).
		Msg("joining voice channel via signaling")

	// Start voice engine for this channel
	if err := o.engine.JoinChannel(ctx, channelID); err != nil {
		return fmt.Errorf("voice: engine join: %w", err)
	}

	// Register ICE candidate callback on the engine
	o.engine.SetOnICECandidate(func(peerID string, candidate webrtc.ICECandidateInit) {
		o.mu.Lock()
		client := o.sigClient
		sID := o.serverID
		chID := o.channelID
		o.mu.Unlock()

		if client == nil {
			return
		}

		icePayload := signaling.ICECandidatePayload{
			Candidate: candidate.Candidate,
		}
		if candidate.SDPMid != nil {
			icePayload.SDPMid = *candidate.SDPMid
		}
		if candidate.SDPMLineIndex != nil {
			icePayload.SDPMLineIndex = *candidate.SDPMLineIndex
		}

		if err := client.SendICECandidate(sID, chID, peerID, icePayload); err != nil {
			o.logger.Warn().Err(err).Str("peer_id", peerID).Msg("failed to send ICE candidate")
		}
	})

	// Create signaling client and register handlers
	client := signaling.NewClient(sigURL, o.logger)
	o.registerHandlers(client)

	// Connect WebSocket
	if err := client.Connect(ctx); err != nil {
		_ = o.engine.LeaveChannel()
		return fmt.Errorf("voice: signaling connect: %w", err)
	}

	o.sigClient = client

	// Send join signal
	joinPayload := signaling.JoinPayload{
		UserID: userID,
		PeerID: o.peerID,
	}
	if err := client.JoinChannel(serverID, channelID, joinPayload); err != nil {
		_ = client.Close()
		_ = o.engine.LeaveChannel()
		o.sigClient = nil
		return fmt.Errorf("voice: send join: %w", err)
	}

	o.logger.Info().Msg("voice signaling join sent")
	return nil
}

// Leave disconnects from the voice channel and signaling server.
func (o *Orchestrator) Leave() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.sigClient == nil {
		// Not connected via signaling, just leave engine
		return o.engine.LeaveChannel()
	}

	o.logger.Info().
		Str("server_id", o.serverID).
		Str("channel_id", o.channelID).
		Msg("leaving voice channel")

	// Send leave signal
	_ = o.sigClient.LeaveChannel(o.serverID, o.channelID, o.userID)

	// Close signaling connection
	_ = o.sigClient.Close()
	o.sigClient = nil

	// Clear ICE callback
	o.engine.SetOnICECandidate(nil)

	// Leave voice engine (closes all peer connections)
	err := o.engine.LeaveChannel()

	o.serverID = ""
	o.channelID = ""
	o.userID = ""
	o.peerID = ""

	return err
}

// registerHandlers sets up signal handlers on the client.
func (o *Orchestrator) registerHandlers(client *signaling.Client) {
	// peer_list: received after joining — initiate offers to all existing peers
	client.On(signaling.SignalPeerList, func(sig *signaling.Signal) {
		var payload signaling.PeerListPayload
		if err := sig.DecodePayload(&payload); err != nil {
			o.logger.Warn().Err(err).Msg("invalid peer_list payload")
			return
		}

		o.logger.Info().Int("count", len(payload.Peers)).Msg("received peer list")

		for _, peer := range payload.Peers {
			o.initiatePeerOffer(peer.PeerID, peer.UserID)
		}
	})

	// peer_joined: a new peer joined — they will send us an offer, just add them
	client.On(signaling.SignalPeerJoined, func(sig *signaling.Signal) {
		var payload signaling.JoinPayload
		if err := sig.DecodePayload(&payload); err != nil {
			o.logger.Warn().Err(err).Msg("invalid peer_joined payload")
			return
		}

		o.logger.Info().
			Str("peer_id", payload.PeerID).
			Str("user_id", payload.UserID).
			Msg("peer joined — waiting for their offer")

		// Pre-add the peer so engine is ready when the offer arrives
		if err := o.engine.AddPeer(payload.PeerID, payload.UserID, payload.UserID); err != nil {
			o.logger.Error().Err(err).Str("peer_id", payload.PeerID).Msg("failed to add peer")
		}
	})

	// sdp_offer: remote peer sent us an offer — create answer
	client.On(signaling.SignalSDPOffer, func(sig *signaling.Signal) {
		var payload signaling.SDPPayload
		if err := sig.DecodePayload(&payload); err != nil {
			o.logger.Warn().Err(err).Msg("invalid sdp_offer payload")
			return
		}

		fromPeerID := sig.From
		o.logger.Debug().Str("from", fromPeerID).Msg("received SDP offer")

		// Ensure peer exists in engine
		_ = o.engine.AddPeer(fromPeerID, fromPeerID, fromPeerID)

		answerSDP, err := o.engine.HandleOffer(fromPeerID, payload.SDP)
		if err != nil {
			o.logger.Error().Err(err).Str("peer_id", fromPeerID).Msg("failed to handle offer")
			return
		}

		o.mu.Lock()
		client := o.sigClient
		sID := o.serverID
		chID := o.channelID
		o.mu.Unlock()

		if client != nil {
			if err := client.SendSDPAnswer(sID, chID, fromPeerID, answerSDP); err != nil {
				o.logger.Error().Err(err).Str("peer_id", fromPeerID).Msg("failed to send SDP answer")
			}
		}
	})

	// sdp_answer: remote peer answered our offer
	client.On(signaling.SignalSDPAnswer, func(sig *signaling.Signal) {
		var payload signaling.SDPPayload
		if err := sig.DecodePayload(&payload); err != nil {
			o.logger.Warn().Err(err).Msg("invalid sdp_answer payload")
			return
		}

		fromPeerID := sig.From
		o.logger.Debug().Str("from", fromPeerID).Msg("received SDP answer")

		if err := o.engine.HandleAnswer(fromPeerID, payload.SDP); err != nil {
			o.logger.Error().Err(err).Str("peer_id", fromPeerID).Msg("failed to handle answer")
		}
	})

	// ice_candidate: trickle ICE from remote peer
	client.On(signaling.SignalICECandidate, func(sig *signaling.Signal) {
		var payload signaling.ICECandidatePayload
		if err := sig.DecodePayload(&payload); err != nil {
			o.logger.Warn().Err(err).Msg("invalid ice_candidate payload")
			return
		}

		fromPeerID := sig.From
		candidate := webrtc.ICECandidateInit{
			Candidate: payload.Candidate,
		}
		if payload.SDPMid != "" {
			mid := payload.SDPMid
			candidate.SDPMid = &mid
		}
		if payload.SDPMLineIndex > 0 {
			idx := payload.SDPMLineIndex
			candidate.SDPMLineIndex = &idx
		}

		if err := o.engine.AddICECandidate(fromPeerID, candidate); err != nil {
			o.logger.Warn().Err(err).Str("peer_id", fromPeerID).Msg("failed to add ICE candidate")
		}
	})

	// peer_left: remote peer disconnected
	client.On(signaling.SignalPeerLeft, func(sig *signaling.Signal) {
		fromPeerID := sig.From
		o.logger.Info().Str("peer_id", fromPeerID).Msg("peer left voice channel")
		o.engine.RemovePeer(fromPeerID)
	})
}

// initiatePeerOffer adds a peer and creates + sends an SDP offer.
func (o *Orchestrator) initiatePeerOffer(peerID, userID string) {
	if err := o.engine.AddPeer(peerID, userID, userID); err != nil {
		o.logger.Error().Err(err).Str("peer_id", peerID).Msg("failed to add peer")
		return
	}

	sdp, err := o.engine.CreateOffer(peerID)
	if err != nil {
		o.logger.Error().Err(err).Str("peer_id", peerID).Msg("failed to create offer")
		return
	}

	o.mu.Lock()
	client := o.sigClient
	sID := o.serverID
	chID := o.channelID
	o.mu.Unlock()

	if client != nil {
		if err := client.SendSDPOffer(sID, chID, peerID, sdp); err != nil {
			o.logger.Error().Err(err).Str("peer_id", peerID).Msg("failed to send SDP offer")
		}
	}
}

// buildSignalingURL converts an HTTP URL to a WebSocket signaling URL.
func buildSignalingURL(baseURL string) string {
	u := baseURL
	u = strings.TrimSuffix(u, "/")

	if strings.HasPrefix(u, "https://") {
		u = "wss://" + strings.TrimPrefix(u, "https://")
	} else if strings.HasPrefix(u, "http://") {
		u = "ws://" + strings.TrimPrefix(u, "http://")
	} else if !strings.HasPrefix(u, "ws://") && !strings.HasPrefix(u, "wss://") {
		u = "ws://" + u
	}

	return u + "/ws/signaling"
}

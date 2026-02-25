package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/concord-chat/concord/internal/network/signaling"
	"github.com/concord-chat/concord/internal/voice"
)

// handleVoiceParticipants returns the list of users currently in a voice channel.
// GET /api/v1/servers/{serverID}/channels/{channelID}/voice/participants
func (s *Server) handleVoiceParticipants(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "serverID")
	channelID := chi.URLParam(r, "channelID")

	if s.signaling == nil {
		writeJSON(w, http.StatusOK, []signaling.PeerEntry{})
		return
	}

	peers := s.signaling.GetChannelPeers(serverID, channelID)
	if peers == nil {
		peers = []signaling.PeerEntry{}
	}

	writeJSON(w, http.StatusOK, peers)
}

// handleServerVoiceParticipants returns voice participants for all channels in a server.
// GET /api/v1/servers/{serverID}/voice/participants
func (s *Server) handleServerVoiceParticipants(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "serverID")

	if s.signaling == nil {
		writeJSON(w, http.StatusOK, map[string][]signaling.PeerEntry{})
		return
	}

	byChannel := s.signaling.GetServerChannelPeers(serverID)
	if byChannel == nil {
		byChannel = map[string][]signaling.PeerEntry{}
	}

	writeJSON(w, http.StatusOK, byChannel)
}

// handleVoiceICEConfig returns browser ICE server configuration for WebRTC.
// GET /api/v1/voice/ice-config
func (s *Server) handleVoiceICEConfig(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	requestHost := r.Host

	if s.iceProvider == nil || !s.iceProvider.Enabled() {
		// Without a dedicated TURN server, include a free public relay
		// so peers behind symmetric NATs can still connect.
		writeJSON(w, http.StatusOK, voice.ICEConfigResponse{
			Servers: []voice.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
				{URLs: []string{"stun:stun1.l.google.com:19302"}},
				{
					URLs:       []string{"turn:openrelay.metered.ca:80", "turn:openrelay.metered.ca:443", "turns:openrelay.metered.ca:443"},
					Username:   "openrelayproject",
					Credential: "openrelayproject",
				},
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, s.iceProvider.BuildConfig(userID, requestHost))
}

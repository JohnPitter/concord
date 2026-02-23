package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/concord-chat/concord/internal/network/signaling"
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

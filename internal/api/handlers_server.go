package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/concord-chat/concord/internal/server"
)

// createServerRequest is the expected body for POST /api/v1/servers.
type createServerRequest struct {
	Name string `json:"name"`
}

// createChannelRequest is the expected body for POST /api/v1/servers/{serverID}/channels.
type createChannelRequest struct {
	Name string `json:"name"`
	Type string `json:"type"` // "text" or "voice"
}

// updateServerRequest is the expected body for PUT /api/v1/servers/{serverID}.
type updateServerRequest struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}

// updateMemberRoleRequest is the expected body for PUT /api/v1/servers/{serverID}/members/{userID}/role.
type updateMemberRoleRequest struct {
	Role string `json:"role"` // "admin", "moderator", "member"
}

// handleListServers returns all servers the authenticated user belongs to.
// GET /api/v1/servers
// Complexity: O(n) where n is the number of user's servers
func (s *Server) handleListServers(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	servers, err := s.servers.ListUserServers(r.Context(), userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to list servers")
		writeError(w, http.StatusInternalServerError, "failed to list servers")
		return
	}

	if servers == nil {
		servers = []*server.Server{}
	}

	writeJSON(w, http.StatusOK, servers)
}

// handleCreateServer creates a new server owned by the authenticated user.
// POST /api/v1/servers
// Body: { "name": "My Server" }
// Complexity: O(1) (DB insert + default channel creation)
func (s *Server) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "server name is required")
		return
	}

	srv, err := s.servers.CreateServer(r.Context(), req.Name, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to create server")
		writeError(w, http.StatusInternalServerError, "failed to create server")
		return
	}

	writeJSON(w, http.StatusCreated, srv)
}

// handleGetServer retrieves a single server by ID.
// GET /api/v1/servers/{serverID}
// Complexity: O(1) — indexed lookup
func (s *Server) handleGetServer(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	srv, err := s.servers.GetServer(r.Context(), serverID)
	if err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to get server")
		writeError(w, http.StatusInternalServerError, "failed to get server")
		return
	}
	if srv == nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	writeJSON(w, http.StatusOK, srv)
}

// handleUpdateServer updates a server's name and icon.
// PUT /api/v1/servers/{serverID}
// Body: { "name": "New Name", "icon_url": "https://..." }
// Requires PermManageServer.
// Complexity: O(1)
func (s *Server) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	var req updateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "server name is required")
		return
	}

	if err := s.servers.UpdateServer(r.Context(), serverID, userID, req.Name, req.IconURL); err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to update server")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteServer deletes a server. Only the owner can delete.
// DELETE /api/v1/servers/{serverID}
// Complexity: O(n) where n is server data (channels, members, messages)
func (s *Server) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	if err := s.servers.DeleteServer(r.Context(), serverID, userID); err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to delete server")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListChannels returns all channels for a server.
// GET /api/v1/servers/{serverID}/channels
// Complexity: O(n) where n is the number of channels
func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	channels, err := s.servers.ListChannels(r.Context(), serverID)
	if err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to list channels")
		writeError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}

	if channels == nil {
		channels = []*server.Channel{}
	}

	writeJSON(w, http.StatusOK, channels)
}

// handleCreateChannel creates a new channel in a server.
// POST /api/v1/servers/{serverID}/channels
// Body: { "name": "general", "type": "text" }
// Requires PermManageChannels.
// Complexity: O(1)
func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	var req createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "channel name is required")
		return
	}
	if req.Type == "" {
		req.Type = "text"
	}

	ch, err := s.servers.CreateChannel(r.Context(), serverID, userID, req.Name, req.Type)
	if err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to create channel")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ch)
}

// handleListMembers returns all members of a server.
// GET /api/v1/servers/{serverID}/members
// Complexity: O(n) where n is the number of members
func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	members, err := s.servers.ListMembers(r.Context(), serverID)
	if err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to list members")
		writeError(w, http.StatusInternalServerError, "failed to list members")
		return
	}

	if members == nil {
		members = []*server.Member{}
	}

	writeJSON(w, http.StatusOK, members)
}

// handleKickMember removes a member from a server.
// DELETE /api/v1/servers/{serverID}/members/{userID}
// Requires PermManageMembers and higher role than target.
// Complexity: O(1)
func (s *Server) handleKickMember(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	actorID := UserIDFromContext(r.Context())
	serverID := chi.URLParam(r, "serverID")
	targetID := chi.URLParam(r, "userID")

	if serverID == "" || targetID == "" {
		writeError(w, http.StatusBadRequest, "server ID and user ID are required")
		return
	}

	if err := s.servers.KickMember(r.Context(), serverID, actorID, targetID); err != nil {
		s.logger.Error().Err(err).
			Str("server_id", serverID).
			Str("target_id", targetID).
			Msg("failed to kick member")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateMemberRole changes a member's role in a server.
// PUT /api/v1/servers/{serverID}/members/{userID}/role
// Body: { "role": "admin" }
// Requires PermManageMembers.
// Complexity: O(1)
func (s *Server) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	actorID := UserIDFromContext(r.Context())
	serverID := chi.URLParam(r, "serverID")
	targetID := chi.URLParam(r, "userID")

	if serverID == "" || targetID == "" {
		writeError(w, http.StatusBadRequest, "server ID and user ID are required")
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := server.Role(req.Role)
	if role != server.RoleAdmin && role != server.RoleModerator && role != server.RoleMember {
		writeError(w, http.StatusBadRequest, "role must be admin, moderator, or member")
		return
	}

	if err := s.servers.UpdateMemberRole(r.Context(), serverID, actorID, targetID, role); err != nil {
		s.logger.Error().Err(err).
			Str("server_id", serverID).
			Str("target_id", targetID).
			Str("role", req.Role).
			Msg("failed to update member role")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGenerateInvite creates a new invite code for a server.
// POST /api/v1/servers/{serverID}/invite
// Requires PermCreateInvite.
// Complexity: O(1)
func (s *Server) handleGenerateInvite(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	serverID := chi.URLParam(r, "serverID")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server ID is required")
		return
	}

	code, err := s.servers.GenerateInvite(r.Context(), serverID, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("server_id", serverID).Msg("failed to generate invite")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"invite_code": code})
}

// handleRedeemInvite adds the authenticated user to a server via invite code.
// POST /api/v1/invite/{code}/redeem
// Complexity: O(1)
func (s *Server) handleRedeemInvite(w http.ResponseWriter, r *http.Request) {
	if s.servers == nil {
		writeError(w, http.StatusServiceUnavailable, "server service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	code := chi.URLParam(r, "code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "invite code is required")
		return
	}

	srv, err := s.servers.RedeemInvite(r.Context(), code, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("code", code).Msg("failed to redeem invite")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, srv)
}

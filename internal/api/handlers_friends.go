package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/concord-chat/concord/internal/friends"
)

// sendFriendRequestBody is the expected body for POST /api/v1/friends/request.
type sendFriendRequestBody struct {
	Username string `json:"username"`
}

// sendDirectMessageBody is the expected body for POST /api/v1/friends/{friendID}/messages.
type sendDirectMessageBody struct {
	Content string `json:"content"`
}

// handleSendFriendRequest sends a friend request to a user by username.
// POST /api/v1/friends/request
// Body: { "username": "someone" }
func (s *Server) handleSendFriendRequest(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req sendFriendRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.friends.SendRequest(r.Context(), userID, req.Username); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("target", req.Username).Msg("failed to send friend request")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetPendingRequests returns all pending friend requests.
// GET /api/v1/friends/requests
func (s *Server) handleGetPendingRequests(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requests, err := s.friends.GetPendingRequests(r.Context(), userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get pending requests")
		writeError(w, http.StatusInternalServerError, "failed to get pending requests")
		return
	}

	writeJSON(w, http.StatusOK, requests)
}

// handleAcceptFriendRequest accepts a friend request.
// PUT /api/v1/friends/requests/{id}/accept
func (s *Server) handleAcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requestID := chi.URLParam(r, "requestID")
	if err := s.friends.AcceptRequest(r.Context(), requestID, userID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("request_id", requestID).Msg("failed to accept friend request")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRejectFriendRequest rejects/cancels a friend request.
// DELETE /api/v1/friends/requests/{id}
func (s *Server) handleRejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	requestID := chi.URLParam(r, "requestID")
	if err := s.friends.RejectRequest(r.Context(), requestID, userID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("request_id", requestID).Msg("failed to reject friend request")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetFriends returns all friends for the authenticated user.
// GET /api/v1/friends
func (s *Server) handleGetFriends(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendsList, err := s.friends.GetFriends(r.Context(), userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Msg("failed to get friends")
		writeError(w, http.StatusInternalServerError, "failed to get friends")
		return
	}

	writeJSON(w, http.StatusOK, friendsList)
}

// handleRemoveFriend removes a friendship.
// DELETE /api/v1/friends/{friendID}
func (s *Server) handleRemoveFriend(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendID := chi.URLParam(r, "friendID")
	if err := s.friends.RemoveFriend(r.Context(), userID, friendID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("friend_id", friendID).Msg("failed to remove friend")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleBlockUser blocks a user.
// POST /api/v1/friends/{friendID}/block
func (s *Server) handleBlockUser(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendID := chi.URLParam(r, "friendID")
	if err := s.friends.BlockUser(r.Context(), userID, friendID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("target_id", friendID).Msg("failed to block user")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUnblockUser unblocks a user.
// DELETE /api/v1/friends/{friendID}/block
func (s *Server) handleUnblockUser(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendID := chi.URLParam(r, "friendID")
	if err := s.friends.UnblockUser(r.Context(), userID, friendID); err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("target_id", friendID).Msg("failed to unblock user")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetDirectMessages returns direct messages with one friend.
// GET /api/v1/friends/{friendID}/messages
func (s *Server) handleGetDirectMessages(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendID := chi.URLParam(r, "friendID")
	if friendID == "" {
		writeError(w, http.StatusBadRequest, "friend ID is required")
		return
	}

	opts := friends.DMPaginationOpts{
		After: r.URL.Query().Get("after"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		opts.Limit = limit
	}

	messages, err := s.friends.GetDirectMessages(r.Context(), userID, friendID, opts)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("friend_id", friendID).Msg("failed to get direct messages")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, messages)
}

// handleSendDirectMessage sends a direct message to one friend.
// POST /api/v1/friends/{friendID}/messages
// Body: { "content": "Hello!" }
func (s *Server) handleSendDirectMessage(w http.ResponseWriter, r *http.Request) {
	if s.friends == nil {
		writeError(w, http.StatusServiceUnavailable, "friends service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	friendID := chi.URLParam(r, "friendID")
	if friendID == "" {
		writeError(w, http.StatusBadRequest, "friend ID is required")
		return
	}

	var req sendDirectMessageBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	msg, err := s.friends.SendDirectMessage(r.Context(), userID, friendID, req.Content)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID).Str("friend_id", friendID).Msg("failed to send direct message")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

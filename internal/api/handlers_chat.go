package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/concord-chat/concord/internal/chat"
)

// sendMessageRequest is the expected body for POST /api/v1/channels/{channelID}/messages.
type sendMessageRequest struct {
	Content string `json:"content"`
}

// editMessageRequest is the expected body for PUT /api/v1/messages/{messageID}.
type editMessageRequest struct {
	Content string `json:"content"`
}

// handleGetMessages retrieves messages for a channel with cursor-based pagination.
// GET /api/v1/channels/{channelID}/messages
// Query params: before, after, limit
// Complexity: O(log n) — indexed query with LIMIT
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service not available")
		return
	}

	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "channel ID is required")
		return
	}

	opts := chat.PaginationOpts{
		Before: r.URL.Query().Get("before"),
		After:  r.URL.Query().Get("after"),
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		opts.Limit = limit
	}

	messages, err := s.chat.GetMessages(r.Context(), channelID, opts)
	if err != nil {
		s.logger.Error().Err(err).Str("channel_id", channelID).Msg("failed to get messages")
		writeError(w, http.StatusInternalServerError, "failed to get messages")
		return
	}

	if messages == nil {
		messages = []*chat.Message{}
	}

	writeJSON(w, http.StatusOK, messages)
}

// handleSendMessage creates a new message in a channel.
// POST /api/v1/channels/{channelID}/messages
// Body: { "content": "Hello!" }
// Complexity: O(1) + O(log n) FTS index update
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "channel ID is required")
		return
	}

	var req sendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "message content is required")
		return
	}

	msg, err := s.chat.SendMessage(r.Context(), channelID, userID, req.Content)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channelID).
			Str("user_id", userID).
			Msg("failed to send message")
		writeError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

// handleEditMessage updates the content of an existing message.
// PUT /api/v1/messages/{messageID}
// Body: { "content": "Updated content" }
// Only the message author can edit.
// Complexity: O(1) + O(log n) FTS update
func (s *Server) handleEditMessage(w http.ResponseWriter, r *http.Request) {
	if s.chat == nil {
		writeError(w, http.StatusServiceUnavailable, "chat service not available")
		return
	}

	userID := UserIDFromContext(r.Context())
	messageID := chi.URLParam(r, "messageID")
	if messageID == "" {
		writeError(w, http.StatusBadRequest, "message ID is required")
		return
	}

	var req editMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "message content is required")
		return
	}

	msg, err := s.chat.EditMessage(r.Context(), messageID, userID, req.Content)
	if err != nil {
		s.logger.Error().Err(err).
			Str("message_id", messageID).
			Str("user_id", userID).
			Msg("failed to edit message")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, msg)
}

// handleDeleteMessage removes a message.
// DELETE /api/v1/messages/{messageID}
// Query: ?is_manager=true (optional, for permission-based delete)
// Complexity: O(1) + O(log n) FTS cleanup
func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	messageID := chi.URLParam(r, "messageID")
	if messageID == "" {
		writeError(w, http.StatusBadRequest, "message ID is required")
		return
	}

	isManager := r.URL.Query().Get("is_manager") == "true"

	if err := s.chat.DeleteMessage(r.Context(), messageID, userID, isManager); err != nil {
		s.logger.Error().Err(err).
			Str("message_id", messageID).
			Str("user_id", userID).
			Bool("is_manager", isManager).
			Msg("failed to delete message")
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleSearchMessages performs full-text search within a channel.
// GET /api/v1/channels/{channelID}/messages/search
// Query: q (search query), limit (max results)
// Complexity: O(log n) — FTS5 inverted index lookup
func (s *Server) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		writeError(w, http.StatusBadRequest, "channel ID is required")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "search query (q) is required")
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}

	results, err := s.chat.SearchMessages(r.Context(), channelID, query, limit)
	if err != nil {
		s.logger.Error().Err(err).
			Str("channel_id", channelID).
			Str("query", query).
			Msg("failed to search messages")
		writeError(w, http.StatusInternalServerError, "failed to search messages")
		return
	}

	if results == nil {
		results = []*chat.SearchResult{}
	}

	writeJSON(w, http.StatusOK, results)
}

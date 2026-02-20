package api

import (
	"encoding/json"
	"net/http"
)

// deviceCodeRequest is intentionally empty; StartLogin takes no parameters.
// Kept for forward-compatibility in case request body evolves.

// tokenRequest is the expected body for POST /api/v1/auth/token.
type tokenRequest struct {
	DeviceCode string `json:"device_code"`
	Interval   int    `json:"interval"`
}

// refreshRequest is the expected body for POST /api/v1/auth/refresh.
type refreshRequest struct {
	UserID string `json:"user_id"`
}

// handleDeviceCode initiates the GitHub Device Flow.
// POST /api/v1/auth/device-code
// Returns a DeviceCodeResponse with user_code and verification_uri.
// Complexity: O(1) — single upstream HTTP call
func (s *Server) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service not available")
		return
	}

	resp, err := s.auth.StartLogin(r.Context())
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to start device code flow")
		writeError(w, http.StatusInternalServerError, "failed to initiate device code flow")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleToken exchanges a device code for an access token (polls GitHub).
// POST /api/v1/auth/token
// Body: { "device_code": "...", "interval": 5 }
// Returns an AuthState with access_token and user info.
// Complexity: O(n) where n = expires_in / interval (polling loop)
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service not available")
		return
	}

	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DeviceCode == "" {
		writeError(w, http.StatusBadRequest, "device_code is required")
		return
	}

	if req.Interval < 5 {
		req.Interval = 5
	}

	state, err := s.auth.CompleteLogin(r.Context(), req.DeviceCode, req.Interval)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to complete login")
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}

	writeJSON(w, http.StatusOK, state)
}

// handleRefresh restores a session using a stored refresh token.
// POST /api/v1/auth/refresh
// Body: { "user_id": "..." }
// Returns an AuthState with a new access_token if the session is valid.
// Complexity: O(1)
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service not available")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	state, err := s.auth.RestoreSession(r.Context(), req.UserID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to restore session")
		writeError(w, http.StatusUnauthorized, "session restoration failed")
		return
	}

	writeJSON(w, http.StatusOK, state)
}

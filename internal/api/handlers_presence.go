package api

import (
	"net/http"
	"strings"
)

// handleSetPresenceOffline marks the authenticated user as offline immediately.
// POST /api/v1/presence/offline
//
// Accepts authentication via:
//  1. Standard Authorization header (normal API calls)
//  2. ?token= query param (navigator.sendBeacon fallback — sendBeacon cannot set custom headers)
func (s *Server) handleSetPresenceOffline(w http.ResponseWriter, r *http.Request) {
	if s.presence == nil {
		writeError(w, http.StatusServiceUnavailable, "presence tracker not available")
		return
	}

	// Try context first (set by AuthMiddleware from Authorization header).
	userID := UserIDFromContext(r.Context())

	// Fallback: extract JWT from ?token= query param (sendBeacon path).
	if userID == "" {
		if tokenParam := strings.TrimSpace(r.URL.Query().Get("token")); tokenParam != "" {
			if jwtMgr := s.jwtManager(); jwtMgr != nil {
				if claims, err := jwtMgr.ValidateToken(tokenParam); err == nil {
					userID = claims.UserID
				}
			}
		}
	}

	if userID == "" {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	s.presence.SetOffline(userID)
	w.WriteHeader(http.StatusNoContent)
}

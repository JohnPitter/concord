package api

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

// errorResponse wraps API errors in a consistent JSON structure.
type errorResponse struct {
	Error errorDetail `json:"error"`
}

// errorDetail contains the error code and human-readable message.
type errorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// writeJSON serializes data as JSON and writes it to the response writer.
// Complexity: O(n) where n is the serialized size of data
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// At this point the header is already written, so we can only log the error.
		// The caller should ensure data is JSON-serializable.
		zerolog.DefaultContextLogger.Error().Err(err).Msg("failed to encode JSON response")
	}
}

// writeError writes a structured error response.
// Complexity: O(1)
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{
		Error: errorDetail{
			Code:    status,
			Message: message,
		},
	})
}

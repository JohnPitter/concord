package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/config"
)

// contextKey is a private type used for context value keys to prevent collisions.
type contextKey string

const userIDKey contextKey = "user_id"

// UserIDFromContext extracts the authenticated user ID from the request context.
// Returns an empty string if no user ID is present.
// Complexity: O(1)
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

// AuthMiddleware validates the Bearer JWT token from the Authorization header
// and injects the user_id into the request context.
// Requests without a valid token receive a 401 Unauthorized response.
// Complexity: O(1) per request (JWT validation is constant time)
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			claims, err := jwtManager.ValidateToken(parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing headers.
// It responds to preflight OPTIONS requests and sets appropriate CORS headers.
// Complexity: O(n) where n is the number of allowed origins (checked per request)
func CORSMiddleware(cfg config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			allowed := false
			for _, o := range cfg.AllowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs each request with method, path, status code, and duration
// using structured zerolog output.
// Complexity: O(1) per request
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.statusCode).
				Dur("duration_ms", duration).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("http request")
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// SecurityHeaders adds standard security headers to every response.
// Complexity: O(1) per request
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitEntry tracks request timestamps for a single IP address.
type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

// RateLimitMiddleware implements a simple in-memory sliding-window rate limiter per IP.
// The rps parameter controls the maximum requests allowed per second.
// Complexity: O(1) per request (sync.Map lookup + atomic counter)
func RateLimitMiddleware(rps int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limits := make(map[string]*rateLimitEntry)

	// Background cleanup every 60 seconds to prevent memory leaks.
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			mu.Lock()
			for ip, entry := range limits {
				if now.After(entry.windowEnd) {
					delete(limits, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			// Use X-Forwarded-For if present (behind reverse proxy)
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = strings.SplitN(forwarded, ",", 2)[0]
				ip = strings.TrimSpace(ip)
			}

			now := time.Now()
			mu.Lock()
			entry, exists := limits[ip]
			if !exists || now.After(entry.windowEnd) {
				limits[ip] = &rateLimitEntry{
					count:     1,
					windowEnd: now.Add(time.Second),
				}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			entry.count++
			if entry.count > rps {
				mu.Unlock()
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize limits the size of the request body.
// Requests exceeding maxBytes receive a 413 Payload Too Large response.
// Complexity: O(1) per request
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/concord-chat/concord/internal/auth"
	"github.com/concord-chat/concord/internal/config"
	"github.com/concord-chat/concord/internal/observability"
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
			w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self' ws: wss:")
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

// MetricsMiddleware collects HTTP request metrics (total, duration, response size)
// using the pre-defined Prometheus metrics from observability.Metrics.
// Complexity: O(1) per request
func MetricsMiddleware(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			mw := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(mw, r)

			duration := time.Since(start).Milliseconds()
			status := strconv.Itoa(mw.statusCode)

			// Normalize path to prevent cardinality explosion
			// e.g. /api/v1/servers/abc123 → /api/v1/servers/{id}
			path := normalizePath(r.URL.Path)

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(float64(duration))
			metrics.HTTPResponseSize.WithLabelValues(r.Method, path).Observe(float64(mw.bytesWritten))
		})
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status and bytes written.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	headerSent   bool
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if !w.headerSent {
		w.statusCode = code
		w.headerSent = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.headerSent {
		w.headerSent = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// normalizePath replaces dynamic path segments with placeholders
// to prevent Prometheus label cardinality explosion.
func normalizePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		// Replace UUIDs and numeric IDs with {id}
		if len(part) >= 8 && !isStaticSegment(part) {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

// isStaticSegment returns true for known static route segments.
func isStaticSegment(s string) bool {
	switch s {
	case "api", "v1", "auth", "servers", "channels", "members",
		"messages", "invite", "health", "metrics", "device-code",
		"token", "refresh", "search", "role":
		return true
	}
	return false
}

// RateLimitWithHeaders implements rate limiting with standard headers.
// Uses config-driven RPS and adds X-RateLimit-* headers.
// Complexity: O(1) per request
func RateLimitWithHeaders(rps int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limits := make(map[string]*rateLimitEntry)

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
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = strings.TrimSpace(strings.SplitN(forwarded, ",", 2)[0])
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
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rps-1))
				next.ServeHTTP(w, r)
				return
			}

			entry.count++
			remaining := rps - entry.count
			if remaining < 0 {
				remaining = 0
			}
			resetAt := entry.windowEnd.Unix()

			if entry.count > rps {
				mu.Unlock()
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rps))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))
			next.ServeHTTP(w, r)
		})
	}
}

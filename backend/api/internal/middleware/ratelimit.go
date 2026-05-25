package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// IncrFunc increments the fixed-window counter for key and returns the resulting
// count within the current window. The implementation is responsible for setting
// the window TTL on first use (count == 1).
type IncrFunc func(ctx context.Context, key string) (int, error)

// RateLimit applies a per-client-IP fixed-window rate limit to a single handler.
// `name` namespaces the counter so endpoints don't share a budget; `limit` is the
// maximum number of requests allowed per window. On exceedance it responds 429.
//
// If the backing store errors, the request is allowed through (fail-open) so a
// transient Redis blip cannot lock everyone out of authentication.
func RateLimit(name string, limit int, incr IncrFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := "ratelimit:" + name + ":" + clientIP(r)
			count, err := incr(r.Context(), key)
			if err == nil && count > limit {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}
}

// clientIP extracts the caller's IP. X-Forwarded-For / X-Real-IP are honored
// because the production deployment runs behind a trusted nginx reverse proxy;
// in local dev (no proxy) RemoteAddr is used.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

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
// maximum number of requests allowed per window. `trustProxy` controls whether
// forwarded headers are honored when identifying the client (see clientIP). On
// exceedance it responds 429.
//
// If the backing store errors, the request is allowed through (fail-open) so a
// transient store hiccup cannot lock everyone out of authentication.
func RateLimit(name string, limit int, trustProxy bool, incr IncrFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := "ratelimit:" + name + ":" + clientIP(r, trustProxy)
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

// clientIP extracts the caller's IP for rate-limiting.
//
// X-Forwarded-For and X-Real-IP are attacker-controlled: a client talking to the
// server directly can set them to any value. Honoring them unconditionally lets
// an attacker rotate fake IPs to defeat the per-IP limit (unlimited credential
// guessing). They are therefore trusted only when trustProxy is true — i.e. the
// deployment sets TRUST_PROXY because a reverse proxy (nginx) sanitises them.
//
// When trusted we prefer X-Real-IP (nginx sets it to the real transport peer,
// overwriting any client value), then fall back to the LAST X-Forwarded-For hop
// (nginx appends the real peer, so earlier, client-supplied entries are
// spoofable). When not trusted, only the transport peer (RemoteAddr) is used.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
			return xr
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
				return last
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

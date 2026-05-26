package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewServerHasTimeouts guards the Slowloris hardening: the API server must
// never fall back to net/http's unbounded (zero-value) timeouts. It fails if a
// future refactor drops any of the four timeouts.
func TestNewServerHasTimeouts(t *testing.T) {
	srv := newServer(":0", http.NewServeMux())

	cases := []struct {
		name string
		got  time.Duration
	}{
		{"ReadHeaderTimeout", srv.ReadHeaderTimeout},
		{"ReadTimeout", srv.ReadTimeout},
		{"WriteTimeout", srv.WriteTimeout},
		{"IdleTimeout", srv.IdleTimeout},
	}
	for _, c := range cases {
		if c.got <= 0 {
			t.Errorf("%s must be > 0 to guard against Slowloris; got %v", c.name, c.got)
		}
	}
}

// TestSecurityHeadersCSPNoUnsafeInline guards that the API-wide CSP never
// permits inline styles. The API serves JSON only, so 'unsafe-inline' adds
// attack surface (HTML/SVG error responses, future regressions) without any
// upside.
func TestSecurityHeadersCSPNoUnsafeInline(t *testing.T) {
	h := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/health", nil))

	csp := rr.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header is missing")
	}
	if strings.Contains(csp, "'unsafe-inline'") {
		t.Errorf("CSP must not allow 'unsafe-inline'; got %q", csp)
	}
}

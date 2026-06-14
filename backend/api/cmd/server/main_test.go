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

// TestUploadsSecurityHeaders guards the /uploads/ hardening: user-uploaded files
// are served with MIME sniffing disabled and a locked-down CSP that forbids any
// active content, neutralizing a malicious payload that slips past upload
// validation.
func TestUploadsSecurityHeaders(t *testing.T) {
	h := uploadsSecurity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/uploads/x.png", nil))

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	const wantCSP = "default-src 'none'; sandbox"
	if got := rr.Header().Get("Content-Security-Policy"); got != wantCSP {
		t.Errorf("Content-Security-Policy = %q, want %q", got, wantCSP)
	}
}

// TestCORSReflectsOnlyAllowlistedOrigin guards the credentialed-CORS contract:
// the server reflects an Origin into Access-Control-Allow-Origin ONLY when it is
// on the allowlist. Reflecting an arbitrary origin alongside
// Allow-Credentials:true would let any site make authenticated cross-origin
// requests.
func TestCORSReflectsOnlyAllowlistedOrigin(t *testing.T) {
	old := allowedOrigins
	t.Cleanup(func() { allowedOrigins = old })
	allowedOrigins = map[string]bool{"https://store.example": true}

	dummy := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

	// (a) allowlisted origin → reflected, credentials allowed, Vary: Origin.
	rrOK := httptest.NewRecorder()
	reqOK := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	reqOK.Header.Set("Origin", "https://store.example")
	cors(dummy)(rrOK, reqOK)

	if got := rrOK.Header().Get("Access-Control-Allow-Origin"); got != "https://store.example" {
		t.Errorf("allowlisted ACAO = %q, want https://store.example", got)
	}
	if got := rrOK.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if got := rrOK.Header().Get("Vary"); !strings.Contains(got, "Origin") {
		t.Errorf("Vary = %q, want it to contain Origin", got)
	}

	// (b) non-allowlisted origin → NOT reflected (header absent ⇒ empty string).
	rrEvil := httptest.NewRecorder()
	reqEvil := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	reqEvil.Header.Set("Origin", "https://evil.example")
	cors(dummy)(rrEvil, reqEvil)

	if got := rrEvil.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("non-allowlisted ACAO = %q, want empty (origin must not be reflected)", got)
	}
}

// TestCORSPreflightReturns204 guards that an OPTIONS preflight short-circuits in
// the cors wrapper: it returns 204 without ever invoking the wrapped handler.
// Mirrors the OPTIONS /api/ route's behavior.
func TestCORSPreflightReturns204(t *testing.T) {
	called := false
	h := cors(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest(http.MethodOptions, "/api/products", nil))

	if rr.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", rr.Code)
	}
	if called {
		t.Error("preflight must short-circuit before the wrapped handler, but it was called")
	}
}

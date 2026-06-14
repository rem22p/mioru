package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		trustProxy bool
		remoteAddr string
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name:       "untrusted ignores forwarded headers",
			trustProxy: false,
			remoteAddr: "203.0.113.7:54321",
			xff:        "1.1.1.1, 2.2.2.2",
			xRealIP:    "9.9.9.9",
			want:       "203.0.113.7",
		},
		{
			name:       "untrusted strips port from RemoteAddr",
			trustProxy: false,
			remoteAddr: "198.51.100.4:1234",
			want:       "198.51.100.4",
		},
		{
			name:       "trusted prefers X-Real-IP",
			trustProxy: true,
			remoteAddr: "127.0.0.1:8000",
			xff:        "1.1.1.1, 5.5.5.5",
			xRealIP:    "5.5.5.5",
			want:       "5.5.5.5",
		},
		{
			name:       "trusted uses last XFF hop when no X-Real-IP",
			trustProxy: true,
			remoteAddr: "127.0.0.1:8000",
			xff:        "1.1.1.1, 2.2.2.2, 5.5.5.5",
			want:       "5.5.5.5",
		},
		{
			name:       "trusted but no forwarded headers falls back to RemoteAddr",
			trustProxy: true,
			remoteAddr: "203.0.113.9:40000",
			want:       "203.0.113.9",
		},
		{
			name:       "RemoteAddr without port returned as-is",
			trustProxy: false,
			remoteAddr: "203.0.113.9",
			want:       "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				r.Header.Set("X-Real-IP", tt.xRealIP)
			}
			if got := clientIP(r, tt.trustProxy); got != tt.want {
				t.Errorf("clientIP(trustProxy=%v) = %q, want %q", tt.trustProxy, got, tt.want)
			}
		})
	}
}

// TestRateLimitEnvelope pins the JSON envelope + RATE_LIMITED machine code on
// the 429 path (fix for #31 — was http.Error text/plain). Without the
// envelope the SPA cannot distinguish rate-limited from any other 4xx and
// the generic-error path swallows retry logic.
func TestRateLimitEnvelope(t *testing.T) {
	const limit = 3
	over := func(_ context.Context, _ string) (int, error) { return limit + 1, nil }
	rl := RateLimit("t-envelope", limit, false, over)

	called := false
	h := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	h.ServeHTTP(rr, req)

	if called {
		t.Fatal("handler should not have been called when limit is exceeded")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rr.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want 60", got)
	}
	var env struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body: %v (body %q)", err, rr.Body.String())
	}
	if env.Code != "RATE_LIMITED" {
		t.Errorf("code = %q, want RATE_LIMITED", env.Code)
	}
}

// TestRateLimitUnderLimitPasses ensures the happy path: when count <= limit,
// the handler runs and the middleware is invisible. Guards against an
// over-broad fix that would 429 too eagerly.
func TestRateLimitUnderLimitPasses(t *testing.T) {
	const limit = 5
	under := func(_ context.Context, _ string) (int, error) { return limit, nil }
	rl := RateLimit("t-under", limit, false, under)

	called := false
	h := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	h.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler should have been called at the limit boundary")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

// TestRateLimitFailOpenOnStoreError covers the documented behaviour: if the
// backing store errors, requests are allowed through (fail-open) so a
// transient store hiccup cannot lock everyone out of authentication. The
// envelope must NOT be returned on the fail-open path — the handler runs.
func TestRateLimitFailOpenOnStoreError(t *testing.T) {
	erring := func(_ context.Context, _ string) (int, error) { return 0, context.DeadlineExceeded }
	rl := RateLimit("t-failopen", 1, false, erring)

	called := false
	h := rl(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	h.ServeHTTP(rr, req)

	if !called {
		t.Fatal("handler should run on store error (fail-open)")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fail-open path)", rr.Code)
	}
}

package main

import (
	"net/http"
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

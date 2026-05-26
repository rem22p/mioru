package middleware

import (
	"net/http"
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

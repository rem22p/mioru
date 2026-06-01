package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name     string
		getRole  func(context.Context, string) (string, error)
		wantCode int
	}{
		{
			name: "admin passes",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "admin", nil
			},
			wantCode: http.StatusOK,
		},
		{
			name: "super_admin passes",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "super_admin", nil
			},
			wantCode: http.StatusOK,
		},
		{
			name: "customer is forbidden",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "customer", nil
			},
			wantCode: http.StatusForbidden,
		},
		{
			name: "unknown role forbidden",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "viewer", nil
			},
			wantCode: http.StatusForbidden,
		},
		{
			name: "getRole error is 500",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "", context.DeadlineExceeded
			},
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			mw := RequireAdmin(tt.getRole)(next)

			req := httptest.NewRequest("GET", "/", nil)
			ctx := context.WithValue(req.Context(), ctxKey{}, "testuser")
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			mw.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestRequireAdminNoUsername(t *testing.T) {
	getRole := func(ctx context.Context, user string) (string, error) {
		return "admin", nil
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireAdmin(getRole)(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

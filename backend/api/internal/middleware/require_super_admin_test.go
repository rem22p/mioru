package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireSuperAdmin(t *testing.T) {
	tests := []struct {
		name     string
		getRole  func(context.Context, string) (string, error)
		wantCode int
	}{
		{
			name: "super_admin passes",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "super_admin", nil
			},
			wantCode: http.StatusOK,
		},
		{
			name: "admin is forbidden",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "admin", nil
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			mw := RequireSuperAdmin(tt.getRole)(next)

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

func TestRequireSuperAdminNoUsername(t *testing.T) {
	// Without AuthMW setting the username, RequireSuperAdmin should 401.
	getRole := func(ctx context.Context, user string) (string, error) {
		return "super_admin", nil
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := RequireSuperAdmin(getRole)(next)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

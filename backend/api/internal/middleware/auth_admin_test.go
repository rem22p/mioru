package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAdmin(t *testing.T) {
	tests := []struct {
		name       string
		getRole    func(context.Context, string) (string, error)
		wantCode   int
		wantErrCode string // empty when 200 expected
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
			wantCode:    http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name: "unknown role forbidden",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "viewer", nil
			},
			wantCode:    http.StatusForbidden,
			wantErrCode: "FORBIDDEN",
		},
		{
			name: "getRole error is 500",
			getRole: func(ctx context.Context, user string) (string, error) {
				return "", context.DeadlineExceeded
			},
			wantCode:    http.StatusInternalServerError,
			wantErrCode: "INTERNAL",
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

			if tt.wantErrCode != "" {
				assertJSONError(t, rr, tt.wantErrCode)
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
	assertJSONError(t, rr, "AUTH_REQUIRED")
}

// assertJSONError checks Content-Type is application/json and the body contains
// the expected machine "code".
func assertJSONError(t *testing.T, rr *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Code != wantCode {
		t.Errorf("code = %q, want %q", body.Code, wantCode)
	}
}

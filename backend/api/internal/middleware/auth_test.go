package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mioru/internal/auth"
)

const mwTestSecret = "test-secret-at-least-32-bytes-long-xxxxx"

// epochOK builds a UserEpochFunc that always reports the user exists with the
// given password-changed-at timestamp.
func epochOK(at time.Time) UserEpochFunc {
	return func(context.Context, string) (time.Time, bool, error) {
		return at, true, nil
	}
}

func userTokenReq(t *testing.T, sub string) *http.Request {
	t.Helper()
	tok, err := auth.CreateToken(sub, auth.TokenTypeUser, mwTestSecret, 60)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	return r
}

// okHandler records whether the protected handler was reached.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthMWSessionRevocation(t *testing.T) {
	tests := []struct {
		name      string
		epoch     UserEpochFunc
		wantCode  int
		wantReach bool
	}{
		{
			// Password changed an hour ago; a token minted now is newer → allowed.
			name:      "fresh token passes",
			epoch:     epochOK(time.Now().Add(-time.Hour)),
			wantCode:  http.StatusOK,
			wantReach: true,
		},
		{
			// Password changed in the future relative to the token's iat → the token
			// predates the change and must be rejected (session revoked).
			name:      "stale token rejected",
			epoch:     epochOK(time.Now().Add(time.Hour)),
			wantCode:  http.StatusUnauthorized,
			wantReach: false,
		},
		{
			// User no longer exists → reject.
			name: "deleted user rejected",
			epoch: func(context.Context, string) (time.Time, bool, error) {
				return time.Time{}, false, nil
			},
			wantCode:  http.StatusUnauthorized,
			wantReach: false,
		},
		{
			// Epoch lookup failed → 500, never silently allow.
			name: "lookup error is 500",
			epoch: func(context.Context, string) (time.Time, bool, error) {
				return time.Time{}, false, context.DeadlineExceeded
			},
			wantCode:  http.StatusInternalServerError,
			wantReach: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			h := AuthMW(mwTestSecret, tt.epoch)(okHandler(&reached))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, userTokenReq(t, "admin"))

			if rr.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantCode)
			}
			if reached != tt.wantReach {
				t.Errorf("handler reached = %v, want %v", reached, tt.wantReach)
			}
		})
	}
}

func TestCustomerAuthMWSessionRevocation(t *testing.T) {
	tests := []struct {
		name      string
		epoch     CustomerEpochFunc
		wantCode  int
		wantReach bool
	}{
		{
			name: "fresh token passes",
			epoch: func(context.Context, int64) (time.Time, bool, error) {
				return time.Now().Add(-time.Hour), true, nil
			},
			wantCode:  http.StatusOK,
			wantReach: true,
		},
		{
			name: "stale token rejected",
			epoch: func(context.Context, int64) (time.Time, bool, error) {
				return time.Now().Add(time.Hour), true, nil
			},
			wantCode:  http.StatusUnauthorized,
			wantReach: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := auth.CreateToken("42", auth.TokenTypeCustomer, mwTestSecret, 60)
			if err != nil {
				t.Fatalf("CreateToken: %v", err)
			}
			r := httptest.NewRequest(http.MethodGet, "/api/store/customers/me", nil)
			r.Header.Set("Authorization", "Bearer "+tok)

			reached := false
			h := CustomerAuthMW(mwTestSecret, tt.epoch)(okHandler(&reached))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)

			if rr.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantCode)
			}
			if reached != tt.wantReach {
				t.Errorf("handler reached = %v, want %v", reached, tt.wantReach)
			}
		})
	}
}

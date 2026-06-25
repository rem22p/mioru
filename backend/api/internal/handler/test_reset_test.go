//go:build e2e
// +build e2e

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"mioru/internal/model"
)

// fakeAdminStore is a minimal stub of *store.PostgresStore for testing
// the reset handler in isolation. Records the last call so we can assert
// it round-trips.
type fakeAdminStore struct {
	lastUser model.User
	lastTime time.Time
	err      error
}

func (f *fakeAdminStore) ResetAdminForTest(_ context.Context, u model.User, t time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.lastUser = u
	f.lastTime = t
	return nil
}

func newReq(method, body string) *http.Request {
	r := httptest.NewRequest(method, "/api/_test/reset-admin", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func do(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

// TestResetAdminRejectsMissingKey verifies that without the X-E2E-Reset-Key
// header the handler returns 403 and never invokes the store — critical
// because the route lives behind only a build tag (no auth middleware) and
// would otherwise be an unauthenticated admin-takeover primitive.
func TestResetAdminRejectsMissingKey(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeAdminStore{}
	h := NewTestResetAdminHandler(store)

	req := newReq("POST", `{"username":"x","hashed_password":"y"}`)
	w := do(h, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403 (missing X-E2E-Reset-Key)", w.Code)
	}
	if store.lastUser.Username != "" {
		t.Fatalf("store was invoked with %q on missing key — auth gate failed",
			store.lastUser.Username)
	}
}

// TestResetAdminRejectsWrongKey verifies constant-time mismatch behavior.
func TestResetAdminRejectsWrongKey(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeAdminStore{}
	h := NewTestResetAdminHandler(store)

	req := newReq("POST", `{"username":"x","hashed_password":"y"}`)
	req.Header.Set("X-E2E-Reset-Key", "wrong-secret")
	w := do(h, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403 (wrong X-E2E-Reset-Key)", w.Code)
	}
	if store.lastUser.Username != "" {
		t.Fatalf("store was invoked with %q on wrong key", store.lastUser.Username)
	}
}

// TestResetAdminRequiresServerSideKey verifies the fail-closed path: if
// E2E_RESET_KEY is unset on the server, the handler returns 503 even
// when the client provides a key. (We never want a misconfigured server
// to fall back to "no auth" — that's how the original Blocker arose.)
func TestResetAdminRequiresServerSideKey(t *testing.T) {
	os.Unsetenv("E2E_RESET_KEY")

	store := &fakeAdminStore{}
	h := NewTestResetAdminHandler(store)

	req := newReq("POST", `{"username":"x","hashed_password":"y"}`)
	req.Header.Set("X-E2E-Reset-Key", "client-supplied")
	w := do(h, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503 (no server-side E2E_RESET_KEY)", w.Code)
	}
	if store.lastUser.Username != "" {
		t.Fatalf("store was invoked with %q despite unset server-side key",
			store.lastUser.Username)
	}
}

// TestResetAdminHappyPath verifies a correctly-keyed request reaches the
// store with the expected fields and a past password_changed_at default.
func TestResetAdminHappyPath(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeAdminStore{}
	h := NewTestResetAdminHandler(store)

	body := `{
		"username": "admin",
		"hashed_password": "$2a$12$abcdef",
		"email": "admin@mioru.store",
		"role": "super_admin"
	}`
	req := newReq("POST", body)
	req.Header.Set("X-E2E-Reset-Key", "ci-secret-1234567890abcdef")
	w := do(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if got := decode(t, w)["ok"]; got != true {
		t.Fatalf("body.ok: got %v, want true", got)
	}
	if store.lastUser.Username != "admin" {
		t.Errorf("store username: got %q, want admin", store.lastUser.Username)
	}
	if store.lastUser.HashedPW != "$2a$12$abcdef" {
		t.Errorf("store hashed_password: got %q, want $2a$12$abcdef", store.lastUser.HashedPW)
	}
	if store.lastUser.Role != "super_admin" {
		t.Errorf("store role: got %q, want super_admin", store.lastUser.Role)
	}
	// Default password_changed_at is "1 hour in the past" — must be < now
	// so the next login's iat > changed_at is guaranteed.
	cutoff := time.Now().Add(-30 * time.Minute)
	if !store.lastTime.Before(cutoff) {
		t.Errorf("password_changed_at default: got %v, want < %v (i.e. at least 30min in the past)",
			store.lastTime, cutoff)
	}
}

// TestResetAdminErrorPathDoesNotLeak verifies that when the store fails,
// the response body contains only the generic envelope (no err.Error()).
// The full error detail must go to the server log (slog), not the client.
func TestResetAdminErrorPathDoesNotLeak(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeAdminStore{err: errors.New("internal-db-thing-12345")}
	h := NewTestResetAdminHandler(store)

	req := newReq("POST", `{"username":"x","hashed_password":"y"}`)
	req.Header.Set("X-E2E-Reset-Key", "ci-secret-1234567890abcdef")
	w := do(h, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "internal-db-thing-12345") {
		t.Fatalf("response leaked internal error detail: %s", body)
	}
	bodyMap := decode(t, w)
	if bodyMap["code"] != "INTERNAL" {
		t.Errorf("code: got %v, want INTERNAL", bodyMap["code"])
	}
	if bodyMap["error"] != "internal" {
		t.Errorf("error: got %v, want \"internal\"", bodyMap["error"])
	}
}

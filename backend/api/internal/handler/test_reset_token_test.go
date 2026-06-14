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

// fakeResetTokenStore is a minimal stub of the parts of *store.PostgresStore
// that TestCreateResetTokenHandler needs. Records every call so tests can
// assert on the round-trip, and can be configured to return errors.
type fakeResetTokenStore struct {
	// getUserFn is called by GetUser. Default returns (adminUser, nil).
	getUserFn func(ctx context.Context, username string) (*model.User, error)
	// createResetTokenFn is called by CreateResetToken. Default records
	// the username + token and returns nil.
	createResetTokenFn func(ctx context.Context, username, rawToken string) error
	// recordedToken is the raw token most recently passed to
	// CreateResetToken, for round-trip assertions.
	recordedToken string
	// recordedUsername is the username most recently passed to
	// CreateResetToken.
	recordedUsername string
	// callCountCreate is the number of times CreateResetToken was
	// invoked — proves the store is or isn't called.
	callCountCreate int
}

func (f *fakeResetTokenStore) GetUser(ctx context.Context, username string) (*model.User, error) {
	if f.getUserFn != nil {
		return f.getUserFn(ctx, username)
	}
	return &model.User{Username: username, Email: username + "@mioru.store", Role: "super_admin"}, nil
}

func (f *fakeResetTokenStore) CreateResetToken(ctx context.Context, username, rawToken string) error {
	f.callCountCreate++
	f.recordedUsername = username
	f.recordedToken = rawToken
	if f.createResetTokenFn != nil {
		return f.createResetTokenFn(ctx, username, rawToken)
	}
	return nil
}

func newResetTokenReq(method, body string) *http.Request {
	r := httptest.NewRequest(method, "/api/_test/create-reset-token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func doResetToken(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func decodeResetToken(t *testing.T, w *httptest.ResponseRecorder) createResetTokenResponse {
	t.Helper()
	var body createResetTokenResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

// TestCreateResetTokenRejectsMissingKey verifies that without the
// X-E2E-Reset-Key header the handler returns 403 and never invokes
// the store. Mirrors TestResetAdminRejectsMissingKey.
func TestCreateResetTokenRejectsMissingKey(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"admin"}`)
	w := doResetToken(h, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403 (missing X-E2E-Reset-Key)", w.Code)
	}
	if store.callCountCreate != 0 {
		t.Fatalf("CreateResetToken was called %d times on missing key — auth gate failed",
			store.callCountCreate)
	}
}

// TestCreateResetTokenRejectsWrongKey mirrors TestResetAdminRejectsWrongKey.
func TestCreateResetTokenRejectsWrongKey(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"admin"}`)
	req.Header.Set("X-E2E-Reset-Key", "wrong-secret")
	w := doResetToken(h, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403 (wrong X-E2E-Reset-Key)", w.Code)
	}
	if store.callCountCreate != 0 {
		t.Fatalf("CreateResetToken was called %d times on wrong key", store.callCountCreate)
	}
}

// TestCreateResetTokenRequiresServerSideKey mirrors
// TestResetAdminRequiresServerSideKey.
func TestCreateResetTokenRequiresServerSideKey(t *testing.T) {
	os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"admin"}`)
	req.Header.Set("X-E2E-Reset-Key", "client-supplied")
	w := doResetToken(h, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503 (no server-side E2E_RESET_KEY)", w.Code)
	}
	if store.callCountCreate != 0 {
		t.Fatalf("CreateResetToken was called %d times despite unset server-side key",
			store.callCountCreate)
	}
}

// TestCreateResetTokenHappyPath verifies a correctly-keyed request
// reaches the store with the expected username and a non-empty
// 64-char hex raw token (32 bytes from crypto/rand, hex-encoded —
// matches the production ForgotPassword flow).
func TestCreateResetTokenHappyPath(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"admin"}`)
	req.Header.Set("X-E2E-Reset-Key", "ci-secret-1234567890abcdef")
	w := doResetToken(h, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", w.Header().Get("Content-Type"))
	}

	body := decodeResetToken(t, w)
	if body.Token == "" {
		t.Fatal("response token is empty")
	}
	if len(body.Token) != 64 {
		t.Errorf("token length: got %d, want 64 (32 bytes hex-encoded)", len(body.Token))
	}
	// Round-trip: the token the handler returned to the spec must be
	// the same one passed to CreateResetToken — this pins the contract
	// the spec relies on.
	if store.recordedToken != body.Token {
		t.Errorf("token round-trip: handler returned %q, store got %q", body.Token, store.recordedToken)
	}
	if store.recordedUsername != "admin" {
		t.Errorf("username: got %q, want admin", store.recordedUsername)
	}
}

// TestCreateResetTokenErrorPathDoesNotLeak mirrors
// TestResetAdminErrorPathDoesNotLeak. The 500 envelope must contain
// only the generic code/message — never the underlying error detail.
func TestCreateResetTokenErrorPathDoesNotLeak(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{
		createResetTokenFn: func(_ context.Context, _, _ string) error {
			return errors.New("internal-db-thing-12345")
		},
	}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"admin"}`)
	req.Header.Set("X-E2E-Reset-Key", "ci-secret-1234567890abcdef")
	w := doResetToken(h, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d, want 500", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "internal-db-thing-12345") {
		t.Fatalf("response leaked internal error detail: %s", body)
	}
	var bodyMap map[string]any
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&bodyMap); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if bodyMap["code"] != "INTERNAL" {
		t.Errorf("code: got %v, want INTERNAL", bodyMap["code"])
	}
	if bodyMap["error"] != "internal" {
		t.Errorf("error: got %v, want \"internal\"", bodyMap["error"])
	}
}

// TestCreateResetTokenNotFound verifies that an unknown username
// returns 404 without invoking CreateResetToken. (We never want to
// hash a token for a user that doesn't exist.)
func TestCreateResetTokenNotFound(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{
		getUserFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, nil // GetUser convention: (nil, nil) = not found
		},
	}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"ghost"}`)
	req.Header.Set("X-E2E-Reset-Key", "ci-secret-1234567890abcdef")
	w := doResetToken(h, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404 (unknown user)", w.Code)
	}
	if store.callCountCreate != 0 {
		t.Fatalf("CreateResetToken was called %d times for a non-existent user", store.callCountCreate)
	}
}

// TestCreateResetTokenFKViolationTreatedAs404 covers the defensive
// branch: if a store implementation ever lets a nil user through
// (e.g. via a different GetUser signature), the FK constraint on
// password_reset_tokens will reject the INSERT, and we map that to
// 404 instead of 500. The string-match logic is what we test here.
func TestCreateResetTokenFKViolationTreatedAs404(t *testing.T) {
	os.Setenv("E2E_RESET_KEY", "ci-secret-1234567890abcdef")
	defer os.Unsetenv("E2E_RESET_KEY")

	store := &fakeResetTokenStore{
		// GetUser returns a (non-nil) user so we pass the 404-pre-check.
		// CreateResetToken then "fails" with a real-looking FK violation.
		createResetTokenFn: func(_ context.Context, _, _ string) error {
			return errors.New(`ERROR: insert or update on table "password_reset_tokens" violates foreign key constraint "password_reset_tokens_username_fkey" (SQLSTATE 23503)`)
		},
	}
	h := NewTestCreateResetTokenHandler(store)

	req := newResetTokenReq("POST", `{"username":"admin"}`)
	req.Header.Set("X-E2E-Reset-Key", "ci-secret-1234567890abcdef")
	w := doResetToken(h, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404 (FK violation mapped to NOT_FOUND)", w.Code)
	}
	if strings.Contains(w.Body.String(), "violates foreign key constraint") {
		t.Fatalf("response leaked internal error detail: %s", w.Body.String())
	}
}

// TestIsForeignKeyViolation sanity-checks the helper used by the
// FK-violation branch.
func TestIsForeignKeyViolation(t *testing.T) {
	cases := []struct {
		err  string
		want bool
	}{
		{`insert or update on table "x" violates foreign key constraint "x_fkey"`, true},
		{`SQLSTATE 23503`, true},
		{`ERROR: duplicate key value violates unique constraint "x_pkey"`, false},
		{`connection refused`, false},
		{``, false},
	}
	for _, c := range cases {
		if got := isForeignKeyViolation(c.err); got != c.want {
			t.Errorf("isForeignKeyViolation(%q): got %v, want %v", c.err, got, c.want)
		}
	}
}

// Reference time so the import is kept if we add timing-sensitive
// tests later. (The handler itself is timing-agnostic — it relies
// on the store to enforce the 1h TTL.)
var _ = time.Hour

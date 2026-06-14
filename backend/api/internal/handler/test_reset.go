//go:build e2e
// +build e2e

// TestResetAdminHandler is registered ONLY in e2e-test builds (build tag
// above). Production builds do not even compile this file — defense in
// depth on top of the cfg.IsProduction() runtime gate in main.go.
//
// Exposes a single endpoint used exclusively by apps/admin/e2e/security.spec.ts
// to reset the bootstrap admin's password and password_changed_at before
// running tests that mutate auth state. Authenticates via the X-E2E-Reset-Key
// header (constant-time compared to the E2E_RESET_KEY env var) — without the
// header the request is rejected with 403 BEFORE any state mutation. This
// keeps the route safe even on a misconfigured host where APP_ENV is empty
// (cfg.IsProduction() fail-open is now masked by the missing build tag).
package handler

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"os"
	"time"

	"mioru/internal/jsonerr"
	"mioru/internal/model"
)

type TestResetAdminHandler struct {
	store testAdminStore
}

type testAdminStore interface {
	ResetAdminForTest(ctx context.Context, u model.User, passwordChangedAt time.Time) error
}

func NewTestResetAdminHandler(pgStore testAdminStore) *TestResetAdminHandler {
	return &TestResetAdminHandler{store: pgStore}
}

type resetAdminRequest struct {
	Username          string `json:"username"`
	HashedPassword    string `json:"hashed_password"`
	Email             string `json:"email"`
	DisplayName       string `json:"display_name"`
	Role              string `json:"role"`
	PasswordChangedAt string `json:"password_changed_at"` // RFC3339, optional
}

const e2eResetKeyHeader = "X-E2E-Reset-Key"

func (h *TestResetAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonerr.ErrorCode(w, "method not allowed", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	// Auth: constant-time compare against E2E_RESET_KEY. An empty server-side
	// secret is treated as a misconfiguration and 503s — never fail-open.
	expected := os.Getenv("E2E_RESET_KEY")
	if expected == "" {
		slog.Error("test reset: E2E_RESET_KEY not set; refusing to handle request")
		jsonerr.ErrorCode(w, "test reset disabled", http.StatusServiceUnavailable, "TEST_RESET_DISABLED")
		return
	}
	got := r.Header.Get(e2eResetKeyHeader)
	if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		jsonerr.ErrorCode(w, "invalid or missing X-E2E-Reset-Key", http.StatusForbidden, "FORBIDDEN")
		return
	}

	var body resetAdminRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Username == "" || body.HashedPassword == "" {
		jsonerr.ErrorCode(w, "username and hashed_password required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if body.Email == "" {
		body.Email = body.Username + "@mioru.store"
	}
	if body.DisplayName == "" {
		body.DisplayName = body.Username
	}
	if body.Role == "" {
		body.Role = "super_admin"
	}
	passwordChangedAt := time.Now().Add(-1 * time.Hour)
	if body.PasswordChangedAt != "" {
		if t, err := time.Parse(time.RFC3339, body.PasswordChangedAt); err == nil {
			passwordChangedAt = t
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user := model.User{
		Username:    body.Username,
		Email:       body.Email,
		HashedPW:    body.HashedPassword,
		DisplayName: body.DisplayName,
		AvatarColor: "#44944A",
		Role:        body.Role,
	}

	if err := h.store.ResetAdminForTest(ctx, user, passwordChangedAt); err != nil {
		// OWASP: never leak internal error detail to the client. Log it
		// server-side and return a generic 500 envelope.
		slog.Error("test reset: ResetAdminForTest failed", "err", err.Error(), "username", body.Username)
		jsonerr.ErrorCode(w, "internal", http.StatusInternalServerError, "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

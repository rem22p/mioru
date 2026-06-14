package handler

import (
	"context"
	"net/http"
	"time"

	"mioru/internal/jsonerr"
	"mioru/internal/model"
)

// TestResetAdminHandler exposes a single endpoint used exclusively by
// E2E tests (apps/admin/e2e/security.spec.ts) to reset the bootstrap admin's
// password and password_changed_at before running tests that mutate auth
// state. It is intentionally registered behind an APP_ENV check in
// cmd/server/main.go — never exposed in production builds.
//
// The handler upserts the admin user with the supplied (hashed) password,
// so calling it is idempotent across CI re-runs.
type TestResetAdminHandler struct {
	store testAdminStore
}

// testAdminStore is a narrow seam so this handler does not have to
// implement the full userStore interface (which is consumed by a large
// number of unit-test fakes).
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

func (h *TestResetAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonerr.ErrorCode(w, "method not allowed", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
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
	// Defaults match BOOTSTRAP_ADMIN_* env contract.
	if body.Email == "" {
		body.Email = body.Username + "@mioru.store"
	}
	if body.DisplayName == "" {
		body.DisplayName = body.Username
	}
	if body.Role == "" {
		body.Role = "super_admin"
	}
	// Default password_changed_at: an hour in the past, so the next login's
	// iat > changed_at is guaranteed even on a clock that ticks during the
	// request. (The clock-skew window is what the original failure was.)
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
		jsonerr.ErrorCode(w, "internal: "+err.Error(), http.StatusInternalServerError, "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

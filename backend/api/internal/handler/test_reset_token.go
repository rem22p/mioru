//go:build e2e
// +build e2e

// TestCreateResetTokenHandler is registered ONLY in e2e-test builds (build
// tag above). Production builds do not even compile this file — defense in
// depth on top of the cfg.IsProduction() runtime gate in main.go.
//
// Exposes a single endpoint used exclusively by
// apps/admin/e2e/password-reset.spec.ts to obtain a fresh raw password-reset
// token for a given username. Authenticates via the X-E2E-Reset-Key header
// (constant-time compared to the E2E_RESET_KEY env var) — without the header
// the request is rejected with 403 BEFORE any state mutation. This keeps the
// route safe even on a misconfigured host where APP_ENV is empty
// (cfg.IsProduction() fail-open is now masked by the missing build tag).
//
// Security model (mirrors TestResetAdminHandler from #49):
//   - The raw token is generated via crypto/rand (same source as the
//     production ForgotPassword flow), persisted via the regular
//     CreateResetToken path (so the SHA-256 hash lands in
//     password_reset_tokens exactly as it would for a real reset), and
//     returned once in the response body. It is NOT logged, NOT written
//     to a side column, NOT echoed to any other channel. The handler
//     is the single point of truth for "test wants a raw token" — same
//     security contract as `TestResetTokenHashedAtRest` ensures for
//     production traffic.
package handler

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"mioru/internal/jsonerr"
	"mioru/internal/model"
)

type TestCreateResetTokenHandler struct {
	store testResetTokenStore
}

// testResetTokenStore is a narrow seam so this handler does not bloat
// the main userStore interface (which is consumed by a large number
// of unit-test fakes). It only needs: look up a user by username,
// persist a token. Mirrors the `testAdminStore` pattern from
// TestResetAdminHandler.
type testResetTokenStore interface {
	GetUser(ctx context.Context, username string) (*model.User, error)
	// CreateResetToken stores a one-time password-reset token. The
	// implementation must hash the token (per security contract
	// TestResetTokenHashedAtRest) before persistence — never store
	// the raw value.
	CreateResetToken(ctx context.Context, username, rawToken string) error
}

func NewTestCreateResetTokenHandler(pgStore testResetTokenStore) *TestCreateResetTokenHandler {
	return &TestCreateResetTokenHandler{store: pgStore}
}

type createResetTokenRequest struct {
	Username string `json:"username"`
}

type createResetTokenResponse struct {
	Token string `json:"token"`
}

func (h *TestCreateResetTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonerr.ErrorCode(w, "method not allowed", http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}
	// Auth: constant-time compare against E2E_RESET_KEY. An empty server-side
	// secret is treated as a misconfiguration and 503s — never fail-open.
	expected := os.Getenv("E2E_RESET_KEY")
	if expected == "" {
		slog.Error("test create-reset-token: E2E_RESET_KEY not set; refusing to handle request")
		jsonerr.ErrorCode(w, "test reset disabled", http.StatusServiceUnavailable, "TEST_RESET_DISABLED")
		return
	}
	got := r.Header.Get(e2eResetKeyHeader)
	if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		jsonerr.ErrorCode(w, "invalid or missing X-E2E-Reset-Key", http.StatusForbidden, "FORBIDDEN")
		return
	}

	var body createResetTokenRequest
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.Username == "" {
		jsonerr.ErrorCode(w, "username required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Look up the user. If not found, return 404 BEFORE generating a
	// token (no point hashing something the DB will reject, and a
	// distinct 404 helps the spec write a precise assertion).
	user, err := h.store.GetUser(ctx, body.Username)
	if err != nil {
		// OWASP: never leak internal error detail to the client. Log
		// it server-side and return a generic envelope.
		slog.Error("test create-reset-token: GetUser failed", "err", err.Error(), "username", body.Username)
		jsonerr.ErrorCode(w, "internal", http.StatusInternalServerError, "INTERNAL")
		return
	}
	if user == nil {
		jsonerr.ErrorCode(w, "user not found", http.StatusNotFound, "NOT_FOUND")
		return
	}

	// Generate a fresh token. Same entropy (32 bytes) and encoding
	// (hex) as the production ForgotPassword flow, so the lifetime
	// and hash semantics match exactly.
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		slog.Error("test create-reset-token: rand.Read failed", "err", err.Error())
		jsonerr.ErrorCode(w, "internal", http.StatusInternalServerError, "INTERNAL")
		return
	}
	rawToken := hex.EncodeToString(tokenBytes)

	if err := h.store.CreateResetToken(ctx, body.Username, rawToken); err != nil {
		// Distinguish "unknown user" (FK violation) from "real error".
		// Production path is well-typed; we use a substring match on
		// the pgx error message here so the test-only file does not
		// need to import pgconn.
		errStr := err.Error()
		if isForeignKeyViolation(errStr) {
			jsonerr.ErrorCode(w, "user not found", http.StatusNotFound, "NOT_FOUND")
			return
		}
		slog.Error("test create-reset-token: CreateResetToken failed", "err", errStr, "username", body.Username)
		jsonerr.ErrorCode(w, "internal", http.StatusInternalServerError, "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(createResetTokenResponse{Token: rawToken})
}

// isForeignKeyViolation detects pgx "violates foreign key constraint"
// errors. We use a substring match so the test-only handler does not
// import pgx/pgconn.
func isForeignKeyViolation(errStr string) bool {
	return strings.Contains(errStr, "violates foreign key constraint") ||
		strings.Contains(errStr, "23503") // SQLSTATE
}

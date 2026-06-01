package middleware

import (
	"context"
	"net/http"
	"time"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/jsonerr"
)

type ctxKey struct{}

// UserEpochFunc returns the password-changed-at timestamp for the user, used to
// invalidate tokens minted before the last password change. ok is false when the
// user no longer exists (their tokens must then be rejected).
type UserEpochFunc func(ctx context.Context, username string) (changedAt time.Time, ok bool, err error)

// AuthMW authenticates admin/staff requests via a JWT (typ=user) carried in
// an HttpOnly cookie (no Authorization-header / Bearer path — closed in #8 to
// remove the localStorage XSS-exfil surface). Beyond signature/expiry/audience
// checks it enforces session revocation: a token whose issued-at predates the
// user's password_changed_at is rejected, so changing or resetting a password
// logs out every previously issued session.
func AuthMW(secret string, userEpoch UserEpochFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieauth.AdminAuthCookie)
			if err != nil || c.Value == "" {
				jsonerr.ErrorCode(w, "authentication required", http.StatusUnauthorized, "AUTH_REQUIRED")
				return
			}
			username, iat, err := auth.ParseToken(c.Value, secret, auth.TokenTypeUser)
			if err != nil {
				jsonerr.ErrorCode(w, "invalid or expired token", http.StatusUnauthorized, "AUTH_INVALID")
				return
			}
			changedAt, ok, err := userEpoch(r.Context(), username)
			if err != nil {
				jsonerr.ErrorCode(w, "internal error", http.StatusInternalServerError, "INTERNAL")
				return
			}
			// Compare at second granularity (iat is Unix seconds): a token issued
			// in the same second as the change is accepted, anything older is not.
			if !ok || iat < changedAt.Unix() {
				jsonerr.ErrorCode(w, "session revoked", http.StatusUnauthorized, "AUTH_INVALID")
				return
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Username(r *http.Request) string {
	v := r.Context().Value(ctxKey{})
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// RequireAdmin must be composed *after* AuthMW. It looks up the authenticated
// user's role via getRole (DB-backed — it never trusts a token claim) and
// rejects any non-admin with 403. super_admin inherits all admin privileges.
func RequireAdmin(getRole func(context.Context, string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := Username(r)
			if username == "" {
				jsonerr.ErrorCode(w, "authentication required", http.StatusUnauthorized, "AUTH_REQUIRED")
				return
			}
			role, err := getRole(r.Context(), username)
			if err != nil {
				jsonerr.ErrorCode(w, "internal error", http.StatusInternalServerError, "INTERNAL")
				return
			}
			if role != "admin" && role != "super_admin" {
				jsonerr.ErrorCode(w, "forbidden", http.StatusForbidden, "FORBIDDEN")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

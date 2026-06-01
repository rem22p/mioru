package middleware

import (
	"context"
	"net/http"

	"mioru/internal/jsonerr"
)

// RequireSuperAdmin must be composed *after* AuthMW. It calls getRole (DB-backed
// — it never trusts a token claim) and rejects any user whose role is not
// "super_admin" with 403. Regular admins get 403; unauthenticated users are
// caught by AuthMW before reaching this function.
func RequireSuperAdmin(getRole func(context.Context, string) (string, error)) func(http.Handler) http.Handler {
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
			if role != "super_admin" {
				jsonerr.ErrorCode(w, "forbidden", http.StatusForbidden, "FORBIDDEN")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

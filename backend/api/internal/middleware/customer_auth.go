package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/jsonerr"
)

type customerCtxKey struct{}

// CustomerEpochFunc returns the password-changed-at timestamp for the customer,
// used to invalidate tokens minted before the last password change. ok is false
// when the customer no longer exists.
type CustomerEpochFunc func(ctx context.Context, id int64) (changedAt time.Time, ok bool, err error)

// CustomerAuthMW is the auth middleware for store customers. The JWT is
// carried in an HttpOnly cookie (#8 removed the Authorization/Bearer path),
// with subject = customer ID (typ=customer). Like AuthMW it rejects tokens
// issued before the customer's last password change (session revocation).
//
// Every rejection path goes through jsonerr.ErrorCode so the SPA receives
// `application/json` with a reserved machine `code` (per CLAUDE.md API
// contract) and can branch on `code` rather than on the human text. The
// admin path has used this shape since the envelope refactor; this middleware
// was the last holdout (mirrored in fix for #31).
func CustomerAuthMW(secret string, customerEpoch CustomerEpochFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieauth.StoreAuthCookie)
			if err != nil || c.Value == "" {
				jsonerr.ErrorCode(w, "authentication required", http.StatusUnauthorized, "AUTH_REQUIRED")
				return
			}
			sub, iat, err := auth.ParseToken(c.Value, secret, auth.TokenTypeCustomer)
			if err != nil {
				jsonerr.ErrorCode(w, "invalid or expired token", http.StatusUnauthorized, "AUTH_INVALID")
				return
			}
			// sub is the customer ID as string
			id, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				jsonerr.ErrorCode(w, "invalid or expired token", http.StatusUnauthorized, "AUTH_INVALID")
				return
			}
			changedAt, ok, err := customerEpoch(r.Context(), id)
			if err != nil {
				jsonerr.ErrorCode(w, "internal error", http.StatusInternalServerError, "INTERNAL")
				return
			}
			if !ok || iat < changedAt.Unix() {
				jsonerr.ErrorCode(w, "session revoked", http.StatusUnauthorized, "AUTH_INVALID")
				return
			}
			ctx := context.WithValue(r.Context(), customerCtxKey{}, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CustomerID extracts the customer ID from the request context.
func CustomerID(r *http.Request) int64 {
	v := r.Context().Value(customerCtxKey{})
	if id, ok := v.(int64); ok {
		return id
	}
	return 0
}

// WithCustomerID returns a context with the customer ID set. Used in tests to
// bypass CustomerAuthMW when testing handlers that call CustomerID(ctx).
func WithCustomerID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, customerCtxKey{}, id)
}

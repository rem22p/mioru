package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mioru/internal/auth"
)

type customerCtxKey struct{}

// CustomerEpochFunc returns the password-changed-at timestamp for the customer,
// used to invalidate tokens minted before the last password change. ok is false
// when the customer no longer exists.
type CustomerEpochFunc func(ctx context.Context, id int64) (changedAt time.Time, ok bool, err error)

// CustomerAuthMW is the auth middleware for store customers. It expects a JWT
// with subject = customer ID (typ=customer) and, like AuthMW, rejects tokens
// issued before the customer's last password change (session revocation).
func CustomerAuthMW(secret string, customerEpoch CustomerEpochFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")
			sub, iat, err := auth.ParseToken(token, secret, auth.TokenTypeCustomer)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			// sub is the customer ID as string
			id, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			changedAt, ok, err := customerEpoch(r.Context(), id)
			if err != nil {
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}
			if !ok || iat < changedAt.Unix() {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
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

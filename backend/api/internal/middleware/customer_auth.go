package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"mioru/internal/auth"
)

type customerCtxKey struct{}

// CustomerAuthMW is the auth middleware for store customers.
// It expects a JWT with subject = customer ID.
func CustomerAuthMW(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")
			sub, err := auth.ParseToken(token, secret, auth.TokenTypeCustomer)
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

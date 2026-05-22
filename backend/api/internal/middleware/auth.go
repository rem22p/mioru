package middleware

import (
	"context"
	"net/http"
	"strings"

	"mioru/internal/auth"
)

type ctxKey struct{}

func AuthMW(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(h, "Bearer ")
			username, err := auth.ParseToken(token, secret)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
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

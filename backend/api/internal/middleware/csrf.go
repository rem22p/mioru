package middleware

import (
	"crypto/subtle"
	"net/http"

	"mioru/internal/jsonerr"
)

// CSRF returns the double-submit-cookie middleware for a given CSRF cookie
// name (one per audience — see cookieauth.AdminCSRFCookie /
// cookieauth.StoreCSRFCookie). On every state-changing method it requires
// the request to carry the same value in BOTH places: the cookie (set on
// login, readable by the SPA via document.cookie) and the X-CSRF-Token
// header (echoed by the SPA on each mutation). They are compared in
// constant time; a missing, empty, or mismatching token returns 403 with
// code CSRF_INVALID.
//
// Safe methods (GET / HEAD / OPTIONS) are passed through unchecked — they
// must not have side effects, so an attacker forcing one from a foreign
// origin cannot leverage the ambient cookie to do harm.
//
// This middleware does NOT itself authenticate the request; pair it with
// AuthMW or CustomerAuthMW on protected routes.
func CSRF(cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}

			c, err := r.Cookie(cookieName)
			if err != nil || c.Value == "" {
				jsonerr.ErrorCode(w, "csrf token missing", http.StatusForbidden, "CSRF_INVALID")
				return
			}
			header := r.Header.Get("X-CSRF-Token")
			if header == "" {
				jsonerr.ErrorCode(w, "csrf token missing", http.StatusForbidden, "CSRF_INVALID")
				return
			}
			// ConstantTimeCompare returns 0 also for length mismatches, so it
			// short-circuits safely on unequal-length inputs.
			if subtle.ConstantTimeCompare([]byte(header), []byte(c.Value)) != 1 {
				jsonerr.ErrorCode(w, "csrf token mismatch", http.StatusForbidden, "CSRF_INVALID")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

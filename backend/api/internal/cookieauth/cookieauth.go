// Package cookieauth provides the cookie + CSRF primitives used by the auth
// middleware. The JWT lives in an HttpOnly cookie (not localStorage, which is
// XSS-exfiltratable); a separate non-HttpOnly CSRF cookie is read by the SPA
// and echoed in the X-CSRF-Token header on every mutation, where it is
// compared constant-time against the cookie value (double-submit pattern).
package cookieauth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// Cookie names — separated by audience so the admin and storefront apps,
// which share the eTLD+1 (mioru.store), do not overwrite each other's cookies.
const (
	AdminAuthCookie = "auth_token"
	AdminCSRFCookie = "csrf_token"
	StoreAuthCookie = "store_auth"
	StoreCSRFCookie = "store_csrf"
)

// csrfTokenBytes is the entropy of a CSRF token before base64 encoding. 32
// bytes (256 bits) is well past any feasible online-guess threat.
const csrfTokenBytes = 32

// GenCSRFToken returns a cryptographically random URL-safe CSRF token.
func GenCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetAuthCookie writes the JWT cookie. It is HttpOnly so the SPA cannot read
// it (closing the XSS-exfiltration path that localStorage left open),
// SameSite=Lax so it is attached to same-site cross-origin fetches (the
// storefront → api.mioru.store path) but never to cross-site requests, and
// Secure in production so it is never sent over plaintext. maxAgeSec ties the
// cookie's lifetime to the JWT's exp claim.
func SetAuthCookie(w http.ResponseWriter, name, value string, secure bool, maxAgeSec int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetCSRFCookie writes the CSRF cookie. It is NOT HttpOnly — the SPA must
// read it via document.cookie to echo into the X-CSRF-Token header on
// mutations. Knowing the value is fine: an attacker on another site cannot
// read cookies for our domain (SameSite + Same-Origin Policy together),
// which is what the double-submit pattern relies on.
func SetCSRFCookie(w http.ResponseWriter, name, value string, secure bool, maxAgeSec int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAgeSec,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie writes an expired cookie under the same name to instruct the
// browser to drop the previous value. Used by the logout endpoints. The path
// must match what was originally set, otherwise the browser keeps the old
// cookie alongside the new one.
func ClearCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true, // irrelevant for an empty value, but keeps the attribute consistent
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

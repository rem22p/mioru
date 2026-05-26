package cookieauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// findCookie returns the Set-Cookie header struct for the given name, or nil
// when the response did not set one. ResponseRecorder.Result() walks the
// Set-Cookie headers exactly the way a real client would.
func findCookie(t *testing.T, rr *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestSetAuthCookieFlags(t *testing.T) {
	rr := httptest.NewRecorder()
	SetAuthCookie(rr, AdminAuthCookie, "the-jwt", true, 3600)

	c := findCookie(t, rr, AdminAuthCookie)
	if c == nil {
		t.Fatal("auth cookie not set")
	}
	if c.Value != "the-jwt" {
		t.Errorf("value = %q, want %q", c.Value, "the-jwt")
	}
	if !c.HttpOnly {
		t.Error("auth cookie must be HttpOnly (closes the XSS-exfil path)")
	}
	if !c.Secure {
		t.Error("auth cookie must be Secure when secure=true (prod)")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want /", c.Path)
	}
	if c.MaxAge != 3600 {
		t.Errorf("MaxAge = %d, want 3600", c.MaxAge)
	}
}

func TestSetAuthCookieInsecureInDev(t *testing.T) {
	// When secure=false (dev/HTTP), the Secure flag must be off; otherwise
	// the browser silently drops the cookie outside localhost-exception
	// browsers, breaking dev login.
	rr := httptest.NewRecorder()
	SetAuthCookie(rr, AdminAuthCookie, "x", false, 60)

	c := findCookie(t, rr, AdminAuthCookie)
	if c == nil {
		t.Fatal("cookie not set")
	}
	if c.Secure {
		t.Error("auth cookie must NOT be Secure when secure=false (dev)")
	}
}

func TestSetCSRFCookieIsReadableByJS(t *testing.T) {
	rr := httptest.NewRecorder()
	SetCSRFCookie(rr, AdminCSRFCookie, "csrf-value", true, 3600)

	c := findCookie(t, rr, AdminCSRFCookie)
	if c == nil {
		t.Fatal("csrf cookie not set")
	}
	if c.HttpOnly {
		t.Error("CSRF cookie must NOT be HttpOnly — the SPA must read it via document.cookie")
	}
	if !c.Secure {
		t.Error("CSRF cookie must be Secure in prod")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", c.SameSite)
	}
}

func TestClearCookieMaxAgeNegative(t *testing.T) {
	rr := httptest.NewRecorder()
	ClearCookie(rr, AdminAuthCookie, true)

	c := findCookie(t, rr, AdminAuthCookie)
	if c == nil {
		t.Fatal("clear cookie not set")
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want < 0 (instructs browser to drop the cookie)", c.MaxAge)
	}
	if c.Value != "" {
		t.Errorf("Value = %q, want empty", c.Value)
	}
	if c.Path != "/" {
		t.Errorf("Path = %q, want / (must match original to actually overwrite)", c.Path)
	}
}

func TestGenCSRFTokenUniqueAndEncoded(t *testing.T) {
	t1, err := GenCSRFToken()
	if err != nil {
		t.Fatalf("GenCSRFToken: %v", err)
	}
	t2, err := GenCSRFToken()
	if err != nil {
		t.Fatalf("GenCSRFToken: %v", err)
	}
	if t1 == t2 {
		t.Error("two CSRF tokens collided — entropy source is broken")
	}
	// base64.RawURLEncoding(32 bytes) = 43 chars (no padding).
	if len(t1) != 43 {
		t.Errorf("token length = %d, want 43 (base64 of 32 raw bytes, no padding)", len(t1))
	}
	for _, b := range []byte(t1) {
		ok := (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') ||
			(b >= '0' && b <= '9') || b == '-' || b == '_'
		if !ok {
			t.Errorf("CSRF token contains non-URL-safe byte %q", b)
			break
		}
	}
}

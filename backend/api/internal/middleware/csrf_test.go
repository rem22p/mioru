package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mioru/internal/cookieauth"
)

func csrfHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestCSRFAllowsSafeMethods(t *testing.T) {
	// Safe methods must pass through without any CSRF check; otherwise every
	// GET would need a token, which is both pointless and would break SPA
	// bootstrapping (the cookie isn't readable before the first response).
	for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(m, func(t *testing.T) {
			reached := false
			h := CSRF(cookieauth.AdminCSRFCookie)(csrfHandler(&reached))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest(m, "/api/users/me", nil))

			if rr.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
			}
			if !reached {
				t.Errorf("handler must be reached for %s", m)
			}
		})
	}
}

func TestCSRFRejectsMissingHeaderOrCookie(t *testing.T) {
	cases := []struct {
		name   string
		cookie string
		header string
	}{
		{"no cookie, no header", "", ""},
		{"cookie only, no header", "abc", ""},
		{"header only, no cookie", "", "abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			h := CSRF(cookieauth.AdminCSRFCookie)(csrfHandler(&reached))

			r := httptest.NewRequest(http.MethodPost, "/api/admin/products", nil)
			if tc.cookie != "" {
				r.AddCookie(&http.Cookie{Name: cookieauth.AdminCSRFCookie, Value: tc.cookie})
			}
			if tc.header != "" {
				r.Header.Set("X-CSRF-Token", tc.header)
			}

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)

			if rr.Code != http.StatusForbidden {
				t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
			}
		if reached {
			t.Error("handler must not be reached when CSRF check fails")
		}
		assertJSONError(t, rr, "CSRF_INVALID")
		})
	}
}

func TestCSRFRejectsMismatch(t *testing.T) {
	reached := false
	h := CSRF(cookieauth.AdminCSRFCookie)(csrfHandler(&reached))

	r := httptest.NewRequest(http.MethodPost, "/api/admin/products", nil)
	r.AddCookie(&http.Cookie{Name: cookieauth.AdminCSRFCookie, Value: "token-a"})
	r.Header.Set("X-CSRF-Token", "token-b")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
	if reached {
		t.Error("handler must not be reached when CSRF tokens differ")
	}
	assertJSONError(t, rr, "CSRF_INVALID")
}

func TestCSRFAcceptsMatchingPair(t *testing.T) {
	reached := false
	h := CSRF(cookieauth.AdminCSRFCookie)(csrfHandler(&reached))

	r := httptest.NewRequest(http.MethodPost, "/api/admin/products", nil)
	r.AddCookie(&http.Cookie{Name: cookieauth.AdminCSRFCookie, Value: "the-token"})
	r.Header.Set("X-CSRF-Token", "the-token")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !reached {
		t.Error("handler must be reached when CSRF cookie and header match")
	}
}

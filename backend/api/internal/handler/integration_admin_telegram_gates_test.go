package handler_test

import (
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

// TestIntegrationAdminTelegramGates verifies the AuthZ/CSRF
// gates on all three admin Telegram endpoints.
//
// B3 (final gate): the pre-existing tests injected a session
// directly into the context, bypassing AuthMW → RequireAdmin →
// CSRF. This test exercises the real middleware chain through
// e.wrapAdmin and e.wrapUserCSRF, plus unauthenticated and
// customer-session callers.
func TestIntegrationAdminTelegramGates(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "tg-gates", "admin")
	cust, _ := e.customerSession(t, "tg-gates-cust@ex.com")

	// ── GET endpoints: anonymous, customer, no-admin role ──
	getEndpoints := []struct {
		name string
		h    http.HandlerFunc
		path string
	}{
		{"diagnose", e.adminTelegramH.Diagnose, "/api/admin/telegram/diagnose"},
		{"messages", e.adminTelegramH.Messages, "/api/admin/telegram/messages"},
	}

	for _, ep := range getEndpoints {
		// Anonymous → blocked by AuthMW (wrapAdmin without session).
		rrAnon := e.do(t, e.wrapAdmin(ep.h), http.MethodGet, ep.path, reqOpts{})
		if rrAnon.Code != http.StatusUnauthorized {
			t.Errorf("%s anonymous: got %d, want 401", ep.name, rrAnon.Code)
		}

		// Customer session → blocked by AuthMW (customer JWT
		// rejected by admin AuthMW because token type is
		// TokenTypeCustomer, not TokenTypeUser).
		rrCust := e.do(t, e.wrapAdmin(ep.h), http.MethodGet, ep.path, reqOpts{sess: cust})
		if rrCust.Code == http.StatusOK {
			t.Errorf("%s customer: got 200, want blocked", ep.name)
		}
	}

	// Admin sees the endpoints.
	rrDiag := e.do(t, e.wrapAdmin(e.adminTelegramH.Diagnose), http.MethodGet,
		"/api/admin/telegram/diagnose", reqOpts{sess: admin})
	if rrDiag.Code != http.StatusOK {
		t.Errorf("diagnose admin: want 200, got %d", rrDiag.Code)
	}

	// ── POST /test without CSRF → 403 ──
	rrNoCSRF := e.do(t, e.wrapUserCSRF(e.adminTelegramH.Test), http.MethodPost,
		"/api/admin/telegram/test",
		reqOpts{sess: admin, body: map[string]any{}})
	if rrNoCSRF.Code != http.StatusForbidden {
		t.Errorf("test no CSRF: want 403, got %d", rrNoCSRF.Code)
	}

	// ── POST /test with CSRF → 503 (no token configured) ──
	rrWithCSRF := e.do(t, e.wrapUserCSRF(e.adminTelegramH.Test), http.MethodPost,
		"/api/admin/telegram/test",
		reqOpts{
			sess:           admin,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{},
		})
	if rrWithCSRF.Code != http.StatusServiceUnavailable {
		t.Errorf("test with CSRF: want 503, got %d (%s)", rrWithCSRF.Code, rrWithCSRF.Body.String())
	}
}

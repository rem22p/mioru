package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

// TestIntegrationAdminLoginHappyAndReuseCookie logs in with a real (bcrypt)
// password, then reuses the freshly minted session cookie on GET me.
func TestIntegrationAdminLoginHappyAndReuseCookie(t *testing.T) {
	e := newEnv(t)
	// Create the user (its returned session is irrelevant — we drive login).
	e.adminUserWithPassword(t, "adminlogin", "admin")

	rr := e.do(t, http.HandlerFunc(e.authH.Login), http.MethodPost, "/api/auth/login",
		reqOpts{body: map[string]any{"username": "adminlogin", "password": goodPassword}})
	if rr.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var prof struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decode(t, rr, &prof)
	if prof.Username != "adminlogin" {
		t.Errorf("login username = %q, want adminlogin", prof.Username)
	}

	// Reuse the minted cookies on an authenticated GET me.
	sess := sessionFromResponse(t, rr, cookieauth.AdminAuthCookie, cookieauth.AdminCSRFCookie)
	me := e.do(t, e.wrapUserAuth(e.authH.Me), http.MethodGet, "/api/users/me", reqOpts{sess: sess})
	if me.Code != http.StatusOK {
		t.Fatalf("me with minted cookie: want 200, got %d (%s)", me.Code, me.Body.String())
	}
	var meBody struct {
		Username string `json:"username"`
	}
	decode(t, me, &meBody)
	if meBody.Username != "adminlogin" {
		t.Errorf("me username = %q, want adminlogin", meBody.Username)
	}
}

func TestIntegrationAdminLoginWrongPassword(t *testing.T) {
	e := newEnv(t)
	e.adminUserWithPassword(t, "wrongpw", "admin")

	rr := e.do(t, http.HandlerFunc(e.authH.Login), http.MethodPost, "/api/auth/login",
		reqOpts{body: map[string]any{"username": "wrongpw", "password": "NotThePassw0rd"}})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong password: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("code = %q, want AUTH_INVALID", env.Code)
	}
}

func TestIntegrationAdminLoginMissingUser(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.authH.Login), http.MethodPost, "/api/auth/login",
		reqOpts{body: map[string]any{"username": "ghost", "password": goodPassword}})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing user: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("code = %q, want AUTH_INVALID", env.Code)
	}
}

// TestIntegrationAdminLogoutCSRFGate pins that admin logout is CSRF-gated: a
// mismatched X-CSRF-Token is rejected with 403, a matching one succeeds.
func TestIntegrationAdminLogoutCSRFGate(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "adminlogout", "admin")

	bad := e.do(t, e.wrapUserCSRF(e.authH.Logout), http.MethodPost, "/api/auth/logout",
		reqOpts{sess: sess, csrfCookieName: cookieauth.AdminCSRFCookie, badCSRF: true})
	if bad.Code != http.StatusForbidden {
		t.Fatalf("logout bad CSRF: want 403, got %d (%s)", bad.Code, bad.Body.String())
	}

	ok := e.do(t, e.wrapUserCSRF(e.authH.Logout), http.MethodPost, "/api/auth/logout",
		reqOpts{sess: sess, csrfCookieName: cookieauth.AdminCSRFCookie})
	if ok.Code != http.StatusOK {
		t.Fatalf("logout valid CSRF: want 200, got %d (%s)", ok.Code, ok.Body.String())
	}
}

// TestIntegrationAdminRegisterRequiresSuperAdmin pins that a plain admin (not
// super_admin) cannot create new users — RequireSuperAdmin must 403.
func TestIntegrationAdminRegisterRequiresSuperAdmin(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "plainadmin", "admin")

	rr := e.do(t, e.wrapSuperAdmin(e.authH.Register), http.MethodPost, "/api/auth/register",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body: map[string]any{
				"first_name": "New", "last_name": "Admin",
				"email": "newadmin@ex.com", "username": "newadmin", "password": goodPassword,
			},
		})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("register as non-super: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "FORBIDDEN" {
		t.Errorf("code = %q, want FORBIDDEN", env.Code)
	}
}

// TestIntegrationAdminRegisterSuperAdminHappy: a super_admin creates a new user;
// the created account exists with role "admin".
func TestIntegrationAdminRegisterSuperAdminHappy(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "super", "super_admin")

	rr := e.do(t, e.wrapSuperAdmin(e.authH.Register), http.MethodPost, "/api/auth/register",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body: map[string]any{
				"first_name": "New", "last_name": "Admin",
				"email": "newadmin@ex.com", "username": "newadmin", "password": goodPassword,
			},
		})
	if rr.Code != http.StatusCreated {
		t.Fatalf("super register: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	u, err := e.st.GetUser(context.Background(), "newadmin")
	if err != nil || u == nil {
		t.Fatalf("GetUser(newadmin): %v / %v", u, err)
	}
	if u.Role != "admin" {
		t.Errorf("created role = %q, want admin", u.Role)
	}
}

// TestIntegrationAdminRegisterDuplicate: registering an already-taken username
// is rejected. The dup branch (handler.go) returns 400, which the jsonerr
// envelope maps to VALIDATION_FAILED.
func TestIntegrationAdminRegisterDuplicate(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "super", "super_admin")

	rr := e.do(t, e.wrapSuperAdmin(e.authH.Register), http.MethodPost, "/api/auth/register",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body: map[string]any{
				"first_name": "Dup", "last_name": "User",
				"email": "dup@ex.com", "username": "super", "password": goodPassword,
			},
		})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("dup register: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

// TestIntegrationAdminForgotPasswordAlways200: an unknown email still returns
// 200 (no account enumeration). A registered email also returns 200.
func TestIntegrationAdminForgotPasswordAlways200(t *testing.T) {
	e := newEnv(t)

	unknown := e.do(t, http.HandlerFunc(e.authH.ForgotPassword), http.MethodPost, "/api/auth/forgot-password",
		reqOpts{body: map[string]any{"email": "nobody@ex.com"}})
	if unknown.Code != http.StatusOK {
		t.Fatalf("forgot unknown email: want 200, got %d (%s)", unknown.Code, unknown.Body.String())
	}

	e.adminUserWithPassword(t, "knownuser", "admin")
	known := e.do(t, http.HandlerFunc(e.authH.ForgotPassword), http.MethodPost, "/api/auth/forgot-password",
		reqOpts{body: map[string]any{"email": "knownuser@ex.com"}})
	if known.Code != http.StatusOK {
		t.Fatalf("forgot known email: want 200, got %d (%s)", known.Code, known.Body.String())
	}
}

func TestIntegrationAdminForgotPasswordBadEmail(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.authH.ForgotPassword), http.MethodPost, "/api/auth/forgot-password",
		reqOpts{body: map[string]any{"email": "notanemail"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad email: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationAdminResetPasswordInvalidToken(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.authH.ResetPassword), http.MethodPost, "/api/auth/reset-password",
		reqOpts{body: map[string]any{"token": "deadbeef", "password": "N3wStr0ngPass"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid token: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationAdminResetPasswordHappy: a stored reset token is consumed,
// the password changes, and the new password subsequently logs in.
func TestIntegrationAdminResetPasswordHappy(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.adminUserWithPassword(t, "resetme", "admin")
	if err := e.st.CreateResetToken(ctx, "resetme", "raw-reset-token-xyz"); err != nil {
		t.Fatalf("CreateResetToken: %v", err)
	}

	const newPW = "N3wStr0ngPass"
	rr := e.do(t, http.HandlerFunc(e.authH.ResetPassword), http.MethodPost, "/api/auth/reset-password",
		reqOpts{body: map[string]any{"token": "raw-reset-token-xyz", "password": newPW}})
	if rr.Code != http.StatusOK {
		t.Fatalf("reset: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	// The new password must now authenticate.
	login := e.do(t, http.HandlerFunc(e.authH.Login), http.MethodPost, "/api/auth/login",
		reqOpts{body: map[string]any{"username": "resetme", "password": newPW}})
	if login.Code != http.StatusOK {
		t.Fatalf("login with new password: want 200, got %d (%s)", login.Code, login.Body.String())
	}
}

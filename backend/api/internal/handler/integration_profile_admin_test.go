package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

func TestIntegrationAdminMeReturnsProfile(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "meadmin", "admin")

	rr := e.do(t, e.wrapUserAuth(e.authH.Me), http.MethodGet, "/api/users/me", reqOpts{sess: sess})
	if rr.Code != http.StatusOK {
		t.Fatalf("me: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var body struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	decode(t, rr, &body)
	if body.Username != "meadmin" {
		t.Errorf("username = %q, want meadmin", body.Username)
	}
	if body.Role != "admin" {
		t.Errorf("role = %q, want admin", body.Role)
	}
}

// TestIntegrationAdminUpdateProfileHappyNoPassword pins that admin profile
// update needs NO current_password (unlike the storefront customer path).
func TestIntegrationAdminUpdateProfileHappyNoPassword(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "upadmin", "admin")

	rr := e.do(t, e.wrapUserCSRF(e.authH.UpdateProfile), http.MethodPut, "/api/users/me/profile",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{"display_name": "New Name"},
		})
	if rr.Code != http.StatusOK {
		t.Fatalf("update profile: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var ok struct {
		OK bool `json:"ok"`
	}
	decode(t, rr, &ok)
	if !ok.OK {
		t.Errorf("update profile body = %q, want {ok:true}", rr.Body.String())
	}

	// Verify the change is persisted.
	me := e.do(t, e.wrapUserAuth(e.authH.Me), http.MethodGet, "/api/users/me", reqOpts{sess: sess})
	if me.Code != http.StatusOK {
		t.Fatalf("me after update: want 200, got %d (%s)", me.Code, me.Body.String())
	}
	var body struct {
		DisplayName string `json:"display_name"`
	}
	decode(t, me, &body)
	if body.DisplayName != "New Name" {
		t.Errorf("display_name = %q, want New Name", body.DisplayName)
	}
}

// TestIntegrationAdminUpdateProfileValidation: an empty body and an unknown
// field are both rejected with 400.
func TestIntegrationAdminUpdateProfileValidation(t *testing.T) {
	e := newEnv(t)
	sess := e.userSession(t, "upvalid", "admin")

	empty := e.do(t, e.wrapUserCSRF(e.authH.UpdateProfile), http.MethodPut, "/api/users/me/profile",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{},
		})
	if empty.Code != http.StatusBadRequest {
		t.Fatalf("empty body: want 400, got %d (%s)", empty.Code, empty.Body.String())
	}

	bogus := e.do(t, e.wrapUserCSRF(e.authH.UpdateProfile), http.MethodPut, "/api/users/me/profile",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{"bogus": "x"},
		})
	if bogus.Code != http.StatusBadRequest {
		t.Fatalf("unknown field: want 400, got %d (%s)", bogus.Code, bogus.Body.String())
	}
}

func TestIntegrationAdminChangePasswordHappy(t *testing.T) {
	e := newEnv(t)
	sess := e.adminUserWithPassword(t, "cpadmin", "admin")

	rr := e.do(t, e.wrapUserCSRF(e.authH.ChangePassword), http.MethodPut, "/api/users/me/password",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{"current_password": goodPassword, "new_password": "N3wStr0ngPass"},
		})
	if rr.Code != http.StatusOK {
		t.Fatalf("change password: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var ok struct {
		OK bool `json:"ok"`
	}
	decode(t, rr, &ok)
	if !ok.OK {
		t.Errorf("body = %q, want {ok:true}", rr.Body.String())
	}
}

func TestIntegrationAdminChangePasswordWrongCurrent(t *testing.T) {
	e := newEnv(t)
	sess := e.adminUserWithPassword(t, "cpwrong", "admin")

	rr := e.do(t, e.wrapUserCSRF(e.authH.ChangePassword), http.MethodPut, "/api/users/me/password",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{"current_password": "WrongPass0rd", "new_password": "N3wStr0ngPass"},
		})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong current: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("code = %q, want AUTH_INVALID", env.Code)
	}
}

// TestIntegrationAdminChangePasswordInvalidatesOldToken: after a password
// change, a token issued before password_changed_at is rejected; one issued at
// or after the epoch is accepted. Deterministic — drives iat explicitly, no
// time.Sleep. Note UserPasswordChangedAt is keyed by username, not user id.
func TestIntegrationAdminChangePasswordInvalidatesOldToken(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	sess := e.adminUserWithPassword(t, "cpinval", "admin")

	cp := e.do(t, e.wrapUserCSRF(e.authH.ChangePassword), http.MethodPut, "/api/users/me/password",
		reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.AdminCSRFCookie,
			body:           map[string]any{"current_password": goodPassword, "new_password": "N3wStr0ngPass"},
		})
	if cp.Code != http.StatusOK {
		t.Fatalf("change password: want 200, got %d (%s)", cp.Code, cp.Body.String())
	}

	changedAt, ok, err := e.st.UserPasswordChangedAt(ctx, "cpinval")
	if err != nil || !ok {
		t.Fatalf("UserPasswordChangedAt: ok=%v err=%v", ok, err)
	}
	epoch := changedAt.Unix()

	// Token issued 5s before the change → revoked (iat < epoch).
	before := e.userSessionWithIat(t, "cpinval", epoch-5)
	rejected := e.do(t, e.wrapUserAuth(e.authH.Me), http.MethodGet, "/api/users/me", reqOpts{sess: before})
	if rejected.Code != http.StatusUnauthorized {
		t.Fatalf("pre-change token: want 401, got %d (%s)", rejected.Code, rejected.Body.String())
	}

	// Token issued 5s after the change → accepted.
	after := e.userSessionWithIat(t, "cpinval", epoch+5)
	accepted := e.do(t, e.wrapUserAuth(e.authH.Me), http.MethodGet, "/api/users/me", reqOpts{sess: after})
	if accepted.Code != http.StatusOK {
		t.Fatalf("post-change token: want 200, got %d (%s)", accepted.Code, accepted.Body.String())
	}
}

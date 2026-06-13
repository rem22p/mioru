package handler_test

import (
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

const goodPassword = "Tr0ub4dourX9"

func registerBody(email string) map[string]any {
	return map[string]any{"email": email, "password": goodPassword, "first_name": "Reg"}
}

func TestIntegrationCustomerRegisterHappy(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody("reg-happy@ex.com")})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	var prof struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	decode(t, rr, &prof)
	if prof.ID == 0 || prof.Email != "reg-happy@ex.com" {
		t.Errorf("unexpected profile: %+v", prof)
	}
	// Bootstrap cookies must be set.
	sess := sessionFromResponse(t, rr, cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie)
	if sess.csrfValue == "" {
		t.Error("register set no store_csrf cookie value")
	}
}

func TestIntegrationCustomerRegisterDuplicate(t *testing.T) {
	e := newEnv(t)
	first := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody("dup@ex.com")})
	if first.Code != http.StatusCreated {
		t.Fatalf("first register: want 201, got %d (%s)", first.Code, first.Body.String())
	}
	second := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody("dup@ex.com")})
	if second.Code != http.StatusConflict {
		t.Fatalf("dup register: want 409, got %d (%s)", second.Code, second.Body.String())
	}
	var env errEnvelope
	decode(t, second, &env)
	if env.Code != "CONFLICT" {
		t.Errorf("code = %q, want CONFLICT", env.Code)
	}
}

func TestIntegrationCustomerRegisterValidation(t *testing.T) {
	e := newEnv(t)
	// Missing first_name → 400.
	rr := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: map[string]any{"email": "noname@ex.com", "password": goodPassword}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing first_name: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

func TestIntegrationCustomerLoginHappyAndReuseCookie(t *testing.T) {
	e := newEnv(t)
	const email = "login-happy@ex.com"
	reg := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody(email)})
	if reg.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%s)", reg.Code, reg.Body.String())
	}
	login := e.do(t, http.HandlerFunc(e.customerH.Login), http.MethodPost, "/api/store/auth/login",
		reqOpts{body: map[string]any{"email": email, "password": goodPassword}})
	if login.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d (%s)", login.Code, login.Body.String())
	}
	// The cookies login minted must authenticate a real protected call.
	sess := sessionFromResponse(t, login, cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie)
	me := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{sess: sess})
	if me.Code != http.StatusOK {
		t.Fatalf("me with login cookie: want 200, got %d (%s)", me.Code, me.Body.String())
	}
}

func TestIntegrationCustomerLoginWrongPassword(t *testing.T) {
	e := newEnv(t)
	const email = "login-wrong@ex.com"
	if rr := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody(email)}); rr.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	rr := e.do(t, http.HandlerFunc(e.customerH.Login), http.MethodPost, "/api/store/auth/login",
		reqOpts{body: map[string]any{"email": email, "password": "WrongPass0rd"}})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong pw: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("code = %q, want AUTH_INVALID", env.Code)
	}
}

func TestIntegrationCustomerLoginMissingUser(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.customerH.Login), http.MethodPost, "/api/store/auth/login",
		reqOpts{body: map[string]any{"email": "ghost@ex.com", "password": goodPassword}})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing user: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationCustomerLogoutCSRFGate(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "logout@ex.com")
	// Bad CSRF → 403.
	bad := e.do(t, e.wrapCustomer(e.customerH.Logout), http.MethodPost, "/api/store/auth/logout",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, badCSRF: true})
	if bad.Code != http.StatusForbidden {
		t.Errorf("logout bad CSRF: want 403, got %d (%s)", bad.Code, bad.Body.String())
	}
	// Valid CSRF → 200.
	ok := e.do(t, e.wrapCustomer(e.customerH.Logout), http.MethodPost, "/api/store/auth/logout",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie})
	if ok.Code != http.StatusOK {
		t.Errorf("logout valid CSRF: want 200, got %d (%s)", ok.Code, ok.Body.String())
	}
}

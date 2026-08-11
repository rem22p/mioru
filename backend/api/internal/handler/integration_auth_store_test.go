package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
	"mioru/internal/model"
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
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("code = %q, want AUTH_INVALID", env.Code)
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

// TestIntegrationCustomerLoginReportsTelegramBinding pins the login payload
// against the storefront contract: the SPA seeds its auth state straight from
// this body and only refetches /me on a cold start, so a stale linked:false
// here locks a customer with a valid binding out of checkout.
func TestIntegrationCustomerLoginReportsTelegramBinding(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	reg := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody("tg-bound@ex.com")})
	if reg.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%s)", reg.Code, reg.Body.String())
	}
	c, err := e.st.GetCustomerByEmail(ctx, "tg-bound@ex.com")
	if err != nil || c == nil {
		t.Fatalf("GetCustomerByEmail: %v / %v", c, err)
	}
	if err := e.st.LinkOAuth(ctx, c.ID, model.CustomerOAuth{
		Provider:    "telegram",
		OAuthID:     "tg-login-contract",
		ProfileData: `{"username":"neo","first_name":"Neo"}`,
	}); err != nil {
		t.Fatalf("LinkOAuth(telegram): %v", err)
	}

	rr := e.do(t, http.HandlerFunc(e.customerH.Login), http.MethodPost, "/api/store/auth/login",
		reqOpts{body: map[string]any{"email": "tg-bound@ex.com", "password": goodPassword}})
	if rr.Code != http.StatusOK {
		t.Fatalf("login: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var prof struct {
		Telegram *struct {
			Linked    bool   `json:"linked"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"telegram"`
	}
	decode(t, rr, &prof)
	if prof.Telegram == nil {
		t.Fatalf("login response carries no telegram field: %s", rr.Body.String())
	}
	if !prof.Telegram.Linked {
		t.Fatalf("login telegram.linked = false, want true (%s)", rr.Body.String())
	}
	if prof.Telegram.Username != "neo" || prof.Telegram.FirstName != "Neo" {
		t.Errorf("telegram summary = %+v, want username=neo first_name=Neo", *prof.Telegram)
	}
}

// TestIntegrationCustomerRegisterReportsNoTelegram is the counterpart: a fresh
// account has no binding, so the storefront must gate checkout.
func TestIntegrationCustomerRegisterReportsNoTelegram(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody("tg-unbound@ex.com")})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	var prof struct {
		Telegram *struct {
			Linked bool `json:"linked"`
		} `json:"telegram"`
	}
	decode(t, rr, &prof)
	if prof.Telegram == nil || prof.Telegram.Linked {
		t.Errorf("register telegram = %+v, want linked=false", prof.Telegram)
	}
}

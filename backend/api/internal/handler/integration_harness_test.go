package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/email"
	"mioru/internal/handler"
	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
	"mioru/internal/storetest"
	"mioru/internal/telegram"
)

// testSecret is the HS256 secret used to mint tokens in integration tests.
const testSecret = "integration-test-secret-key-32chars-min"

const tokenExpiryMin = 1440 // arbitrary; the harness does not test token expiry

// env bundles the real store and the handlers under test, all sharing one
// throwaway database.
type env struct {
	st             *store.PostgresStore
	customerH      *handler.CustomerHandler
	authH          *handler.AuthHandler
	productH       *handler.ProductHandler
	storeH         *handler.StoreHandler
	adminOrdH      *handler.AdminOrderHandler
	adminCustomerH *handler.AdminCustomerHandler
	adminTelegramH *handler.AdminTelegramHandler
}

// newEnv builds an env on a fresh test database. Handlers are constructed with
// the same constructors main.go uses, with secure=false (dev), nil Telegram
// notifier (no network), and an in-memory email service.
func newEnv(t *testing.T) *env {
	t.Helper()
	st := storetest.Fresh(t)
	// Non-nil notifier so adminTelegramH.Diagnose can call
	// LastHealthCheck / BotTokenSet without nil deref.
	// Empty token + no chat IDs → no network calls.
	tgNotifier := telegram.NewNotifier("", nil, "", t.TempDir(), "", "")
	// separate upload dirs: each handler gets its own root
	return &env{
		st:             st,
		customerH:      handler.NewCustomerHandler(st, testSecret, tokenExpiryMin, false, "", "", t.TempDir(), tgNotifier),
		authH:          handler.NewAuthHandler(st, email.NewService(), testSecret, tokenExpiryMin, false, ""),
		productH:       handler.NewProductHandler(st, t.TempDir()),
		storeH:         handler.NewStoreHandler(st),
		adminOrdH:      handler.NewAdminOrderHandler(st),
		adminCustomerH: handler.NewAdminCustomerHandler(st),
		adminTelegramH: handler.NewAdminTelegramHandler(st, tgNotifier),
	}
}

// getRole resolves a user's role from the DB — mirrors main.go's getRole.
func (e *env) getRole(ctx context.Context, username string) (string, error) {
	u, err := e.st.GetUser(ctx, username)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", nil
	}
	return u.Role, nil
}

// --- middleware wrappers (mirror main.go's composition) ---

func (e *env) wrapCustomer(h http.HandlerFunc) http.Handler {
	mw := middleware.CustomerAuthMW(testSecret, e.st.CustomerPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.StoreCSRFCookie)
	return mw(csrf(h))
}

func (e *env) wrapAdmin(h http.HandlerFunc) http.Handler {
	mw := middleware.AuthMW(testSecret, e.st.UserPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.AdminCSRFCookie)
	return mw(middleware.RequireAdmin(e.getRole)(csrf(h)))
}

func (e *env) wrapSuperAdmin(h http.HandlerFunc) http.Handler {
	mw := middleware.AuthMW(testSecret, e.st.UserPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.AdminCSRFCookie)
	return mw(middleware.RequireSuperAdmin(e.getRole)(csrf(h)))
}

// wrapUserAuth wraps a handler with AuthMW only — no RequireAdmin, no CSRF.
// Mirrors main.go's `GET /api/users/me` (authMW(authH.Me)): the admin profile
// read needs a valid session but is neither role-gated nor a mutation.
func (e *env) wrapUserAuth(h http.HandlerFunc) http.Handler {
	return middleware.AuthMW(testSecret, e.st.UserPasswordChangedAt)(h)
}

// wrapUserCSRF wraps a handler with AuthMW + CSRF (admin CSRF cookie) but NO
// RequireAdmin. Mirrors main.go's `PUT /api/users/me/profile` and
// `PUT /api/users/me/password` (authMW(adminCSRF(handler))) and the admin
// logout — auth + CSRF, not role-gated.
func (e *env) wrapUserCSRF(h http.HandlerFunc) http.Handler {
	mw := middleware.AuthMW(testSecret, e.st.UserPasswordChangedAt)
	csrf := middleware.CSRF(cookieauth.AdminCSRFCookie)
	return mw(csrf(h))
}

// --- session fixtures ---

type session struct {
	authCookie *http.Cookie
	csrfValue  string // value echoed in X-CSRF-Token and the csrf cookie
}

// customerSession creates a customer row and mints a real customer JWT cookie.
// The customer gets a Telegram binding so order-related tests pass the
// TELEGRAM_REQUIRED gate by default. Tests that exercise the gate itself
// build a customer without a binding directly (see
// TestIntegrationCreateOrderRequiresTelegram).
func (e *env) customerSession(t *testing.T, email string) (*session, int64) {
	t.Helper()
	ctx := context.Background()
	if err := e.st.CreateCustomer(ctx, model.Customer{Email: email, HashedPW: "x", FirstName: "T"}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	c, err := e.st.GetCustomerByEmail(ctx, email)
	if err != nil || c == nil {
		t.Fatalf("GetCustomerByEmail: %v / %v", c, err)
	}
	if err := e.st.LinkOAuth(ctx, c.ID, model.CustomerOAuth{
		Provider:    "telegram",
		OAuthID:     fmt.Sprintf("tg-%d", c.ID),
		ProfileData: `{"username":"testuser","first_name":"Test"}`,
	}); err != nil {
		t.Fatalf("LinkOAuth(telegram): %v", err)
	}
	tok, err := auth.CreateToken(strconv.FormatInt(c.ID, 10), auth.TokenTypeCustomer, testSecret, tokenExpiryMin)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return &session{
		authCookie: &http.Cookie{Name: cookieauth.StoreAuthCookie, Value: tok},
		csrfValue:  "csrf-customer-token",
	}, c.ID
}

// userSession creates an admin/staff user row and mints a real user JWT cookie.
func (e *env) userSession(t *testing.T, username, role string) *session {
	t.Helper()
	ctx := context.Background()
	if err := e.st.CreateUser(ctx, model.User{Username: username, Email: username + "@ex.com", HashedPW: "x", Role: role}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok, err := auth.CreateToken(username, auth.TokenTypeUser, testSecret, tokenExpiryMin)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return &session{
		authCookie: &http.Cookie{Name: cookieauth.AdminAuthCookie, Value: tok},
		csrfValue:  "csrf-admin-token",
	}
}

// adminUserWithPassword inserts an admin/staff user whose stored password is
// goodPassword (a real bcrypt hash, unlike userSession's unusable "x"), and
// mints a valid session for it. Use for endpoints that verify the password
// (login, change-password).
func (e *env) adminUserWithPassword(t *testing.T, username, role string) *session {
	t.Helper()
	ctx := context.Background()
	hash, err := auth.HashPassword(goodPassword)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := e.st.CreateUser(ctx, model.User{Username: username, Email: username + "@ex.com", HashedPW: hash, FirstName: "A", Role: role}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	tok, err := auth.CreateToken(username, auth.TokenTypeUser, testSecret, tokenExpiryMin)
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}
	return &session{authCookie: &http.Cookie{Name: cookieauth.AdminAuthCookie, Value: tok}, csrfValue: "csrf-admin-token"}
}

// userSessionWithIat mints a user session whose JWT carries an explicit iat
// (Unix seconds), so a test can place the token deterministically before or
// after an account's password_changed_at epoch. The user-token subject is the
// username (auth.CreateToken in handler.Login passes req.Username; AuthMW reads
// it back via middleware.Username), so sub is the username string here too.
func (e *env) userSessionWithIat(t *testing.T, username string, iat int64) *session {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"typ": auth.TokenTypeUser,
		"iat": iat,
		"exp": iat + 86400,
	})
	s, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return &session{authCookie: &http.Cookie{Name: cookieauth.AdminAuthCookie, Value: s}, csrfValue: "csrf-admin-token"}
}

// sessionFromResponse builds a session from the Set-Cookie headers a bootstrap
// endpoint (register/login) wrote, so a test can reuse the real minted cookies
// on subsequent authenticated calls.
func sessionFromResponse(t *testing.T, rr *httptest.ResponseRecorder, authCookieName, csrfCookieName string) *session {
	t.Helper()
	var authCookie *http.Cookie
	var csrf string
	for _, c := range rr.Result().Cookies() {
		switch c.Name {
		case authCookieName:
			authCookie = c
		case csrfCookieName:
			csrf = c.Value
		}
	}
	if authCookie == nil {
		t.Fatalf("response set no %q cookie; cookies=%v", authCookieName, rr.Result().Cookies())
	}
	return &session{authCookie: authCookie, csrfValue: csrf}
}

// customerSessionWithIat builds a customer session whose JWT carries an explicit
// iat (Unix seconds), so a test can place the token deterministically before or
// after an account's password_changed_at epoch without depending on wall-clock
// second boundaries. exp is set far in the future so only the iat/epoch check
// matters. Built with the same jwt library + claim shape as auth.CreateToken.
func customerSessionWithIat(t *testing.T, customerID, iat int64) *session {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": strconv.FormatInt(customerID, 10),
		"typ": auth.TokenTypeCustomer,
		"iat": iat,
		"exp": iat + 86400,
	})
	s, err := tok.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return &session{
		authCookie: &http.Cookie{Name: cookieauth.StoreAuthCookie, Value: s},
		csrfValue:  "csrf-customer-token",
	}
}

// --- request driver ---

type reqOpts struct {
	sess           *session
	csrfCookieName string // StoreCSRFCookie/AdminCSRFCookie — sends the CSRF cookie AND a matching X-CSRF-Token header (a valid mutation)
	badCSRF        bool   // with csrfCookieName set: sends the cookie but a wrong X-CSRF-Token header (mismatch → expect 403)
	idempotencyKey string
	body           any
	pathValues     map[string]string // applied via req.SetPathValue (for {slug}/{id} routes)
}

// do builds a request, applies cookies/headers per opts, runs it through h via
// a recorder, and returns the recorded result.
func (e *env) do(t *testing.T, h http.Handler, method, target string, o reqOpts) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if o.body != nil {
		b, err := json.Marshal(o.body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, bodyReader)
	if o.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.sess != nil {
		req.AddCookie(o.sess.authCookie)
		if o.csrfCookieName != "" {
			req.AddCookie(&http.Cookie{Name: o.csrfCookieName, Value: o.sess.csrfValue})
			if o.badCSRF {
				req.Header.Set("X-CSRF-Token", "wrong-value")
			} else {
				req.Header.Set("X-CSRF-Token", o.sess.csrfValue)
			}
		}
	}
	if o.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", o.idempotencyKey)
	}
	for k, v := range o.pathValues {
		req.SetPathValue(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// decode unmarshals the recorder body into v.
func decode(t *testing.T, rr *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body %q: %v", rr.Body.String(), err)
	}
}

// errEnvelope is the standard error response shape.
type errEnvelope struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// TestHarnessCustomerSessionAuthenticates is a smoke test: a minted customer
// cookie must pass CustomerAuthMW and reach the handler (GET me → 200).
func TestHarnessCustomerSessionAuthenticates(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "smoke@ex.com")
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{sess: sess})
	if rr.Code != http.StatusOK {
		t.Fatalf("authenticated GET me: want 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
}

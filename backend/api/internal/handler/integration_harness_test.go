package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
	st        *store.PostgresStore
	customerH *handler.CustomerHandler
	authH     *handler.AuthHandler
	productH  *handler.ProductHandler
	storeH    *handler.StoreHandler
	adminOrdH *handler.AdminOrderHandler
}

// newEnv builds an env on a fresh test database. Handlers are constructed with
// the same constructors main.go uses, with secure=false (dev), nil Telegram
// notifier (no network), and an in-memory email service.
func newEnv(t *testing.T) *env {
	t.Helper()
	st := storetest.Fresh(t)
	var tgNotifier *telegram.Notifier // nil — no network in tests
	// separate upload dirs: each handler gets its own root
	return &env{
		st:        st,
		customerH: handler.NewCustomerHandler(st, testSecret, tokenExpiryMin, false, "", "", t.TempDir(), tgNotifier),
		authH:     handler.NewAuthHandler(st, email.NewService(), testSecret, tokenExpiryMin, false, ""),
		productH:  handler.NewProductHandler(st, t.TempDir()),
		storeH:    handler.NewStoreHandler(st),
		adminOrdH: handler.NewAdminOrderHandler(st),
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

// --- session fixtures ---

type session struct {
	authCookie *http.Cookie
	csrfValue  string // value echoed in X-CSRF-Token and the csrf cookie
}

// customerSession creates a customer row and mints a real customer JWT cookie.
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

// --- request driver ---

type reqOpts struct {
	sess           *session
	csrfCookieName string            // StoreCSRFCookie/AdminCSRFCookie — sends the CSRF cookie AND a matching X-CSRF-Token header (a valid mutation)
	badCSRF        bool              // with csrfCookieName set: sends the cookie but a wrong X-CSRF-Token header (mismatch → expect 403)
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

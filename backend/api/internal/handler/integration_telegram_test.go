package handler_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/handler"
)

// testBotToken is the Telegram bot token configured on the test handler when a
// configured-bot path is exercised. signTelegram signs payloads against it.
const testBotToken = "test-bot-token:ABC123"

// signTelegram mirrors auth.VerifyTelegramAuth's data-check-string exactly:
// sorted keys, "key=value" joined by '\n' (no trailing newline), HMAC-SHA256
// with secret = SHA256(botToken), hex-encoded. Empty optional fields are
// excluded, per the Telegram Login Widget spec. Replicated here because the
// white-box helper in package auth cannot be imported from a _test package.
func signTelegram(botToken string, d auth.TelegramAuthData) string {
	pairs := map[string]string{
		"auth_date":  strconv.FormatInt(d.AuthDate, 10),
		"first_name": d.FirstName,
		"id":         strconv.FormatInt(d.ID, 10),
	}
	if d.LastName != "" {
		pairs["last_name"] = d.LastName
	}
	if d.Username != "" {
		pairs["username"] = d.Username
	}
	if d.PhotoURL != "" {
		pairs["photo_url"] = d.PhotoURL
	}
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + "=" + pairs[k])
	}
	secretHash := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretHash[:])
	mac.Write([]byte(sb.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// tgHandler builds a CustomerHandler with testBotToken configured, so the
// Telegram bootstrap/link paths run signature verification (the env default
// customerH has an empty bot token → 503).
func tgHandler(e *env, t *testing.T) *handler.CustomerHandler {
	t.Helper()
	return handler.NewCustomerHandler(e.st, testSecret, tokenExpiryMin, false, testBotToken, "", t.TempDir(), nil)
}

// customerProfile mirrors the handler's customerProfileResp shape for decoding
// (the handler's type is unexported, so a _test package cannot reference it).
type customerProfile struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// telegramBody assembles a JSON body for the telegram login/link payload.
type telegramBody struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// TestIntegrationTelegramLoginNotConfigured: an unconfigured bot token short-
// circuits to 503 before any verification.
func TestIntegrationTelegramLoginNotConfigured(t *testing.T) {
	e := newEnv(t) // customerH has bot token ""
	body := telegramBody{ID: 100, FirstName: "Ann", AuthDate: time.Now().Unix(), Hash: strings.Repeat("a", 64)}
	rr := e.do(t, http.HandlerFunc(e.customerH.TelegramLogin), http.MethodPost, "/api/store/auth/telegram", reqOpts{body: body})
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("not configured: want 503, got %d (body %q)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationTelegramLoginFakeHash: a well-formed payload with a bogus hash
// fails signature verification → 401 AUTH_INVALID.
func TestIntegrationTelegramLoginFakeHash(t *testing.T) {
	e := newEnv(t)
	tgH := tgHandler(e, t)
	body := telegramBody{ID: 200, FirstName: "Bob", AuthDate: time.Now().Unix(), Hash: strings.Repeat("0", 64)}
	rr := e.do(t, http.HandlerFunc(tgH.TelegramLogin), http.MethodPost, "/api/store/auth/telegram", reqOpts{body: body})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("fake hash: want 401, got %d (body %q)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Fatalf("fake hash: want code AUTH_INVALID, got %q", env.Code)
	}
}

// TestIntegrationTelegramLoginNewCustomer: a correctly signed, never-before-seen
// Telegram identity creates an OAuth-only customer and issues a session.
// New-customer path returns 201 (handler sets http.StatusCreated explicitly).
func TestIntegrationTelegramLoginNewCustomer(t *testing.T) {
	e := newEnv(t)
	tgH := tgHandler(e, t)

	authDate := time.Now().Unix()
	d := auth.TelegramAuthData{ID: 300, FirstName: "Cara", AuthDate: authDate}
	body := telegramBody{ID: d.ID, FirstName: d.FirstName, AuthDate: d.AuthDate, Hash: signTelegram(testBotToken, d)}

	rr := e.do(t, http.HandlerFunc(tgH.TelegramLogin), http.MethodPost, "/api/store/auth/telegram", reqOpts{body: body})
	if rr.Code != http.StatusCreated {
		t.Fatalf("new customer: want 201, got %d (body %q)", rr.Code, rr.Body.String())
	}

	// A store_auth session cookie must be set.
	sess := sessionFromResponse(t, rr, cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie)
	if sess.authCookie.Value == "" {
		t.Fatalf("new customer: empty store_auth cookie value")
	}

	var profile customerProfile
	decode(t, rr, &profile)
	if profile.FirstName != "Cara" {
		t.Fatalf("new customer: want first_name Cara, got %q", profile.FirstName)
	}
	if profile.ID <= 0 {
		t.Fatalf("new customer: want positive id, got %d", profile.ID)
	}
}

// TestIntegrationTelegramLoginExistingLink: signing in twice with the same
// Telegram identity returns the existing customer (200, not a duplicate) the
// second time, and the same customer id both times.
func TestIntegrationTelegramLoginExistingLink(t *testing.T) {
	e := newEnv(t)
	tgH := tgHandler(e, t)

	authDate := time.Now().Unix()
	d := auth.TelegramAuthData{ID: 400, FirstName: "Dee", AuthDate: authDate}
	hash := signTelegram(testBotToken, d)
	body := telegramBody{ID: d.ID, FirstName: d.FirstName, AuthDate: d.AuthDate, Hash: hash}

	// First call — creates the customer (201).
	rr1 := e.do(t, http.HandlerFunc(tgH.TelegramLogin), http.MethodPost, "/api/store/auth/telegram", reqOpts{body: body})
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first login: want 201, got %d (body %q)", rr1.Code, rr1.Body.String())
	}
	var p1 customerProfile
	decode(t, rr1, &p1)

	// Second call — existing-link branch returns 200 + cookies, no new customer.
	rr2 := e.do(t, http.HandlerFunc(tgH.TelegramLogin), http.MethodPost, "/api/store/auth/telegram", reqOpts{body: body})
	if rr2.Code != http.StatusOK {
		t.Fatalf("second login: want 200 (existing link), got %d (body %q)", rr2.Code, rr2.Body.String())
	}
	sess := sessionFromResponse(t, rr2, cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie)
	if sess.authCookie.Value == "" {
		t.Fatalf("second login: empty store_auth cookie value")
	}
	var p2 customerProfile
	decode(t, rr2, &p2)

	if p1.ID != p2.ID {
		t.Fatalf("existing link: customer id changed across logins (%d vs %d) — duplicate created", p1.ID, p2.ID)
	}
}

// TestIntegrationLinkOAuthTelegramRejectsUnsigned is the SECURITY INVARIANT: an
// authenticated customer must NOT be able to bind a Telegram identity they do
// not own by supplying an unsigned/forged hash. The link path must verify the
// signature exactly like login does, or any logged-in user can claim a victim's
// Telegram id and hijack their later login. Forged hash → 401 AUTH_INVALID.
func TestIntegrationLinkOAuthTelegramRejectsUnsigned(t *testing.T) {
	e := newEnv(t)
	tgH := tgHandler(e, t)
	sess, _ := e.customerSession(t, "linker@ex.com")

	body := map[string]any{
		"provider":   "telegram",
		"id":         777,
		"auth_date":  time.Now().Unix(),
		"first_name": "X",
		"hash":       strings.Repeat("f", 64), // forged — not signed with the bot token
	}
	rr := e.do(t, e.wrapCustomer(tgH.LinkOAuth), http.MethodPost, "/api/store/customers/me/oauth",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, body: body})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("SECURITY: unsigned telegram link must be rejected — want 401, got %d (body %q). "+
			"200 here is a CRITICAL identity-hijack hole.", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Fatalf("unsigned telegram link: want code AUTH_INVALID, got %q", env.Code)
	}
}

// TestIntegrationLinkOAuthNonTelegramHappy: a non-Telegram provider links via a
// bare oauth_id (no signature) → 200 {"ok":true}.
func TestIntegrationLinkOAuthNonTelegramHappy(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "googler@ex.com")

	body := map[string]any{
		"provider":     "google",
		"oauth_id":     "g-123",
		"profile_data": "{}",
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.LinkOAuth), http.MethodPost, "/api/store/customers/me/oauth",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, body: body})
	if rr.Code != http.StatusOK {
		t.Fatalf("google link: want 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
	var resp struct {
		OK bool `json:"ok"`
	}
	decode(t, rr, &resp)
	if !resp.OK {
		t.Fatalf("google link: want ok=true, got body %q", rr.Body.String())
	}
}

// TestIntegrationLinkOAuthTelegramProfileDataIsServerBuilt is the trust-boundary
// invariant on the *contents* of the binding: the signature covers
// id/first_name/last_name/username/photo_url, so those fields — not an
// unverified client blob — must land in profile_data. Otherwise a customer
// links their own (validly signed) Telegram while labelling it with someone
// else's handle, and the admin console shows that handle next to a real
// chat id: the manager then contacts an account of the customer's choosing.
// TelegramLogin already builds the record server-side; the link path must match.
func TestIntegrationLinkOAuthTelegramProfileDataIsServerBuilt(t *testing.T) {
	e := newEnv(t)
	tgH := tgHandler(e, t)
	// Registered, not e.customerSession: that helper pre-links a telegram row,
	// and the summary would report it instead of the one under test.
	rrReg := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost, "/api/store/auth/register",
		reqOpts{body: registerBody("spoofer@ex.com")})
	if rrReg.Code != http.StatusCreated {
		t.Fatalf("register: want 201, got %d (body %q)", rrReg.Code, rrReg.Body.String())
	}
	sess := sessionFromResponse(t, rrReg, cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie)

	authDate := time.Now().Unix()
	d := auth.TelegramAuthData{ID: 909, FirstName: "Real", Username: "real_handle", AuthDate: authDate}
	body := map[string]any{
		"provider":   "telegram",
		"id":         d.ID,
		"first_name": d.FirstName,
		"username":   d.Username,
		"auth_date":  d.AuthDate,
		"hash":       signTelegram(testBotToken, d),
		// Signed payload says "real_handle"; the client claims otherwise.
		"profile_data": `{"username":"mioru_support","first_name":"MIORU Support"}`,
	}
	rr := e.do(t, e.wrapCustomer(tgH.LinkOAuth), http.MethodPost, "/api/store/customers/me/oauth",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, body: body})
	if rr.Code != http.StatusOK {
		t.Fatalf("signed telegram link: want 200, got %d (body %q)", rr.Code, rr.Body.String())
	}

	rrMe := e.do(t, e.wrapCustomer(tgH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{sess: sess})
	if rrMe.Code != http.StatusOK {
		t.Fatalf("me: want 200, got %d (body %q)", rrMe.Code, rrMe.Body.String())
	}
	var prof struct {
		Telegram *struct {
			Linked    bool   `json:"linked"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"telegram"`
	}
	decode(t, rrMe, &prof)
	if prof.Telegram == nil || !prof.Telegram.Linked {
		t.Fatalf("me after link: want linked=true, got %+v", prof.Telegram)
	}
	if prof.Telegram.Username != "real_handle" {
		t.Errorf("SECURITY: stored username = %q, want %q — the unsigned client blob overwrote the signed payload",
			prof.Telegram.Username, "real_handle")
	}
	if prof.Telegram.FirstName != "Real" {
		t.Errorf("SECURITY: stored first_name = %q, want %q", prof.Telegram.FirstName, "Real")
	}
}

// TestIntegrationLinkOAuthOmittedProfileData: profile_data is optional per the
// API contract, so omitting it must succeed — not blow up casting an empty
// string to jsonb inside the INSERT and surface as 500 INTERNAL.
func TestIntegrationLinkOAuthOmittedProfileData(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "no-profile@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.LinkOAuth), http.MethodPost, "/api/store/customers/me/oauth",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie,
			body: map[string]any{"provider": "google", "oauth_id": "g-no-profile"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("link without profile_data: want 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationLinkOAuthMalformedProfileData: a non-JSON blob is caller error,
// so it must be a generic 400 — never a 500 from the driver.
func TestIntegrationLinkOAuthMalformedProfileData(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "bad-profile@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.LinkOAuth), http.MethodPost, "/api/store/customers/me/oauth",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie,
			body: map[string]any{"provider": "google", "oauth_id": "g-bad-profile", "profile_data": "not json at all"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("malformed profile_data: want 400, got %d (body %q)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("malformed profile_data: want code VALIDATION_FAILED, got %q", env.Code)
	}
}

// TestIntegrationLinkOAuthTelegramNotConfigured: with no bot token the server
// cannot verify anything, and answering "неверная подпись" (401) sends the
// customer and support chasing a client-side problem. TelegramLogin already
// answers 503 here; the link path must agree.
func TestIntegrationLinkOAuthTelegramNotConfigured(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "no-bot-token@ex.com")

	d := auth.TelegramAuthData{ID: 4242, FirstName: "X", AuthDate: time.Now().Unix()}
	body := map[string]any{
		"provider":   "telegram",
		"id":         d.ID,
		"first_name": d.FirstName,
		"auth_date":  d.AuthDate,
		"hash":       signTelegram(testBotToken, d),
	}
	// e.customerH is built without a bot token.
	rr := e.do(t, e.wrapCustomer(e.customerH.LinkOAuth), http.MethodPost, "/api/store/customers/me/oauth",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, body: body})
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("telegram link without bot token: want 503, got %d (body %q)", rr.Code, rr.Body.String())
	}
}

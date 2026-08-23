package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
)

// fakeCustomerStore is a minimal customerStore for the cookie-issuance and
// logout tests. It supports the surface Login/Register exercise without
// dragging the full PostgresStore into the test binary.
type fakeCustomerStore struct {
	byEmail map[string]*model.Customer
	byID    map[int64]*model.Customer
	created *model.Customer
}

func (f *fakeCustomerStore) CreateCustomer(ctx context.Context, c model.Customer) error {
	cp := c
	cp.ID = 1
	f.created = &cp
	if f.byEmail == nil {
		f.byEmail = map[string]*model.Customer{}
	}
	if f.byID == nil {
		f.byID = map[int64]*model.Customer{}
	}
	f.byEmail[c.Email] = &cp
	f.byID[cp.ID] = &cp
	return nil
}
func (f *fakeCustomerStore) GetCustomer(ctx context.Context, id int64) (*model.Customer, error) {
	return f.byID[id], nil
}
func (f *fakeCustomerStore) GetCustomerByEmail(ctx context.Context, email string) (*model.Customer, error) {
	return f.byEmail[email], nil
}
func (f *fakeCustomerStore) UpdateCustomer(ctx context.Context, id int64, updates map[string]string) error {
	return nil
}
func (f *fakeCustomerStore) UpdateCustomerPhoneIfChanged(ctx context.Context, id int64, phone string) (int64, error) {
	return 0, nil
}

func (f *fakeCustomerStore) UpdateCustomerPassword(ctx context.Context, id int64, hashedPW string) error {
	return nil
}
func (f *fakeCustomerStore) GetCustomerByOAuth(ctx context.Context, provider, oauthID string) (*model.Customer, *model.CustomerOAuth, error) {
	return nil, nil, nil
}
func (f *fakeCustomerStore) CreateCustomerWithOAuth(ctx context.Context, c model.Customer, oa model.CustomerOAuth) error {
	return nil
}
func (f *fakeCustomerStore) LinkOAuth(ctx context.Context, customerID int64, oa model.CustomerOAuth) error {
	return nil
}
func (f *fakeCustomerStore) GetCustomerOAuth(ctx context.Context, customerID int64) ([]model.CustomerOAuth, error) {
	// Default: a telegram link exists, so CreateOrder passes the gate.
	// Tests that exercise the gate override this in their own fake.
	return []model.CustomerOAuth{{Provider: "telegram", OAuthID: "tg-1"}}, nil
}
func (f *fakeCustomerStore) ListCustomerOrders(ctx context.Context, customerID int64, page, perPage int) ([]model.Order, int, error) {
	return nil, 0, nil
}

func (f *fakeCustomerStore) CreateOrder(ctx context.Context, customerID int64, o *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	return nil, nil
}

func (f *fakeCustomerStore) GetCustomerCart(ctx context.Context, customerID int64) ([]store.CartItem, error) {
	return nil, nil
}

func (f *fakeCustomerStore) SaveCustomerCart(ctx context.Context, customerID int64, items []store.CartItem) error {
	return nil
}

func (f *fakeCustomerStore) GetCustomerFavorites(ctx context.Context, customerID int64) ([]int, error) {
	return nil, nil
}

func (f *fakeCustomerStore) SaveCustomerFavorites(ctx context.Context, customerID int64, productIDs []int) error {
	return nil
}

func (f *fakeCustomerStore) GetOrderByIdempotencyKey(ctx context.Context, key string, customerID int64) (*store.IdempotencyRecord, error) {
	// Default: no race record exists. Tests that exercise the
	// race-loser path override this in their own fake.
	return nil, nil
}

func newCustomerHandlerForTest(fs customerStore) *CustomerHandler {
	return NewCustomerHandler(fs, "test-secret-key-at-least-32-chars-long!!", 60, false, "", "", "", nil)
}

// TestCustomerLoginIssuesCookiesNotAccessToken locks in the cookie-only
// contract for the storefront side: a successful login must set both
// store_auth (HttpOnly) and store_csrf (readable) cookies, and the JSON body
// must NOT contain access_token.
func TestCustomerLoginIssuesCookiesNotAccessToken(t *testing.T) {
	hashed, err := auth.HashPassword("Tr0ubadour-x9")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	fs := &fakeCustomerStore{
		byEmail: map[string]*model.Customer{
			"buyer@example.com": {
				ID:          42,
				Email:       "buyer@example.com",
				HashedPW:    hashed,
				FirstName:   "Bea",
				LastName:    "Yer",
				AvatarColor: "#fff",
			},
		},
	}
	h := newCustomerHandlerForTest(fs)

	body := `{"email":"buyer@example.com","password":"Tr0ubadour-x9"}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/auth/login", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["access_token"]; ok {
		t.Errorf("response must not contain access_token; got %v", resp)
	}
	if resp["email"] != "buyer@example.com" {
		t.Errorf("email = %v, want buyer@example.com", resp["email"])
	}

	got := map[string]*http.Cookie{}
	for _, c := range rr.Result().Cookies() {
		got[c.Name] = c
	}
	authCk, ok := got[cookieauth.StoreAuthCookie]
	if !ok || authCk.Value == "" {
		t.Fatalf("expected %s cookie to be set", cookieauth.StoreAuthCookie)
	}
	if !authCk.HttpOnly {
		t.Errorf("%s must be HttpOnly", cookieauth.StoreAuthCookie)
	}
	csrfCk, ok := got[cookieauth.StoreCSRFCookie]
	if !ok || csrfCk.Value == "" {
		t.Fatalf("expected %s cookie to be set", cookieauth.StoreCSRFCookie)
	}
	if csrfCk.HttpOnly {
		t.Errorf("%s must be readable by JS (HttpOnly=false)", cookieauth.StoreCSRFCookie)
	}
}

// TestCustomerLogoutClearsCookies verifies Logout zeroes both customer
// session cookies via a negative MaxAge.
func TestCustomerLogoutClearsCookies(t *testing.T) {
	h := newCustomerHandlerForTest(&fakeCustomerStore{})

	req := httptest.NewRequest(http.MethodPost, "/api/store/auth/logout", nil)
	rr := httptest.NewRecorder()
	h.Logout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	got := map[string]*http.Cookie{}
	for _, c := range rr.Result().Cookies() {
		got[c.Name] = c
	}
	for _, name := range []string{cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie} {
		c, ok := got[name]
		if !ok {
			t.Fatalf("expected %s clear-cookie", name)
		}
		if c.MaxAge >= 0 {
			t.Errorf("%s MaxAge = %d, want negative (delete)", name, c.MaxAge)
		}
		if c.Value != "" {
			t.Errorf("%s Value = %q, want empty", name, c.Value)
		}
	}
}

// TestLinkOAuthRejectsUnsignedTelegramID guards against account hijack via
// the vulnerable LinkOAuth endpoint (issue #1). A bare oauth_id without a
// valid Telegram HMAC signature must be rejected. This test should FAIL
// before the fix and PASS after.
func TestLinkOAuthRejectsUnsignedTelegramID(t *testing.T) {
	h := newCustomerHandlerForTest(&fakeCustomerStore{})

	// Attempt to link a Telegram ID without any hash/auth_date — this is the
	// exploit path described in issue #1.
	body := `{"provider":"telegram","oauth_id":"123456789"}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/customers/me/oauth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Pretend we are customer 42 (bypass CustomerAuthMW for the test).
	ctx := middleware.WithCustomerID(req.Context(), 42)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	// Also need CSRF bypass — LinkOAuth sits behind customerCSRF in main.go.
	// In tests we call the handler directly, bypassing middleware.
	h.LinkOAuth(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnauthorized {
		t.Errorf("unsigned telegram oauth link must be rejected, got HTTP %d (want 400 or 401)", rr.Code)
	}
}

// TestListOrdersReturnsEmptyWhenNoOrders verifies the handler returns a
// valid empty response (200) when the customer has no orders.
func TestListOrdersReturnsEmptyWhenNoOrders(t *testing.T) {
	h := newCustomerHandlerForTest(&fakeCustomerStore{})

	// Without customer ID in context — should be rejected by handler
	req := httptest.NewRequest(http.MethodGet, "/api/store/customers/me/orders", nil)
	rr := httptest.NewRecorder()
	h.ListOrders(rr, req)

	// CustomerID returns 0 without auth context — handler proceeds but finds no orders
	// (empty list is valid). The auth gate is enforced by CustomerAuthMW in main.go.
	if rr.Code != http.StatusOK {
		t.Errorf("orders without auth context should return 200 (empty), got %d", rr.Code)
	}
}

// fakeCustomerStoreWithRace simulates a race: the first GetCustomerByOAuth
// returns nil (no existing link), then CreateCustomerWithOAuth returns
// ErrOAuthAlreadyLinked (another request won the race).
type fakeCustomerStoreWithRace struct {
	fakeCustomerStore
	createCalled bool
}

func (f *fakeCustomerStoreWithRace) GetCustomerByOAuth(ctx context.Context, provider, oauthID string) (*model.Customer, *model.CustomerOAuth, error) {
	if f.createCalled {
		// After the conflict, return the customer that won the race.
		return &model.Customer{ID: 42, Email: "race@test.com", FirstName: "Race"}, nil, nil
	}
	return nil, nil, nil
}

func (f *fakeCustomerStoreWithRace) CreateCustomerWithOAuth(ctx context.Context, c model.Customer, oa model.CustomerOAuth) error {
	f.createCalled = true
	return store.ErrOAuthAlreadyLinked
}

func TestTelegramLoginRaceRetry(t *testing.T) {
	fs := &fakeCustomerStoreWithRace{}
	h := newCustomerHandlerForTestWithToken(fs, "000000:TEST-TOKEN")

	// Build a signed Telegram auth payload.
	now := time.Now()
	data := auth.TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  now.Unix(),
	}
	data.Hash = signTelegramDataForTest(t, data)

	body := fmt.Sprintf(`{"id":%d,"first_name":"%s","auth_date":%d,"hash":"%s"}`,
		data.ID, data.FirstName, data.AuthDate, data.Hash)

	req := httptest.NewRequest(http.MethodPost, "/api/store/auth/telegram", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.TelegramLogin(rr, req)

	// Race-retry should issue a session for the now-linked customer (ID 42),
	// not return 409 or 500.
	if rr.Code != http.StatusOK {
		t.Errorf("race-retry: want 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var resp customerProfileResp
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 42 {
		t.Errorf("race-retry: want customer ID 42, got %d", resp.ID)
	}
}

func signTelegramDataForTest(t *testing.T, data auth.TelegramAuthData) string {
	t.Helper()
	pairs := map[string]string{
		"auth_date":  fmt.Sprintf("%d", data.AuthDate),
		"first_name": data.FirstName,
		"id":         fmt.Sprintf("%d", data.ID),
	}
	if data.LastName != "" {
		pairs["last_name"] = data.LastName
	}
	if data.Username != "" {
		pairs["username"] = data.Username
	}
	if data.PhotoURL != "" {
		pairs["photo_url"] = data.PhotoURL
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
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(pairs[k])
	}

	secretHash := sha256.Sum256([]byte("000000:TEST-TOKEN"))
	mac := hmac.New(sha256.New, secretHash[:])
	mac.Write([]byte(sb.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

func newCustomerHandlerForTestWithToken(fs customerStore, botToken string) *CustomerHandler {
	return NewCustomerHandler(fs, "test-secret-key-at-least-32-chars-long!!", 60, false, botToken, "", "", nil)
}

func TestPhoneValidation(t *testing.T) {
	// KAN-53: strict +373 + exactly 8 digits. Mirror of the frontend
	// PHONE_RE test (apps/store/src/lib/phoneValidation.test.ts).
	tests := []struct {
		phone string
		valid bool
	}{
		{"+37369123456", true},  // canonical МД mobile
		{"+37360000000", true},  // manager example (ПМР), also the integration-test number
		{"+37368192547", true},  // manager example (МД)
		{"+373 69 123 456", false}, // spaces not allowed
		{"+3736912345", false},  // 7 digits after +373 — too few
		{"+373777908542", false}, // 9 digits after +373 — too many
		{"37369123456", false},  // missing leading +
		{"69123456", false},     // no +373 prefix at all
		{"+769123456", false},   // non-+373 country code (RUS)
		{"+38068192547", false}, // non-+373 country code (UA)
		{"abc", false},
		{"", false},
		{"+", false},
		{"+373", false}, // prefix only, no digits
	}
	for _, tt := range tests {
		got := phoneRE.MatchString(tt.phone)
		if got != tt.valid {
			t.Errorf("phoneRE.MatchString(%q) = %v, want %v", tt.phone, got, tt.valid)
		}
	}
}

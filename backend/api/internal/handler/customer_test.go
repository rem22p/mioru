package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/model"
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

func newCustomerHandlerForTest(fs *fakeCustomerStore) *CustomerHandler {
	return NewCustomerHandler(fs, "test-secret-key-at-least-32-chars-long!!", 60, false, "", "")
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

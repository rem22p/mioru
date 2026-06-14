package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

// registerCustomer bootstraps a real storefront customer via the Register
// endpoint and returns a session built from the minted cookies. Because
// Register hashes goodPassword with bcrypt, the stored password is verifiable
// by the password-checking endpoints (UpdateProfile / ChangePassword), unlike
// customerSession which inserts a non-bcrypt placeholder.
func registerCustomer(t *testing.T, e *env, email string) *session {
	t.Helper()
	reg := e.do(t, http.HandlerFunc(e.customerH.Register), http.MethodPost,
		"/api/store/auth/register", reqOpts{body: registerBody(email)})
	if reg.Code != http.StatusCreated {
		t.Fatalf("register %s: want 201, got %d (%s)", email, reg.Code, reg.Body.String())
	}
	return sessionFromResponse(t, reg, cookieauth.StoreAuthCookie, cookieauth.StoreCSRFCookie)
}

// TestIntegrationCustomerMeReturnsProfile — GET /api/store/customers/me
// returns the authenticated customer's profile.
func TestIntegrationCustomerMeReturnsProfile(t *testing.T) {
	e := newEnv(t)
	const email = "me@ex.com"
	sess := registerCustomer(t, e, email)

	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet,
		"/api/store/customers/me", reqOpts{sess: sess})
	if rr.Code != http.StatusOK {
		t.Fatalf("GET me: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
	}
	decode(t, rr, &resp)
	if resp.Email != email {
		t.Errorf("want email %q, got %q", email, resp.Email)
	}
	if resp.ID == 0 {
		t.Errorf("want non-zero id, got %d", resp.ID)
	}
}

// TestIntegrationCustomerUpdateProfileHappy — PUT me with the correct current
// password updates first_name; a subsequent GET reflects the change.
func TestIntegrationCustomerUpdateProfileHappy(t *testing.T) {
	e := newEnv(t)
	sess := registerCustomer(t, e, "upd@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.UpdateProfile), http.MethodPut,
		"/api/store/customers/me", reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body:           map[string]any{"current_password": goodPassword, "first_name": "Renamed"},
		})
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT me: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var ok struct {
		OK bool `json:"ok"`
	}
	decode(t, rr, &ok)
	if !ok.OK {
		t.Errorf("want {ok:true}, got %s", rr.Body.String())
	}

	// Verify the update landed.
	get := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet,
		"/api/store/customers/me", reqOpts{sess: sess})
	if get.Code != http.StatusOK {
		t.Fatalf("GET me after update: want 200, got %d (%s)", get.Code, get.Body.String())
	}
	var prof struct {
		FirstName string `json:"first_name"`
	}
	decode(t, get, &prof)
	if prof.FirstName != "Renamed" {
		t.Errorf("want first_name Renamed, got %q", prof.FirstName)
	}
}

// TestIntegrationCustomerUpdateProfileWrongPassword — wrong current_password
// is rejected 401 AUTH_INVALID.
func TestIntegrationCustomerUpdateProfileWrongPassword(t *testing.T) {
	e := newEnv(t)
	sess := registerCustomer(t, e, "wrongpw@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.UpdateProfile), http.MethodPut,
		"/api/store/customers/me", reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body:           map[string]any{"current_password": "WrongPass0rd", "first_name": "X"},
		})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("PUT me wrong pw: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("want code AUTH_INVALID, got %q", env.Code)
	}
}

// TestIntegrationCustomerUpdateProfileMissingPassword — a PUT with no
// current_password is rejected before any field is touched. The handler
// (customer.go:324-327) returns 401 AUTH_REQUIRED for the missing-password
// case — NOT VALIDATION_FAILED — because it treats the profile change as a
// re-authentication gate. Assertion pinned to the real code.
func TestIntegrationCustomerUpdateProfileMissingPassword(t *testing.T) {
	e := newEnv(t)
	sess := registerCustomer(t, e, "nopw@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.UpdateProfile), http.MethodPut,
		"/api/store/customers/me", reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body:           map[string]any{"first_name": "X"},
		})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("PUT me missing pw: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_REQUIRED" {
		t.Errorf("want code AUTH_REQUIRED, got %q", env.Code)
	}
}

// TestIntegrationCustomerChangePasswordInvalidatesOldToken — the security
// invariant: a successful password change bumps password_changed_at, and the
// JWT minted BEFORE that bump (iat < password_changed_at) must be rejected on
// the next authenticated request. The change itself must succeed (200) using
// the old session's cookie+CSRF, and only THEN does the old auth cookie stop
// working.
func TestIntegrationCustomerChangePasswordInvalidatesOldToken(t *testing.T) {
	e := newEnv(t)
	const email = "rotate@ex.com"
	sessA := registerCustomer(t, e, email)

	// Resolve the customer id (registerCustomer only returns the session).
	cust, err := e.st.GetCustomerByEmail(context.Background(), email)
	if err != nil || cust == nil {
		t.Fatalf("GetCustomerByEmail: %v / %v", cust, err)
	}
	custID := cust.ID

	// 1. The real password change succeeds through the full middleware chain
	//    with session A — this proves the endpoint works AND bumps the epoch.
	const newPW = "N3wPassw0rdY"
	chg := e.do(t, e.wrapCustomer(e.customerH.ChangePassword), http.MethodPut,
		"/api/store/customers/me/password", reqOpts{
			sess:           sessA,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body: map[string]any{
				"current_password": goodPassword,
				"new_password":     newPW,
			},
		})
	if chg.Code != http.StatusOK {
		t.Fatalf("change password: want 200, got %d (%s)", chg.Code, chg.Body.String())
	}

	// The change must have bumped the account's password_changed_at epoch.
	changedAt, ok, err := e.st.CustomerPasswordChangedAt(context.Background(), custID)
	if err != nil || !ok {
		t.Fatalf("CustomerPasswordChangedAt: ok=%v err=%v", ok, err)
	}
	epoch := changedAt.Unix()

	// 2. A token issued strictly BEFORE the epoch must be rejected — the
	//    security invariant, deterministic regardless of second boundaries.
	stale := customerSessionWithIat(t, custID, epoch-5)
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet,
		"/api/store/customers/me", reqOpts{sess: stale})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("SECURITY INVARIANT VIOLATED: pre-epoch token should be 401, "+
			"got %d (%s)", rr.Code, rr.Body.String())
	}

	// 3. A token issued strictly AFTER the epoch must still authenticate — proves
	//    it's specifically the epoch boundary doing the rejection, not a blanket
	//    failure.
	fresh := customerSessionWithIat(t, custID, epoch+5)
	rr = e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet,
		"/api/store/customers/me", reqOpts{sess: fresh})
	if rr.Code != http.StatusOK {
		t.Fatalf("post-epoch token: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationCustomerSetPasswordRejectsPasswordedCustomer — set-password
// is only for OAuth-only customers (empty hashed_password). A registered
// customer already has a password, so it is rejected 409 CONFLICT.
func TestIntegrationCustomerSetPasswordRejectsPasswordedCustomer(t *testing.T) {
	e := newEnv(t)
	sess := registerCustomer(t, e, "setpw@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.SetPassword), http.MethodPost,
		"/api/store/customers/me/set-password", reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body:           map[string]any{"new_password": "N3wPassw0rdY"},
		})
	if rr.Code != http.StatusConflict {
		t.Fatalf("set-password on passworded customer: want 409, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationCustomerFavoritesRoundTrip — GET favorites starts empty and
// non-null; PUT with a valid (seeded) product id persists; GET reflects it.
// A real product is seeded because customer_favorites.product_id is a FK to
// products(id) (migration 007).
func TestIntegrationCustomerFavoritesRoundTrip(t *testing.T) {
	e := newEnv(t)
	sess := registerCustomer(t, e, "fav@ex.com")
	pid := seedProduct(t, e, "fav-product", 500, 10)

	// GET → empty, non-null.
	get := e.do(t, e.wrapCustomer(e.customerH.GetFavorites), http.MethodGet,
		"/api/store/customers/me/favorites", reqOpts{sess: sess})
	if get.Code != http.StatusOK {
		t.Fatalf("GET favorites: want 200, got %d (%s)", get.Code, get.Body.String())
	}
	// product_ids must be present and non-null (empty array, not JSON null).
	var raw struct {
		ProductIDs *[]int `json:"product_ids"`
	}
	decode(t, get, &raw)
	if raw.ProductIDs == nil {
		t.Fatalf("product_ids should be non-null, got null: %s", get.Body.String())
	}
	if len(*raw.ProductIDs) != 0 {
		t.Errorf("want empty favorites, got %v", *raw.ProductIDs)
	}

	// PUT the seeded id.
	put := e.do(t, e.wrapCustomer(e.customerH.SaveFavorites), http.MethodPut,
		"/api/store/customers/me/favorites", reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body:           map[string]any{"product_ids": []int{int(pid)}},
		})
	if put.Code != http.StatusOK {
		t.Fatalf("PUT favorites: want 200, got %d (%s)", put.Code, put.Body.String())
	}

	// GET → contains the id.
	get2 := e.do(t, e.wrapCustomer(e.customerH.GetFavorites), http.MethodGet,
		"/api/store/customers/me/favorites", reqOpts{sess: sess})
	if get2.Code != http.StatusOK {
		t.Fatalf("GET favorites after save: want 200, got %d (%s)", get2.Code, get2.Body.String())
	}
	var resp struct {
		ProductIDs []int `json:"product_ids"`
	}
	decode(t, get2, &resp)
	found := false
	for _, id := range resp.ProductIDs {
		if int64(id) == pid {
			found = true
		}
	}
	if !found {
		t.Errorf("want favorites to contain %d, got %v", pid, resp.ProductIDs)
	}
}

// TestIntegrationCustomerSaveFavoritesValidation — a non-positive product id
// is rejected 400 (customer.go:1251-1255).
func TestIntegrationCustomerSaveFavoritesValidation(t *testing.T) {
	e := newEnv(t)
	sess := registerCustomer(t, e, "favbad@ex.com")

	rr := e.do(t, e.wrapCustomer(e.customerH.SaveFavorites), http.MethodPut,
		"/api/store/customers/me/favorites", reqOpts{
			sess:           sess,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			body:           map[string]any{"product_ids": []int{0}},
		})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("PUT favorites with id=0: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
}

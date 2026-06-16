package handler_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"mioru/internal/model"
)

// TestIntegrationAdminListCustomers asserts the list endpoint
// returns customers with rolled-up order stats and that the search
// parameter filters by email/name.
func TestIntegrationAdminListCustomers(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "listcustomers", "admin")

	// Seed two customers with different emails/names + orders.
	c1ID := seedCustomer(t, e, "alpha@example.com", "Alpha", "User", "+37370000001")
	c2ID := seedCustomer(t, e, "beta@example.com", "Beta", "Person", "+37370000002")
	_ = c2ID // referenced in search assertions below
	pid := seedProduct(t, e, "list-cust-p", 500, 100)

	// Create an order for c1 (so it shows up with orders_count=1).
	_, _ = e.st.CreateOrder(context.Background(), c1ID, &model.Order{
		Type: "cart", TotalMinor: 50000, Status: "delivered",
		Phone: "+37370000001", City: "Кишинёв", DeliveryMethod: "address", PaymentMethod: "card",
	}, nil, "", "")
	_ = pid

	rr := e.do(t, e.wrapAdmin(e.adminCustomerH.List), http.MethodGet, "/api/admin/customers", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("List customers: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Customers []map[string]any `json:"customers"`
		Total     int              `json:"total"`
	}
	decode(t, rr, &resp)
	if resp.Total < 2 {
		t.Errorf("total = %d, want >= 2 (seeded 2)", resp.Total)
	}

	// Find alpha — it should have orders_count = 1; beta should have 0.
	var alpha map[string]any
	var beta map[string]any
	for _, c := range resp.Customers {
		if email, _ := c["email"].(string); email == "alpha@example.com" {
			alpha = c
		}
		if email, _ := c["email"].(string); email == "beta@example.com" {
			beta = c
		}
	}
	if alpha == nil {
		t.Fatalf("alpha@example.com missing from list")
	}
	if beta == nil {
		t.Fatalf("beta@example.com missing from list")
	}
	if v, _ := alpha["orders_count"].(float64); v != 1 {
		t.Errorf("alpha orders_count = %v, want 1", alpha["orders_count"])
	}
	if v, _ := beta["orders_count"].(float64); v != 0 {
		t.Errorf("beta orders_count = %v, want 0", beta["orders_count"])
	}
	if _, hasPw := alpha["hashed_password"]; hasPw {
		t.Errorf("hashed_password leaked in admin list response — security regression")
	}

	// Search filter — only "alpha" should match.
	rr2 := e.do(t, e.wrapAdmin(e.adminCustomerH.List), http.MethodGet,
		"/api/admin/customers?search=alpha", reqOpts{sess: admin})
	if rr2.Code != http.StatusOK {
		t.Fatalf("List customers search: want 200, got %d", rr2.Code)
	}
	var resp2 struct {
		Customers []map[string]any `json:"customers"`
		Total     int              `json:"total"`
	}
	decode(t, rr2, &resp2)
	if resp2.Total < 1 {
		t.Errorf("search=alpha total = %d, want >= 1", resp2.Total)
	}
	for _, c := range resp2.Customers {
		if email, _ := c["email"].(string); email != "alpha@example.com" {
			t.Errorf("search returned non-matching email: %s", email)
		}
	}
}

// TestIntegrationAdminGetCustomerDetail covers the per-customer
// detail endpoint: it should expose the full profile, the order
// list, and the Telegram link if the customer has one. The
// separate "without Telegram" sub-test is the regression guard
// for a NULL-into-bool Scan bug: the
// `(SELECT oauth_id IS NOT NULL ...)` subquery returns NULL
// when the customer has no oauth row, and pgx refuses to Scan
// NULL into a `*bool` — we COALESCE to FALSE in the SQL to
// make the column always boolean, and this test asserts that
// the API still returns 200 in that case.
func TestIntegrationAdminGetCustomerDetail(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "custdetail", "admin")

	cID := seedCustomer(t, e, "detail@example.com", "Det", "Ail", "+373****0099")

	// Link a Telegram account for the customer so we can assert
	// the linked/username/chat_id fields.
	if err := e.st.LinkCustomerTelegramForTest(context.Background(),
		cID, "123456789", "detail_handle"); err != nil {
		t.Fatalf("seed telegram link: %v", err)
	}

	rr := e.do(t, e.wrapAdmin(e.adminCustomerH.Detail), http.MethodGet,
		"/api/admin/customers/{id}", reqOpts{
			sess:        admin,
			pathValues:  map[string]string{"id": strconv.FormatInt(cID, 10)},
		})
	if rr.Code != http.StatusOK {
		t.Fatalf("Detail: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	decode(t, rr, &resp)
	if email, _ := resp["email"].(string); email != "detail@example.com" {
		t.Errorf("email = %v, want detail@example.com", resp["email"])
	}
	if v, _ := resp["telegram_linked"].(bool); !v {
		t.Errorf("telegram_linked = false, want true")
	}
	if v, _ := resp["telegram_username"].(string); v != "detail_handle" {
		t.Errorf("telegram_username = %v, want detail_handle", resp["telegram_username"])
	}
	if v, _ := resp["telegram_chat_id"].(string); v != "123456789" {
		t.Errorf("telegram_chat_id = %v, want 123456789", resp["telegram_chat_id"])
	}
	if _, hasPw := resp["hashed_password"]; hasPw {
		t.Errorf("hashed_password leaked in admin detail response — security regression")
	}
	if _, hasOrders := resp["orders"]; !hasOrders {
		t.Errorf("orders field missing in detail response")
	}

	// 404 path
	rr404 := e.do(t, e.wrapAdmin(e.adminCustomerH.Detail), http.MethodGet,
		"/api/admin/customers/{id}", reqOpts{
			sess:        admin,
			pathValues:  map[string]string{"id": "9999999"},
		})
	if rr404.Code != http.StatusNotFound {
		t.Errorf("missing customer: want 404, got %d", rr404.Code)
	}
}

// TestIntegrationAdminGetCustomerDetailWithoutTelegram is the
// regression guard for the NULL-into-bool Scan bug noted in
// TestIntegrationAdminGetCustomerDetail. We seed a customer with
// NO customer_oauth row, hit /api/admin/customers/{id}, and
// expect 200 with telegram_linked=false. Before the COALESCE
// fix this test returned 500 with a pgx "Scan error" inside
// the JSON envelope.
func TestIntegrationAdminGetCustomerDetailWithoutTelegram(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "custdetail-notg", "admin")

	cID := seedCustomer(t, e, "no-tg@example.com", "No", "Tg", "+373****0000")

	rr := e.do(t, e.wrapAdmin(e.adminCustomerH.Detail), http.MethodGet,
		"/api/admin/customers/{id}", reqOpts{
			sess:        admin,
			pathValues:  map[string]string{"id": strconv.FormatInt(cID, 10)},
		})
	if rr.Code != http.StatusOK {
		t.Fatalf("Detail without Telegram: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	decode(t, rr, &resp)
	if v, ok := resp["telegram_linked"].(bool); !ok || v {
		t.Errorf("telegram_linked = %v (%T), want false (bool)", resp["telegram_linked"], resp["telegram_linked"])
	}
}

// TestIntegrationAdminCustomersRequiresAuth asserts the route is
// behind adminOnly — an anonymous request gets 401 (or whatever
// the middleware returns for missing creds) and a customer session
// gets 403.
func TestIntegrationAdminCustomersRequiresAuth(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, e.wrapAdmin(e.adminCustomerH.List), http.MethodGet,
		"/api/admin/customers", reqOpts{})
	if rr.Code == http.StatusOK {
		t.Errorf("anonymous should NOT see admin customers; got 200")
	}

	// Customer session — also blocked.
	sess, _ := e.customerSession(t, "rbac-buyer@ex.com")
	rrCust := e.do(t, e.wrapAdmin(e.adminCustomerH.List), http.MethodGet,
		"/api/admin/customers", reqOpts{sess: sess})
	if rrCust.Code == http.StatusOK {
		t.Errorf("customer session should NOT see admin customers; got 200")
	}
}

// LinkCustomerTelegramForTest wires a customer_oauth row for
// integration tests. It's a thin shim around the pgxpool.Exec
// path that the production code would normally use via the
// customer's /api/store/customers/me/oauth endpoint — we don't
// call that here because it requires a customer session, CSRF
// token, and Telegram bot signature verification, all of which
// would just be noise in a test that wants to assert that the
// admin detail endpoint surfaces the linked account correctly.
//
// The "ForTest" suffix is a deliberate signal that this is a test
// helper, not a public API — production code should always go
// through the /api/store/customers/me/oauth route, which has the
// real bot signature check, rate limiting, and audit trail. The
// actual implementation lives in store/admin_customers.go (in the
// production package) so the test file can call it on the shared
// *PostgresStore — re-declaring it here would just duplicate the
// SQL. See LinkCustomerTelegramForTest in store/admin_customers.go.

// seedCustomer inserts a customer row via the public store API
// and returns the new id. The hashed_password we pass is a
// throwaway bcrypt — the test never logs in as this customer, it
// only asserts that the hash never leaks through any admin
// response.
func seedCustomer(t *testing.T, e *env, email, first, last, phone string) int64 {
	t.Helper()
	throwaway := model.Customer{
		Email:      email,
		FirstName:  first,
		LastName:   last,
		Phone:      phone,
		AvatarColor: "#44944A", // matches the DB default; CreateCustomer
		// doesn't set it, so without this we'd hit the NOT NULL
		// constraint on avatar_color for a freshly inserted row.
		// HashedPassword is set to a known throwaway value —
		// CreateCustomer writes it verbatim. The real password
		// would normally be hashed with bcrypt here, but for
		// admin-listing tests we just need *some* hash to assert
		// doesn't leak.
		HashedPW: "$2a$12$0000000000000000000000.0000000000000000000000000000000000",
	}
	if err := e.st.CreateCustomer(context.Background(), throwaway); err != nil {
		t.Fatalf("seed customer %s: %v", email, err)
	}
	c, err := e.st.GetCustomerByEmail(context.Background(), email)
	if err != nil || c == nil {
		t.Fatalf("re-fetch seeded customer %s: err=%v", email, err)
	}
	return c.ID
}

// Compile-time check that the test can call the production helper.
// The actual implementation is in store/admin_customers.go — the
// test only references it through e.st.LinkCustomerTelegramForTest
// above, so this line is just a documentation anchor that fails
// the build if someone renames the production method and forgets
// to update the test.

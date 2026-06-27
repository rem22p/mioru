package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"mioru/internal/cookieauth"
	"mioru/internal/model"
)

// seedProduct inserts an active, in-stock product with sizes directly via the store, returning its id.
func seedProduct(t *testing.T, e *env, slug string, priceMDL, stock int) int64 {
	t.Helper()
	id, err := e.st.CreateProduct(context.Background(), model.Product{
		Slug: slug, CategoryID: 1, Brand: "TestBrand", Name: "Test " + slug,
		Price: priceMDL, Color: "red", Status: "in_stock", InStock: true,
		StockQty: stock, CreatedBy: "test", Sizes: []model.ProductSize{model.ProductSize{Label: "M"}, model.ProductSize{Label: "L"}},
	})
	if err != nil {
		t.Fatalf("seedProduct %s: %v", slug, err)
	}
	return id
}

func TestIntegrationListProducts(t *testing.T) {
	e := newEnv(t)
	seedProduct(t, e, "p-1", 500, 10)
	seedProduct(t, e, "p-2", 300, 10)

	rr := e.do(t, http.HandlerFunc(e.storeH.ListProducts), http.MethodGet, "/api/products?per_page=20", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListProducts: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	// ListProducts wraps results in "products" (not "items") per store.go:78.
	var resp struct {
		Products []model.Product `json:"products"`
		Total    int             `json:"total"`
		Page     int             `json:"page"`
		PerPage  int             `json:"per_page"`
	}
	decode(t, rr, &resp)
	if resp.Total != 2 || len(resp.Products) != 2 {
		t.Errorf("want total=2 products=2, got total=%d products=%d", resp.Total, len(resp.Products))
	}
	slugs := map[string]bool{}
	for _, p := range resp.Products {
		slugs[p.Slug] = true
	}
	if !slugs["p-1"] || !slugs["p-2"] {
		t.Errorf("want slugs p-1 and p-2, got %+v", resp.Products)
	}
}

func TestIntegrationGetProductBySlug(t *testing.T) {
	const slug = "shirt-1"
	e := newEnv(t)
	seedProduct(t, e, slug, 500, 10)

	rr := e.do(t, http.HandlerFunc(e.storeH.GetProduct), http.MethodGet, "/api/products/"+slug,
		reqOpts{pathValues: map[string]string{"slug": slug}})
	if rr.Code != http.StatusOK {
		t.Fatalf("GetProduct: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var p model.Product
	decode(t, rr, &p)
	if p.Slug != slug {
		t.Errorf("slug = %q, want %s", p.Slug, slug)
	}
}

func TestIntegrationGetProductNotFound(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.storeH.GetProduct), http.MethodGet, "/api/products/does-not-exist",
		reqOpts{pathValues: map[string]string{"slug": "does-not-exist"}})
	if rr.Code != http.StatusNotFound {
		t.Errorf("missing product: want 404, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationListCategories(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.storeH.ListCategories), http.MethodGet, "/api/categories", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListCategories: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var cats []model.Category
	decode(t, rr, &cats)
	if len(cats) == 0 {
		t.Error("expected seeded category tree, got 0 categories")
	}
}

func TestIntegrationCustomerMeRequiresAuth(t *testing.T) {
	e := newEnv(t)
	// No session cookie → CustomerAuthMW must reject.
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated me: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationCustomerAuthGateEnvelope pins the CORRECT contract for the
// storefront auth gate: a JSON envelope with a machine code, per CLAUDE.md
// ("never http.Error … breaks the SPA's code-based branching"). The admin path
// already complies via jsonerr.ErrorCode; CustomerAuthMW mirrors that (#31/#34).
func TestIntegrationCustomerAuthGateEnvelope(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{})
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_REQUIRED" {
		t.Errorf("code = %q, want AUTH_REQUIRED", env.Code)
	}
}

// TestIntegrationCustomerAuthGateBadTokenEnvelope pins the same envelope for
// the "bad/expired token" branch: AUTH_INVALID + application/json, not the
// historic text/plain http.Error body.
func TestIntegrationCustomerAuthGateBadTokenEnvelope(t *testing.T) {
	e := newEnv(t)
	// Forge a cookie with a clearly invalid token.
	bad := &session{
		authCookie: &http.Cookie{Name: cookieauth.StoreAuthCookie, Value: "garbage-not-a-jwt"},
		csrfValue:  "csrf",
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.Me), http.MethodGet, "/api/store/customers/me", reqOpts{sess: bad})
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "AUTH_INVALID" {
		t.Errorf("code = %q, want AUTH_INVALID", env.Code)
	}
}

func TestIntegrationCartRoundTrip(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "cart@ex.com")
	// Seed a real product so the customer_cart FK on products(id) is satisfied.
	pid := seedProduct(t, e, "cart-product", 500, 10)

	saveBody := map[string]any{"items": []map[string]any{{"product_id": pid, "size_label": "M", "quantity": 2}}}
	rr := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut, "/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: saveBody})
	if rr.Code != http.StatusOK {
		t.Fatalf("SaveCart: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr2 := e.do(t, e.wrapCustomer(e.customerH.GetCart), http.MethodGet, "/api/store/customers/me/cart", reqOpts{sess: sess})
	if rr2.Code != http.StatusOK {
		t.Fatalf("GetCart: want 200, got %d (%s)", rr2.Code, rr2.Body.String())
	}
	var cart struct {
		Items []struct {
			ProductID int    `json:"product_id"`
			SizeLabel string `json:"size_label"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	decode(t, rr2, &cart)
	if len(cart.Items) != 1 || cart.Items[0].ProductID != int(pid) || cart.Items[0].Quantity != 2 {
		t.Errorf("cart round-trip: want 1 item product_id=%d qty=2, got %+v (body %s)", pid, cart.Items, rr2.Body.String())
	}
}

func TestIntegrationSaveCartCSRFGate(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "csrf@ex.com")
	// Mutation with a wrong CSRF token → 403.
	rr := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut, "/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", badCSRF: true, body: map[string]any{"items": []any{}}})
	if rr.Code != http.StatusForbidden {
		t.Errorf("bad CSRF: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationSaveCartRejectsUnknownProduct guards a regression that
// turned a "save cart with a stale product_id" scenario into a 500 ISE.
// Before the defence-in-depth pre-check in store.SaveCustomerCart, a
// missing product_id would surface as a generic SQLSTATE 23503 FK
// violation, which the handler treated as a catch-all error → 500.
// With the sentinel branch in customer.go::SaveCart the same input
// now returns 400 NOT_FOUND, signalling to the frontend that
// the cart is stale and the user has to refresh.
func TestIntegrationSaveCartRejectsUnknownProduct(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "stale-cart@ex.com")

	body := map[string]any{
		"items": []map[string]any{
			{"product_id": 999999, "size_label": "M", "quantity": 1},
		},
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut,
		"/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: body})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("SaveCart with unknown product: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	decode(t, rr, &resp)
	if resp.Code != "NOT_FOUND" {
		t.Errorf("response code = %q, want NOT_FOUND (full: %s)", resp.Code, rr.Body.String())
	}

	// A second PUT with a MIX of valid + invalid product ids should
	// also fail-fast on the first invalid one — we don't want to
	// silently drop the unknown line and keep the rest.
	pid := seedProduct(t, e, "stale-cart-mix", 500, 5)
	body2 := map[string]any{
		"items": []map[string]any{
			{"product_id": pid, "size_label": "M", "quantity": 1},
			{"product_id": 999998, "size_label": "L", "quantity": 1},
		},
	}
	rr2 := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut,
		"/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: body2})
	if rr2.Code != http.StatusBadRequest {
		t.Fatalf("SaveCart with mix valid+invalid: want 400, got %d (%s)", rr2.Code, rr2.Body.String())
	}

	// Sanity: a fully-valid cart still saves OK on the same session.
	body3 := map[string]any{
		"items": []map[string]any{
			{"product_id": pid, "size_label": "M", "quantity": 1},
		},
	}
	rr3 := e.do(t, e.wrapCustomer(e.customerH.SaveCart), http.MethodPut,
		"/api/store/customers/me/cart",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: body3})
	if rr3.Code != http.StatusOK {
		t.Fatalf("SaveCart fully valid: want 200, got %d (%s)", rr3.Code, rr3.Body.String())
	}
}

// TestIntegrationCreateOrderRejectsUnknownProduct is the CreateOrder
// counterpart: even with a clean cart at save time, a product can
// vanish between save and checkout (admin deletes, migration, etc).
// The order must come back 400 NOT_FOUND, not 500 ISE.
func TestIntegrationCreateOrderRejectsUnknownProduct(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "stale-order@ex.com")
	pid := seedProduct(t, e, "stale-order-p", 500, 5)

	body := map[string]any{
		"type":            "cart",
		"phone":           "+373777908542",
		"city":            "Tiraspol",
		"delivery_method": "personal",
		"payment_method":  "cash",
		"items": []map[string]any{
			{"product_id": pid, "size_label": "M", "quantity": 1},
			{"product_id": 999999, "size_label": "L", "quantity": 1},
		},
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost,
		"/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf",
			idempotencyKey: "key-stale-order-1", body: body})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("CreateOrder with unknown product: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	decode(t, rr, &resp)
	if resp.Code != "NOT_FOUND" {
		t.Errorf("response code = %q, want NOT_FOUND (full: %s)", resp.Code, rr.Body.String())
	}
}

// TestIntegrationCreateOrderRejectsCodWithBus is the server-side
// guard for the rule "cash on delivery is not available for bus
// (маршрутка) delivery". The store frontend already disables the
// radio in CheckoutPage::isPaymentBlocked, but a custom curl or a
// stale frontend bundle could still POST this combination. The
// handler must reject it with 400 VALIDATION_FAILED and never
// create the order — the bus courier has no way to take the money,
// so allowing the combination would create a payment that physically
// can't happen.
func TestIntegrationCreateOrderRejectsCodWithBus(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "cod-bus@ex.com")
	pid := seedProduct(t, e, "cod-bus-p", 500, 5)

	body := map[string]any{
		"type":            "cart",
		"phone":           "+373777908542",
		"city":            "Кишинёв", // one of the cities that allows "bus"
		"delivery_method": "bus",
		"payment_method":  "cod", // forbidden combo
		"items":           []map[string]any{{"product_id": pid, "size_label": "M", "quantity": 1}},
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-cod-bus-1", body: body})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("cod + bus: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	decode(t, rr, &resp)
	if resp.Code != "VALIDATION_FAILED" {
		t.Errorf("response code = %q, want VALIDATION_FAILED (full: %s)", resp.Code, rr.Body.String())
	}
	if !strings.Contains(resp.Error, "cash on delivery") && !strings.Contains(resp.Error, "bus") {
		t.Errorf("error message should mention the rule (got: %q)", resp.Error)
	}

	// Sanity: a valid combo (cod + address) still passes.
	body["delivery_method"] = "address"
	body["street"] = "Ленина 1"
	body["house"] = "5"
	rr = e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-cod-bus-2", body: body})
	if rr.Code != http.StatusCreated {
		t.Fatalf("cod + address: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}

	// And card + bus (the other direction) also passes — there is
	// no constraint against card payment for bus.
	body["payment_method"] = "card"
	body["city"] = "Бельцы" // bus-available PNR city
	body["street"] = ""
	body["house"] = ""
	rr = e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-cod-bus-3", body: body})
	if rr.Code != http.StatusCreated {
		t.Fatalf("card + bus: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationCreateOrderAllowsCodWithBusForIndividual locks in
// the exception to the cod+bus rule for type=individual orders:
// when the customer submits an individual order, they ride the
// bus themselves and pay the driver, so this combination is valid
// even though it is rejected for cart orders. Without this test
// a future tightening of the rule (e.g. "always block cod+bus")
// would silently break a legitimate customer flow.
func TestIntegrationCreateOrderAllowsCodWithBusForIndividual(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "cod-bus-individual@ex.com")

	body := map[string]any{
		"type":            "individual",
		"phone":           "+373777908542",
		"city":            "Тирасполь",
		"delivery_method": "bus",
		"payment_method":  "cod",
		"comment":         "привезу на маршрутке",
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-cod-bus-individual-1", body: body})
	if rr.Code != http.StatusCreated {
		t.Fatalf("individual + bus + cod: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationCreateOrderRejectsMoldovaPostInPNR is the
// server-side guard for the rule "Moldova Post is not available
// for Transnistria (PMR) cities". The store frontend already
// disables the radio in lib/deliveryRules.ts::isDeliveryBlocked,
// but a custom curl or a stale frontend bundle could still POST
// this combination. The handler must reject it with 400
// VALIDATION_FAILED and never create the order — the courier
// path doesn't exist for PMR.
func TestIntegrationCreateOrderRejectsMoldovaPostInPNR(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "moldovapost-pnr@ex.com")
	pid := seedProduct(t, e, "moldovapost-pnr-p", 500, 5)

	for _, pnrCity := range []string{"Тирасполь", "Бендеры", "Рыбница", "дубоссары"} {
		body := map[string]any{
			"type":            "cart",
			"phone":           "+373777908542",
			"city":            pnrCity,
			"delivery_method": "moldovaPost",
			"payment_method":  "card",
			"items":           []map[string]any{{"product_id": pid, "size_label": "M", "quantity": 1}},
		}
		rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-mp-pnr-" + pnrCity, body: body})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("moldovaPost + %s: want 400, got %d (%s)", pnrCity, rr.Code, rr.Body.String())
		}
		var resp struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		decode(t, rr, &resp)
		if resp.Code != "VALIDATION_FAILED" {
			t.Errorf("moldovaPost + %s: code = %q, want VALIDATION_FAILED", pnrCity, resp.Code)
		}
	}

	// Sanity: a non-PNR city still passes (e.g. Chisinau).
	body := map[string]any{
		"type":            "cart",
		"phone":           "+373777908542",
		"city":            "Кишинёв",
		"delivery_method": "moldovaPost",
		"payment_method":  "card",
		"items":           []map[string]any{{"product_id": pid, "size_label": "M", "quantity": 1}},
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-mp-chisinau-1", body: body})
	if rr.Code != http.StatusCreated {
		t.Fatalf("moldovaPost + Chisinau: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
}

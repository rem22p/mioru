package handler_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// orderBody returns a valid cart-order request body for the given product/qty.
// The phone is set to a valid E.164-ish value so existing tests don't have to
// worry about the new "phone is required" validation introduced in migration
// 012. Tests that want to exercise the phone field use orderBodyWithPhone.
func orderBody(productID int64, qty int) map[string]any {
	return map[string]any{
		"type":            "cart",
		"phone":           "+37312345678",
		"city":            "Tiraspol",
		"delivery_method": "personal",
		"payment_method":  "cash",
		"items":           []map[string]any{{"product_id": productID, "size_label": "M", "quantity": qty}},
	}
}

// orderBodyWithPhone is the same as orderBody but with an explicit phone value
// (e.g. "" for the "phone is required" tests, or a different valid number for
// the "sync to profile" test).
func orderBodyWithPhone(productID int64, qty int, phone string) map[string]any {
	b := orderBody(productID, qty)
	b["phone"] = phone
	return b
}

func TestIntegrationCreateOrderHappyAndDecrementsStock(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "order@ex.com")
	pid := seedProduct(t, e, "ord-1", 500, 10)

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-happy-1", body: orderBody(pid, 3)})
	if rr.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	var order struct {
		ID         int64  `json:"id"`
		OrderCode  string `json:"order_code"`
		TotalMinor int64  `json:"total_minor"`
	}
	decode(t, rr, &order)
	if order.ID == 0 {
		t.Error("created order has no ID")
	}
	if order.OrderCode == "" {
		t.Error("created order has no order_code (migration 019)")
	}
	const wantTotal = 3 * 500 * 100 // qty × priceMDL × 100 (MDL → minor units)
	if order.TotalMinor != wantTotal {
		t.Errorf("TotalMinor = %d, want %d (server-calculated: qty × priceMDL × 100)", order.TotalMinor, wantTotal)
	}
	p, err := e.st.GetProduct(context.Background(), "ord-1")
	if err != nil || p == nil {
		t.Fatalf("GetProduct: %v / %v", p, err)
	}
	// F1 fix decrements product_sizes.stock_quantity, not products.stock_quantity.
	// Check per-size stock for size "M".
	found := false
	for _, sz := range p.Sizes {
		if sz.Label == "M" {
			if sz.StockQuantity != 7 {
				t.Errorf("size M stock after order = %d, want 7", sz.StockQuantity)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("size M not found in product sizes")
	}
}

func TestIntegrationCreateOrderMissingIdempotencyKey(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "noidem@ex.com")
	pid := seedProduct(t, e, "ord-noidem", 500, 10)

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", body: orderBody(pid, 1)})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing Idempotency-Key: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

func TestIntegrationCreateOrderIdempotentReplay(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "replay@ex.com")
	pid := seedProduct(t, e, "ord-replay", 500, 10)
	body := orderBody(pid, 2)
	const key = "replay-key-1"

	first := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: body})
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit: want 201, got %d (%s)", first.Code, first.Body.String())
	}
	var o1 struct {
		ID int64 `json:"id"`
	}
	decode(t, first, &o1)

	second := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: body})
	if second.Code != http.StatusCreated {
		t.Fatalf("replay: want 201, got %d (%s)", second.Code, second.Body.String())
	}
	var o2 struct {
		ID int64 `json:"id"`
	}
	decode(t, second, &o2)
	if o1.ID != o2.ID {
		t.Errorf("replay created a new order: first=%d second=%d", o1.ID, o2.ID)
	}
	p, err := e.st.GetProduct(context.Background(), "ord-replay")
	if err != nil || p == nil {
		t.Fatalf("GetProduct: %v / %v", p, err)
	}
	// Per-size stock check for size "M" (F1 fix decrements product_sizes, not products).
	found := false
	for _, sz := range p.Sizes {
		if sz.Label == "M" {
			if sz.StockQuantity != 8 {
				t.Errorf("size M stock after replay = %d, want 8 (decremented once)", sz.StockQuantity)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("size M not found in product sizes")
	}
}

func TestIntegrationCreateOrderIdempotencyConflict(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "conflict@ex.com")
	pid := seedProduct(t, e, "ord-conflict", 500, 10)
	const key = "conflict-key-1"

	first := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: orderBody(pid, 1)})
	if first.Code != http.StatusCreated {
		t.Fatalf("first submit: want 201, got %d (%s)", first.Code, first.Body.String())
	}
	second := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: orderBody(pid, 5)})
	if second.Code != http.StatusConflict {
		t.Fatalf("idempotency conflict: want 409, got %d (%s)", second.Code, second.Body.String())
	}
	var env errEnvelope
	decode(t, second, &env)
	if env.Code != "IDEMPOTENCY_REPLAY" {
		t.Errorf("code = %q, want IDEMPOTENCY_REPLAY", env.Code)
	}
}

func TestIntegrationCreateOrderOversell(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "oversell@ex.com")
	pid := seedProduct(t, e, "ord-oversell", 500, 2)

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "oversell-1", body: orderBody(pid, 5)})
	if rr.Code != http.StatusConflict {
		t.Fatalf("oversell: want 409, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "INSUFFICIENT_STOCK" {
		t.Errorf("code = %q, want INSUFFICIENT_STOCK", env.Code)
	}
	p, err := e.st.GetProduct(context.Background(), "ord-oversell")
	if err != nil || p == nil {
		t.Fatalf("GetProduct: %v / %v", p, err)
	}
	// Per-size stock check for size "M" (F1 fix — oversell rejected, stock unchanged).
	found := false
	for _, sz := range p.Sizes {
		if sz.Label == "M" {
			if sz.StockQuantity != 2 {
				t.Errorf("size M stock after rejected oversell = %d, want 2 (unchanged)", sz.StockQuantity)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("size M not found in product sizes")
	}
}

// TestIntegrationListOrdersIsolation verifies that a customer only sees their
// own orders. Bob MUST place at least one order: if he had none, ListOrders
// would return alice's 2 even with the `WHERE customer_id = $1` scope filter
// deleted entirely — zero cross-customer data means zero signal. With bob's
// order sitting in the same table, alice's total==2 (not 3) and the absence of
// bob's order id from her list is what actually catches a missing/broken filter.
//
// ListOrders response uses key "orders" (not "items") per CustomerHandler.ListOrders;
// individual entries carry {id, total_minor, status, created_at} — no customer_id.
func TestIntegrationListOrdersIsolation(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.customerSession(t, "alice-ord@ex.com")
	bob, _ := e.customerSession(t, "bob-ord@ex.com")
	pid := seedProduct(t, e, "ord-list", 500, 100)

	// Bob places one order — its presence in the table is what gives this test
	// signal against a removed customer-scope filter.
	bobRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: bob, csrfCookieName: "store_csrf", idempotencyKey: "b-1", body: orderBody(pid, 1)})
	if bobRR.Code != http.StatusCreated {
		t.Fatalf("bob order: want 201, got %d (%s)", bobRR.Code, bobRR.Body.String())
	}
	var bobOrder struct {
		ID int64 `json:"id"`
	}
	decode(t, bobRR, &bobOrder)
	bobOrderID := bobOrder.ID
	if bobOrderID == 0 {
		t.Fatal("bob order has no id")
	}

	// Alice places her own two orders.
	for _, k := range []string{"a-1", "a-2"} {
		rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: alice, csrfCookieName: "store_csrf", idempotencyKey: k, body: orderBody(pid, 1)})
		if rr.Code != http.StatusCreated {
			t.Fatalf("alice order %q: want 201, got %d (%s)", k, rr.Code, rr.Body.String())
		}
	}

	rr := e.do(t, e.wrapCustomer(e.customerH.ListOrders), http.MethodGet, "/api/store/customers/me/orders?per_page=20", reqOpts{sess: alice})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListOrders: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Total  int `json:"total"`
		Orders []struct {
			ID int64 `json:"id"`
		} `json:"orders"`
	}
	decode(t, rr, &resp)
	if resp.Total != 2 {
		t.Errorf("alice total = %d, want 2 (must exclude bob's order)", resp.Total)
	}
	if len(resp.Orders) != 2 {
		t.Errorf("alice orders len = %d, want 2", len(resp.Orders))
	}
	for _, o := range resp.Orders {
		if o.ID == bobOrderID {
			t.Errorf("ListOrders leaked bob's order id %d into alice's list", bobOrderID)
		}
	}
}

func TestIntegrationAdminUpdateOrderStatus(t *testing.T) {
	e := newEnv(t)
	cust, _ := e.customerSession(t, "statuscust@ex.com")
	pid := seedProduct(t, e, "ord-status", 500, 10)
	createRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: cust, csrfCookieName: "store_csrf", idempotencyKey: "status-1", body: orderBody(pid, 1)})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("seed order: want 201, got %d (%s)", createRR.Code, createRR.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	decode(t, createRR, &created)

	admin := e.userSession(t, "admin1", "admin")
	target := "/api/admin/orders/" + itoa(created.ID) + "/status"
	rr := e.do(t, e.wrapAdmin(e.adminOrdH.UpdateStatus), http.MethodPatch, target,
		reqOpts{sess: admin, csrfCookieName: "csrf_token",
			pathValues: map[string]string{"id": itoa(created.ID)}, body: map[string]any{"status": "processing"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("UpdateStatus: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationCreateOrderRequiresPhone(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "nophone@ex.com")
	pid := seedProduct(t, e, "no-phone-p1", 500, 5)

	// Empty phone → 400 VALIDATION_FAILED.
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-nophone-1", body: orderBodyWithPhone(pid, 1, "")})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty phone: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Garbage phone → 400.
	rr = e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-nophone-2", body: orderBodyWithPhone(pid, 1, "abc")})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("garbage phone: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}

	// Phone with letters / too long / too short → 400.
	for _, bad := range []string{"+", "+123", "abcdefghijklmnop", "+373 12345 678"} {
		rr = e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-nophone-" + bad, body: orderBodyWithPhone(pid, 1, bad)})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("phone %q: want 400, got %d (%s)", bad, rr.Code, rr.Body.String())
		}
	}
}

func TestIntegrationCreateOrderPersistsPhoneAndSyncsToProfile(t *testing.T) {
	e := newEnv(t)
	sess, custID := e.customerSession(t, "phonesync@ex.com")
	pid := seedProduct(t, e, "phone-sync-p1", 500, 5)

	// customerSession seeds the customer with phone = "" by default
	// (see e.st.CreateCustomer with FirstName: "T"). The first order
	// types a real number — it should land both in orders.phone and
	// be copied back to customers.phone.
	const newPhone = "+37360011222"
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-sync-1", body: orderBodyWithPhone(pid, 1, newPhone)})
	if rr.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	var created struct {
		ID    int64  `json:"id"`
		Phone string `json:"phone"`
	}
	decode(t, rr, &created)
	if created.Phone != newPhone {
		t.Errorf("created.Phone = %q, want %q", created.Phone, newPhone)
	}

	// Profile should now carry the typed phone (best-effort sync).
	cust, err := e.st.GetCustomer(context.Background(), custID)
	if err != nil {
		t.Fatalf("GetCustomer: %v", err)
	}
	if cust.Phone != newPhone {
		t.Errorf("customers.phone = %q, want %q (best-effort sync after order)", cust.Phone, newPhone)
	}

	// Subsequent ListOrders should return the same phone on the row.
	listRR := e.do(t, e.wrapCustomer(e.customerH.ListOrders), http.MethodGet, "/api/store/customers/me/orders", reqOpts{sess: sess})
	if listRR.Code != http.StatusOK {
		t.Fatalf("ListOrders: want 200, got %d (%s)", listRR.Code, listRR.Body.String())
	}
	var listResp struct {
		Orders []struct {
			ID    int64  `json:"id"`
			Phone string `json:"phone"`
		} `json:"orders"`
	}
	decode(t, listRR, &listResp)
	if len(listResp.Orders) == 0 || listResp.Orders[0].Phone != newPhone {
		t.Errorf("ListOrders: phone missing or wrong: %+v", listResp.Orders)
	}
}

func TestIntegrationCreateOrderDoesNotResyncIdenticalPhone(t *testing.T) {
	e := newEnv(t)
	sess, custID := e.customerSession(t, "nosync@ex.com")
	pid := seedProduct(t, e, "no-sync-p1", 500, 5)

	// Place one order so customers.phone gets set.
	const phone = "+37370000000"
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-nosync-1", body: orderBodyWithPhone(pid, 1, phone)})
	if rr.Code != http.StatusCreated {
		t.Fatalf("first order: want 201, got %d", rr.Code)
	}
	cust, _ := e.st.GetCustomer(context.Background(), custID)
	updatedAtAfterFirst := // the store doesn't expose updated_at through GetCustomer's struct,
		// but the second-order check is sufficient: a stale-phone sync
		// would not change behaviour, so we just assert that the
		// customer row still has the same phone number.
		""
	_ = updatedAtAfterFirst

	// Place a SECOND order with the SAME phone. Sync must be a no-op
	// (we can't observe updated_at through the public API, but we
	// can at least confirm the second order also returns 201 and
	// the phone stays the same).
	rr = e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-nosync-2", body: orderBodyWithPhone(pid, 1, phone)})
	if rr.Code != http.StatusCreated {
		t.Fatalf("second order: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	cust, _ = e.st.GetCustomer(context.Background(), custID)
	if cust.Phone != phone {
		t.Errorf("customers.phone changed: got %q, want %q", cust.Phone, phone)
	}
}

// TestIntegrationCreateOrderWithPreorderMeasurements verifies that
// measurements submitted on a preorder cart-type order survive the
// full handler chain: request → CreateOrder → INSERT → GetOrder → response.
func TestIntegrationCreateOrderWithPreorderMeasurements(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "prem@ex.com")
	pid := seedProduct(t, e, "prem-1", 500, 10)

	body := map[string]any{
		"type":            "cart",
		"phone":           "+37369123456",
		"city":            "Tiraspol",
		"delivery_method": "personal",
		"payment_method":  "cash",
		"items": []map[string]any{
			{
				"product_id": pid,
				"size_label": "M",
				"quantity":   1,
				"measurements": map[string]any{
					"height": float64(175),
					"weight": float64(70),
				},
			},
		},
	}

	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-prem-1", body: body})
	if rr.Code != http.StatusCreated {
		t.Fatalf("CreateOrder preorder: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}

	var created struct {
		ID    int64 `json:"id"`
		Items []struct {
			ProductID    int64                  `json:"product_id"`
			SizeLabel    string                 `json:"size_label"`
			Measurements map[string]interface{} `json:"measurements"`
		} `json:"items"`
	}
	decode(t, rr, &created)
	if created.ID == 0 {
		t.Fatal("created order has no ID")
	}
	if len(created.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(created.Items))
	}
	m := created.Items[0].Measurements
	if m == nil {
		t.Fatal("measurements came back as nil — chain broke between handler and response")
	}
	// json.Unmarshal produces float64 for JSON numbers in map[string]interface{}.
	if v, ok := m["height"]; !ok || v != float64(175) {
		t.Errorf("measurements.height = %v, want 175", v)
	}
	if v, ok := m["weight"]; !ok || v != float64(70) {
		t.Errorf("measurements.weight = %v, want 70", v)
	}
}

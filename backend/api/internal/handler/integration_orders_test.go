package handler_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// orderBody returns a valid cart-order request body for the given product/qty.
func orderBody(productID int64, qty int) map[string]any {
	return map[string]any{
		"type":            "cart",
		"city":            "Tiraspol",
		"delivery_method": "personal",
		"payment_method":  "cash",
		"items":           []map[string]any{{"product_id": productID, "size_label": "M", "quantity": qty}},
	}
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
		ID         int64 `json:"id"`
		TotalMinor int64 `json:"total_minor"`
	}
	decode(t, rr, &order)
	if order.ID == 0 {
		t.Error("created order has no ID")
	}
	const wantTotal = 3 * 500 * 100 // qty × priceMDL × 100 (MDL → minor units)
	if order.TotalMinor != wantTotal {
		t.Errorf("TotalMinor = %d, want %d (server-calculated: qty × priceMDL × 100)", order.TotalMinor, wantTotal)
	}
	p, err := e.st.GetProduct(context.Background(), "ord-1")
	if err != nil || p == nil {
		t.Fatalf("GetProduct: %v / %v", p, err)
	}
	if p.StockQty != 7 {
		t.Errorf("stock after order = %d, want 7", p.StockQty)
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
	if p.StockQty != 8 {
		t.Errorf("stock after replay = %d, want 8 (decremented once)", p.StockQty)
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
	if p.StockQty != 2 {
		t.Errorf("stock after rejected oversell = %d, want 2 (unchanged)", p.StockQty)
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

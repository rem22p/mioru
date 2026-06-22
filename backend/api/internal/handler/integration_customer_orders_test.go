// Customer-facing ListOrders test coverage.
//
// The store-front ProfilePage renders order history from
// GET /api/store/customers/me/orders. Before the response was expanded
// (see handler/customer.go::ListOrders), the endpoint returned only
// 4 fields per order (id, total_minor, status, created_at) and a
// literal `profile.orderType.<undefined>` raw i18n key was leaking
// into the UI because `o.type` was undefined.
//
// This test pins the contract: every field the store-front expects
// (type, city, delivery_method, payment_method, address, items[] with
// product name/slug/image) is present on the response, and the
// admin-only `customer_id` / `customer_email` are scrubbed.
package handler_test

import (
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

func TestIntegrationCustomerListOrdersIncludesFullDetails(t *testing.T) {
	e := newEnv(t)
	alice, _ := e.customerSession(t, "listorders@ex.com")

	// Two products so items[] has something to render.
	p1 := seedProduct(t, e, "list-orders-p1", 12345, 5)
	p2 := seedProduct(t, e, "list-orders-p2", 67890, 5)

	// Place an order with TWO items so we exercise batched item load.
	body := orderBody(p1, 1)
	body["city"] = "Тирасполь"
	body["street"] = "ул. Ленина"
	body["house"] = "42"
	body["items"] = []map[string]any{
		{"product_id": p1, "size_label": "M", "quantity": 1, "price_minor": 12345},
		{"product_id": p2, "size_label": "L", "quantity": 2, "price_minor": 67890},
	}
	createRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{
			sess:           alice,
			csrfCookieName: cookieauth.StoreCSRFCookie,
			idempotencyKey: "key-listorders-1",
			body:           body,
		})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", createRR.Code, createRR.Body.String())
	}

	// List orders for that customer.
	listRR := e.do(t, e.wrapCustomer(e.customerH.ListOrders), http.MethodGet, "/api/store/customers/me/orders?per_page=20",
		reqOpts{sess: alice})
	if listRR.Code != http.StatusOK {
		t.Fatalf("ListOrders: want 200, got %d (%s)", listRR.Code, listRR.Body.String())
	}

	var resp struct {
		Orders []map[string]any `json:"orders"`
		Total  int              `json:"total"`
	}
	decode(t, listRR, &resp)
	if resp.Total != 1 || len(resp.Orders) != 1 {
		t.Fatalf("want 1 order, got total=%d orders=%d", resp.Total, len(resp.Orders))
	}
	o := resp.Orders[0]

	// Every field the ProfilePage reads must be present.
	// Reviewer finding (re-review PR #51): `phone` was missing from
	// mustHave even though order_postgres.go:65 SELECTs `o.phone`
	// and the OrderCard in ProfilePage renders a `tel:` chip from it.
	// A regression in the Scan/SELECT would pass silently while the
	// chip disappeared. Pin it.
	mustHave := []string{
		"id", "type", "status", "total_minor",
		"phone",
		"city", "delivery_method", "payment_method",
		"street", "house", "apartment", "comment",
		"items", "created_at",
	}
	for _, k := range mustHave {
		if _, ok := o[k]; !ok {
			t.Errorf("order response missing field %q (got keys: %v)", k, mapKeys(o))
		}
	}

	// Field values must match what we inserted.
	if got := o["type"]; got != "cart" {
		t.Errorf("type = %v, want cart", got)
	}
	if got := o["city"]; got != "Тирасполь" {
		t.Errorf("city = %v, want Тирасполь", got)
	}
	if got := o["delivery_method"]; got != "personal" {
		t.Errorf("delivery_method = %v, want personal", got)
	}
	if got := o["payment_method"]; got != "cash" {
		t.Errorf("payment_method = %v, want cash", got)
	}

	// Admin-only fields must be scrubbed.
	if v, ok := o["customer_id"]; ok && v != nil && v != float64(0) {
		t.Errorf("customer_id should be scrubbed (0 / absent), got %v", v)
	}
	if v, ok := o["customer_email"]; ok && v != nil && v != "" {
		t.Errorf("customer_email should be absent, got %v", v)
	}

	// items[] must be present, with 2 entries, each carrying the
	// product join columns. image_url is omitempty in the model and
	// is only populated when the product has at least one row in
	// product_images — seedProduct does not insert one, so the test
	// is permissive about it.
	rawItems, _ := o["items"].([]any)
	if len(rawItems) != 2 {
		t.Fatalf("want 2 items, got %d (raw: %v)", len(rawItems), o["items"])
	}
	for i, ri := range rawItems {
		item, _ := ri.(map[string]any)
		required := []string{
			"product_id", "product_name", "product_slug",
			"size_label", "quantity", "price_minor",
		}
		for _, k := range required {
			if _, ok := item[k]; !ok {
				t.Errorf("items[%d] missing %q (got %v)", i, k, mapKeys(item))
			}
		}
		if item["size_label"] == "" {
			t.Errorf("items[%d].size_label is empty", i)
		}
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

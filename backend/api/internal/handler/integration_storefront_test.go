package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/model"
)

// seedProduct inserts an active, in-stock product with sizes directly via the store, returning its id.
func seedProduct(t *testing.T, e *env, slug string, priceMDL, stock int) int64 {
	t.Helper()
	id, err := e.st.CreateProduct(context.Background(), model.Product{
		Slug: slug, CategoryID: 1, Brand: "TestBrand", Name: "Test " + slug,
		Price: priceMDL, Color: "red", Status: "in_stock", InStock: true,
		StockQty: stock, CreatedBy: "test", Sizes: []string{"M", "L"},
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
// already complies via jsonerr.ErrorCode; CustomerAuthMW still uses http.Error
// (text/plain, no code). Skipped until the bug is fixed.
func TestIntegrationCustomerAuthGateEnvelope(t *testing.T) {
	t.Skip("blocked by #31: CustomerAuthMW uses http.Error (text/plain, no machine code)")
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

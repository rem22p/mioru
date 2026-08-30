package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mioru/internal/model"
)

// TestIntegrationCreateOrderCategory pins KAN-52: individual orders must
// declare a category; shoes carry the insole length, other categories
// reject it, and the values round-trip through the order response.
func TestIntegrationCreateOrderCategory(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "category@ex.com")

	doOrder := func(key string, body map[string]any) *httptest.ResponseRecorder {
		return e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: key, body: body})
	}

	happy := map[string]any{
		"type":            "individual",
		"phone":           "+37377790854",
		"city":            "Тирасполь",
		"delivery_method": "personal",
		"payment_method":  "card",
		"category":        "shoes",
		"foot_length":     27,
	}
	rr := doOrder("key-category-shoes-1", happy)
	if rr.Code != http.StatusCreated {
		t.Fatalf("shoes order: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	// Round-trip: decode the order body and compare the fields, not
	// substrings — a substring assert would accept foot_length=27.5 too.
	var got struct {
		Category   string   `json:"category"`
		FootLength *float64 `json:"foot_length"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode order response: %v (%s)", err, rr.Body.String())
	}
	if got.Category != "shoes" {
		t.Errorf("response category = %q, want shoes", got.Category)
	}
	if got.FootLength == nil || *got.FootLength != 27 {
		t.Errorf("response foot_length = %v, want 27", got.FootLength)
	}

	base := map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card"}
	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing category", base},
		{"bad enum", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "hats"}},
		{"shoes without foot_length", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes"}},
		{"clothing with foot_length", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "clothing", "foot_length": 27}},
		{"shoes with height", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": 27, "height": 180}},
		{"accessories with foot_length", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "accessories", "foot_length": 27}},
		{"foot_length out of bounds low", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": 5}},
		{"foot_length just below 10", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": 9.99}},
		{"foot_length just above 40", map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": 40.01}},
	}
	for i, c := range cases {
		rr := doOrder(keySeq("key-category-reject", i), c.body)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d (%s)", c.name, rr.Code, rr.Body.String())
		}
	}

	// Boundary happy path: 10 and 40 are inclusive on both sides.
	for _, v := range []float64{10, 40} {
		body := map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": v}
		rr := doOrder(keySeq("key-category-boundary", int(v)), body)
		if rr.Code != http.StatusCreated {
			t.Errorf("foot_length=%v: want 201, got %d (%s)", v, rr.Code, rr.Body.String())
		}
	}
}

// TestIntegrationCreateOrderCategoryCartGate pins the F1 review fix: cart
// orders must never carry category/foot_length. A crafted cart order with
// category="hack" used to reach the INSERT and die on the CHECK constraint
// with a 500; a valid enum or in-range foot_length silently persisted.
func TestIntegrationCreateOrderCategoryCartGate(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "cart-category@ex.com")
	pid := seedProduct(t, e, "cart-category-product", 100, 5)

	cartBody := func(extra map[string]any) map[string]any {
		b := map[string]any{
			"type":            "cart",
			"phone":           "+37377790854",
			"city":            "Тирасполь",
			"delivery_method": "personal",
			"payment_method":  "card",
			"items":           []map[string]any{{"product_id": pid, "size_label": "M", "quantity": 1}},
		}
		for k, v := range extra {
			b[k] = v
		}
		return b
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{"cart with arbitrary category", cartBody(map[string]any{"category": "hack"})},
		{"cart with valid enum category", cartBody(map[string]any{"category": "shoes"})},
		{"cart with in-range foot_length", cartBody(map[string]any{"foot_length": 27})},
	}
	for i, c := range cases {
		rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: keySeq("key-cart-gate", i), body: c.body})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d (%s)", c.name, rr.Code, rr.Body.String())
		}
	}

	// Legal cart order without category stays accepted.
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-cart-legal-1", body: cartBody(nil)})
	if rr.Code != http.StatusCreated {
		t.Errorf("legal cart order: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func keySeq(prefix string, i int) string {
	return prefix + "-" + string(rune('a'+i))
}

// TestIntegrationCategoryProductCounts pins KAN-55: the public categories
// endpoint carries a products_count per category, and a parent's count
// includes its children's products (the count badge on the catalog chips).
func TestIntegrationCategoryProductCounts(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	// Seed one product into the clothing parent (id 1) and one into its
	// child tshirts-polo (id 2) — both through the public store API.
	if _, err := e.st.CreateProduct(ctx, model.Product{
		Slug: "cnt-parent", CategoryID: 1, Brands: []string{"CntBrand"},
		Name: "Parent P", Price: 100, Status: "in_stock", InStock: true,
	}); err != nil {
		t.Fatalf("insert parent product: %v", err)
	}
	if _, err := e.st.CreateProduct(ctx, model.Product{
		Slug: "cnt-child", CategoryID: 2, Brands: []string{"CntBrand"},
		Name: "Child P", Price: 100, Status: "in_stock", InStock: true,
	}); err != nil {
		t.Fatalf("insert child product: %v", err)
	}
	// Depth-3: 16 (Аксессуары) → 20 (Ювелирные) → 21 (Браслеты). The
	// parent badge must include the grandchild (the grid filters all
	// descendants when the parent is selected).
	if _, err := e.st.CreateProduct(ctx, model.Product{
		Slug: "cnt-grandchild", CategoryID: 21, Brands: []string{"CntBrand"},
		Name: "Bracelet P", Price: 100, Status: "in_stock", InStock: true,
	}); err != nil {
		t.Fatalf("insert grandchild product: %v", err)
	}

	rr := e.do(t, http.HandlerFunc(e.storeH.ListCategories), http.MethodGet, "/api/categories", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListCategories: want 200, got %d", rr.Code)
	}
	var cats []model.Category
	decode(t, rr, &cats)

	counts := map[int]int{}
	var collect func(cs []model.Category)
	collect = func(cs []model.Category) {
		for _, c := range cs {
			counts[c.ID] = c.ProductsCount
			collect(c.Children)
		}
	}
	collect(cats)
	// Harness truncates data tables before each test, so the counts are
	// exactly the products seeded above.
	if got := counts[1]; got != 2 {
		t.Errorf("clothing products_count = %d, want 2 (own + child)", got)
	}
	if got := counts[20]; got != 1 {
		t.Errorf("jewelry products_count = %d, want 1 (grandchild)", got)
	}
	if got := counts[16]; got != 1 {
		t.Errorf("accessories products_count = %d, want 1 (depth-3 descendant)", got)
	}
}

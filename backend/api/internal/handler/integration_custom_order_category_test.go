package handler_test

import (
	"net/http"
	"testing"
)

// TestIntegrationCreateOrderCategory pins KAN-52: individual orders must
// declare a category; shoes carry the insole length, other categories
// reject it, and the values round-trip through the order response.
func TestIntegrationCreateOrderCategory(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "category@ex.com")

	happy := map[string]any{
		"type":            "individual",
		"phone":           "+37377790854",
		"city":            "Тирасполь",
		"delivery_method": "personal",
		"payment_method":  "card",
		"category":        "shoes",
		"foot_length":     27,
	}
	rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: "key-category-shoes-1", body: happy})
	if rr.Code != http.StatusCreated {
		t.Fatalf("shoes order: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}
	// Round-trip: category + foot_length come back on the order body.
	if !containsJSON(rr.Body.String(), `"category":"shoes"`) {
		t.Errorf("response missing category=shoes: %s", rr.Body.String())
	}
	if !containsJSON(rr.Body.String(), `"foot_length":27`) {
		t.Errorf("response missing foot_length=27: %s", rr.Body.String())
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing category",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card"}},
		{"bad enum",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "hats"}},
		{"shoes without foot_length",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes"}},
		{"clothing with foot_length",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "clothing", "foot_length": 27}},
		{"shoes with height",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": 27, "height": 180}},
		{"accessories with foot_length",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "accessories", "foot_length": 27}},
		{"foot_length out of bounds",
			map[string]any{"type": "individual", "phone": "+37377790854", "city": "Тирасполь", "delivery_method": "personal", "payment_method": "card", "category": "shoes", "foot_length": 5}},
	}
	for i, c := range cases {
		rr := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
			reqOpts{sess: sess, csrfCookieName: "store_csrf", idempotencyKey: keySeq("key-category-reject", i), body: c.body})
		if rr.Code != http.StatusBadRequest {
			t.Errorf("%s: want 400, got %d (%s)", c.name, rr.Code, rr.Body.String())
		}
	}
}

func containsJSON(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func keySeq(prefix string, i int) string {
	return prefix + "-" + string(rune('a'+i))
}

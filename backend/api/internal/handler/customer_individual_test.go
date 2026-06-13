package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/middleware"
	"mioru/internal/model"
)

// oversellIndividualFakeStore simulates the precise vector found in the
// final security pass: type=individual with crafted items that the
// store would happily write through (stock_condition WHERE
// stock_quantity >= -100 always passes; total_minor = product_price *
// -100). If the handler does not pre-validate, CreateOrder gets called.
type oversellIndividualFakeStore struct {
	fakeCustomerStore
	createCalled bool
}

func (o *oversellIndividualFakeStore) CreateOrder(ctx context.Context, customerID int64, ord *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	o.createCalled = true
	// Return a valid order so the handler takes the success path —
	// otherwise the test wouldn't actually demonstrate the stock-drain
	// vector (it would just fail on the rejection path).
	return &model.Order{ID: 999, Type: ord.Type}, nil
}

// TestCreateOrderIndividualRejectsCraftedItems guards the wire-format
// contract for type=individual. Per CLAUDE.md priority #1 (orders,
// payments, stock), per-item bounds (ProductID > 0, Quantity 1-99,
// len 1-50) MUST be enforced regardless of order type. Previously the
// bounds lived inside `if req.Type == "cart"`, so a crafted
// type=individual request with quantity=-100 flowed through to the
// store, which ran `stock_quantity = stock_quantity - (-100)` and
// inflated stock. The test asserts:
//
//   - HTTP 400 with envelope code VALIDATION_FAILED;
//   - store.CreateOrder is NEVER called (failing it would mean the
//     stock-drain vector is still reachable);
//   - the message is informative for the storefront.
func TestCreateOrderIndividualRejectsCraftedItems(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "negative quantity drains stock (the original finding)",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"items": [{"product_id": 105, "size_label": "M", "quantity": -100}],
				"total_minor": 0
			}`,
		},
		{
			name: "zero quantity also out of range",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"items": [{"product_id": 105, "size_label": "M", "quantity": 0}],
				"total_minor": 0
			}`,
		},
		{
			name: "zero product_id is invalid",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"items": [{"product_id": 0, "size_label": "M", "quantity": 1}],
				"total_minor": 0
			}`,
		},
		{
			name: "size_label over 32 chars is rejected",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"items": [{"product_id": 105, "size_label": "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM", "quantity": 1}],
				"total_minor": 0
			}`,
		},
		{
			name: "delivery_time over 10 entries is rejected",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"delivery_time": ["a","b","c","d","e","f","g","h","i","j","k"]
			}`,
		},
		{
			name: "delivery_time element over 32 chars is rejected",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"delivery_time": ["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]
			}`,
		},
		{
			name: "height out of range is rejected",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"height": 1000
			}`,
		},
		{
			name: "weight out of range is rejected",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"weight": -5
			}`,
		},
		// Note: "empty items list" used to be invalid for any type, but
		// the regression fix split per-type presence: cart must have
		// at least one item (see TestCreateOrderCartStillRejectsEmptyItems),
		// individual may carry none (see TestCreateOrderIndividualAcceptsEmptyItems).
		{
			name: "more than 50 items is invalid",
			body: `{
				"type": "individual",
				"city": "Тирасполь",
				"delivery_method": "personal",
				"payment_method": "card",
				"items": [
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1},
{"product_id": 105, "size_label": "M", "quantity": 1}],
				"total_minor": 0
			}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := &oversellIndividualFakeStore{}
			h := newCustomerHandlerForTest(fs)

			req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", "test-individual-"+tc.name)
			req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

			rr := httptest.NewRecorder()
			h.CreateOrder(rr, req)

			if fs.createCalled {
				t.Fatalf("store.CreateOrder was called for crafted items — stock-drain vector is reachable (body: %s)", rr.Body.String())
			}
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", rr.Code, rr.Body.String())
			}
			var resp struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Code != "VALIDATION_FAILED" {
				t.Errorf("code = %q, want VALIDATION_FAILED", resp.Code)
			}
			if resp.Error == "" {
				t.Error("error message must not be empty for the storefront")
			}
		})
	}
}

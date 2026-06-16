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

// individualNoItemsFakeStore records the call and returns a successful
// order, simulating the live behaviour described in
// apps/store/src/routes/CustomOrderPage.tsx:117-130 — a legitimate
// individual order is submitted without an `items` field, the store
// layer treats it as a no-op (total_minor=0, stock untouched), and the
// order is created with no items.
type individualNoItemsFakeStore struct {
	fakeCustomerStore
	createCalled bool
	gotItems     int
}

func (i *individualNoItemsFakeStore) CreateOrder(ctx context.Context, customerID int64, ord *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	i.createCalled = true
	i.gotItems = len(items)
	return &model.Order{ID: 1001, Type: ord.Type}, nil
}

// TestCreateOrderIndividualAcceptsEmptyItems guards the regression
// introduced by the stock-drain fix. After per-item bounds were lifted
// out of the `if cart` block, the `len(req.Items) == 0` guard was also
// lifted, which made every individual order fail with 400. The
// store-layer contract (CreateOrder) explicitly accepts empty items for
// individual orders (no-op, total_minor=0, stock untouched), and the
// CustomOrderPage ships exactly that payload. The handler MUST NOT
// pre-reject this case. The test asserts:
//
//   - HTTP 201 with the order body from the store;
//   - store.CreateOrder IS called (handler reaches the success path);
//   - items slice is empty (per the individual contract).
func TestCreateOrderIndividualAcceptsEmptyItems(t *testing.T) {
	fs := &individualNoItemsFakeStore{}
	h := newCustomerHandlerForTest(fs)

	// Mirrors CustomOrderPage.tsx:117-130 exactly: type=individual,
	// no `items` field, free-form fields (height/weight/photos).
	body := `{
		"type": "individual",
		"phone": "+37360000000",
		"city": "Тирасполь",
		"delivery_method": "personal",
		"payment_method": "card",
		"height": 180,
		"weight": 75,
		"photos": ["/uploads/photo1.jpg"],
		"comment": "test"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-individual-no-items-001")
	req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if !fs.createCalled {
		t.Fatalf("store.CreateOrder was not called — individual-without-items regression (body: %s)", rr.Body.String())
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rr.Code, rr.Body.String())
	}
	if fs.gotItems != 0 {
		t.Errorf("individual order reached store with %d items, want 0", fs.gotItems)
	}
	var resp struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Type != "individual" {
		t.Errorf("type = %q, want individual", resp.Type)
	}
}

// TestCreateOrderCartStillRejectsEmptyItems makes sure the per-type
// split didn't accidentally let cart orders through with no items.
// The cart contract requires at least one item — without it, the
// store has nothing to decrement, and `total_minor` would always be 0.
func TestCreateOrderCartStillRejectsEmptyItems(t *testing.T) {
	fs := &fakeCustomerStore{}
	h := newCustomerHandlerForTest(fs)

	body := `{
		"type": "cart",
		"phone": "+37360000000",
		"city": "Тирасполь",
		"delivery_method": "personal",
		"payment_method": "card",
		"items": []
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-cart-empty-001")
	req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", resp.Code)
	}
}

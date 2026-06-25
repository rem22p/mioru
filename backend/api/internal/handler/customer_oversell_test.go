package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
)

// oversellFakeStore is a minimal customerStore that simulates an
// insufficient-stock error from CreateOrder. It also makes sure the
// handler never reaches any other branch (idempotency, generic internal
// error) when the sentinel is matched.
type oversellFakeStore struct {
	fakeCustomerStore
	createCalled bool
}

func (o *oversellFakeStore) CreateOrder(ctx context.Context, customerID int64, ord *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	o.createCalled = true
	return nil, fmt.Errorf("product 105: %w (requested 99)", store.ErrInsufficientStock)
}

// TestCreateOrderInsufficientStockReturns409 guards the wire-format
// contract for oversell errors. The handler MUST:
//
//   - match store.ErrInsufficientStock via errors.Is (not substring
//     matching — per CLAUDE.md "sentinel + errors.Is, never err.Error()
//     == ...");
//   - return HTTP 409 with envelope code INSUFFICIENT_STOCK;
//   - return a human-readable message in Russian for the storefront.
//
// The test sends a minimal valid-looking payload and asserts on the
// response envelope. The store fake is enough — the full CreateOrder
// validation/price-recalc flow is exercised in the store-layer tests.
func TestCreateOrderInsufficientStockReturns409(t *testing.T) {
	fs := &oversellFakeStore{}
	h := newCustomerHandlerForTest(fs)

	body := `{
		"type": "cart",
		"phone": "+37360000000",
		"city": "Тирасполь",
		"delivery_method": "personal",
		"payment_method": "card",
		"items": [{"product_id": 105, "size_label": "M", "quantity": 99}],
		"total_minor": 50000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-oversell-001")
	// CustomerAuthMW in main.go injects this; in tests we call the
	// handler directly and inject it ourselves.
	req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if !fs.createCalled {
		t.Fatal("CreateOrder was not called on the store")
	}
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
	var resp struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != "INSUFFICIENT_STOCK" {
		t.Errorf("code = %q, want INSUFFICIENT_STOCK", resp.Code)
	}
	if !strings.Contains(strings.ToLower(resp.Error), "наличии") {
		t.Errorf("error = %q, want to mention 'наличии' for the storefront", resp.Error)
	}
}

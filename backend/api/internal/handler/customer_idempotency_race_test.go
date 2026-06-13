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

// idempotencyRaceFakeStore simulates a race on the order_idempotency_pkey
// unique constraint: the concurrent winner inserted first, this loser's
// INSERT tripped the unique index, and the store converted the 23505
// into store.ErrIdempotencyReplay. Money and stock are safe — exactly
// one order exists. The handler must NOT leak 500 to the client.
type idempotencyRaceFakeStore struct {
	fakeCustomerStore
	createCalled bool
}

func (i *idempotencyRaceFakeStore) CreateOrder(ctx context.Context, customerID int64, ord *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	i.createCalled = true
	return nil, fmt.Errorf("idempotency insert: %w", store.ErrIdempotencyReplay)
}

// TestCreateOrderIdempotencyRaceReturns409 verifies that when a concurrent
// submit with the same Idempotency-Key wins the race, the loser's handler
// returns a structured 409 IDEMPOTENCY_REPLAY instead of leaking 500
// internal error. Per CLAUDE.md priority #1 (orders/payments/stock) and
// general hygiene: 500 is wrong; the client should know the order is
// already created and can refetch.
func TestCreateOrderIdempotencyRaceReturns409(t *testing.T) {
	fs := &idempotencyRaceFakeStore{}
	h := newCustomerHandlerForTest(fs)

	body := `{
		"type": "cart",
		"city": "Тирасполь",
		"delivery_method": "personal",
		"payment_method": "card",
		"items": [{"product_id": 105, "size_label": "M", "quantity": 1}],
		"total_minor": 50000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-race-001")
	req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if !fs.createCalled {
		t.Fatal("CreateOrder was not called on the store")
	}
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body: %s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != "IDEMPOTENCY_REPLAY" {
		t.Errorf("code = %q, want IDEMPOTENCY_REPLAY", resp.Code)
	}
	if resp.Error == "" {
		t.Error("error message must not be empty for the storefront")
	}
}

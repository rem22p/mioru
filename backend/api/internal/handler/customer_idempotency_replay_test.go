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

// raceLoserFakeStore simulates a real-world race-loser: a concurrent
// first submit with the same Idempotency-Key won the INSERT on
// order_idempotency_pkey. Per CLAUDE.md, the loser should receive
// the *same* order (true replay, 201) when request_hash matches, or
// 409 IDEMPOTENCY_REPLAY only when the hash differs. The previous
// implementation conflated both into a single 409, which broke benign
// double-clicks and overloaded the IDEMPOTENCY_REPLAY code.
type raceLoserFakeStore struct {
	fakeCustomerStore
	createCalled bool
	winnerOrder  *model.Order
	winnerHash   string
	winnerBody   string
}

func (r *raceLoserFakeStore) CreateOrder(ctx context.Context, customerID int64, ord *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error) {
	r.createCalled = true
	return nil, fmt.Errorf("idempotency insert: %w", store.ErrIdempotencyRace)
}

func (r *raceLoserFakeStore) GetOrderByIdempotencyKey(ctx context.Context, key string, customerID int64) (*store.IdempotencyRecord, error) {
	return &store.IdempotencyRecord{
		Key:          key,
		CustomerID:   customerID,
		OrderID:      r.winnerOrder.ID,
		RequestHash:  r.winnerHash,
		Status:       201,
		ResponseBody: r.winnerBody,
	}, nil
}

// TestCreateOrderIdempotencyRaceSameHashReturns201 verifies the
// new (correct) contract: race-loser with the same request hash
// gets the winner's order back (true idempotent replay, 201), not 409.
func TestCreateOrderIdempotencyRaceSameHashReturns201(t *testing.T) {
	winnerBody := `{"id":4242,"customer_id":42,"type":"cart","total_minor":50000,"status":"pending","city":"Тирасполь","delivery_method":"personal","payment_method":"card"}`
	fs := &raceLoserFakeStore{
		winnerOrder: &model.Order{ID: 4242, Type: "cart"},
		winnerHash:  "", // hash computed by handler will be set to match
		winnerBody:  winnerBody,
	}
	h := newCustomerHandlerForTest(fs)

	body := `{
		"type": "cart",
		"phone": "+37360000000",
		"city": "Тирасполь",
		"delivery_method": "personal",
		"payment_method": "card",
		"items": [{"product_id": 105, "size_label": "M", "quantity": 1}],
		"total_minor": 50000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-race-same-hash-001")
	req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

	// Pre-compute the request hash the way the handler does, so the
	// fake can return a matching record. Mirrors store.OrderRequestHash.
	h2 := store.OrderRequestHash(http.MethodPost, "/api/store/orders", body, 42)
	fs.winnerHash = h2

	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if !fs.createCalled {
		t.Fatal("CreateOrder was not called on the store")
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 4242 {
		t.Errorf("id = %d, want 4242 (winner order id)", resp.ID)
	}
	if resp.Type != "cart" {
		t.Errorf("type = %q, want cart", resp.Type)
	}
}

// TestCreateOrderIdempotencyRaceDifferentHashReturns409 verifies the
// boundary: race-loser with a *different* request hash (a true
// conflict, not a benign double-click) gets 409 IDEMPOTENCY_REPLAY.
func TestCreateOrderIdempotencyRaceDifferentHashReturns409(t *testing.T) {
	fs := &raceLoserFakeStore{
		winnerOrder: &model.Order{ID: 9999, Type: "cart"},
		winnerHash:  "DIFFERENT_HASH_FROM_WINNER",
		winnerBody:  `{"id":9999}`,
	}
	h := newCustomerHandlerForTest(fs)

	body := `{
		"type": "cart",
		"phone": "+37360000000",
		"city": "Тирасполь",
		"delivery_method": "personal",
		"payment_method": "card",
		"items": [{"product_id": 105, "size_label": "M", "quantity": 1}],
		"total_minor": 50000
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/store/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-race-diff-hash-001")
	req = req.WithContext(middleware.WithCustomerID(req.Context(), 42))

	rr := httptest.NewRecorder()
	h.CreateOrder(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body: %s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != "IDEMPOTENCY_REPLAY" {
		t.Errorf("code = %q, want IDEMPOTENCY_REPLAY", resp.Code)
	}
}

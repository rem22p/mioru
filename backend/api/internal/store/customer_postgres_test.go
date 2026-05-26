package store

import (
	"context"
	"testing"
	"time"

	"mioru/internal/model"
)

// TestUpdateCustomerPasswordBumpsEpoch verifies that changing a customer's
// password advances password_changed_at (the session-revocation epoch), and that
// the lookup reports a missing customer via ok=false.
func TestUpdateCustomerPasswordBumpsEpoch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "epoch-cust@example.com", HashedPW: "h1", FirstName: "Epoch",
	}); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, err := s.GetCustomerByEmail(ctx, "epoch-cust@example.com")
	if err != nil || cust == nil {
		t.Fatalf("GetCustomerByEmail: cust=%v err=%v", cust, err)
	}

	e1, ok, err := s.CustomerPasswordChangedAt(ctx, cust.ID)
	if err != nil || !ok {
		t.Fatalf("CustomerPasswordChangedAt: ok=%v err=%v", ok, err)
	}

	time.Sleep(10 * time.Millisecond)
	if err := s.UpdateCustomerPassword(ctx, cust.ID, "h2"); err != nil {
		t.Fatalf("UpdateCustomerPassword: %v", err)
	}

	e2, ok, err := s.CustomerPasswordChangedAt(ctx, cust.ID)
	if err != nil || !ok {
		t.Fatalf("CustomerPasswordChangedAt after update: ok=%v err=%v", ok, err)
	}
	if !e2.After(e1) {
		t.Errorf("password_changed_at did not advance: before=%v after=%v", e1, e2)
	}

	if _, ok, err := s.CustomerPasswordChangedAt(ctx, 999999); err != nil || ok {
		t.Errorf("expected ok=false,err=nil for missing customer; got ok=%v err=%v", ok, err)
	}
}

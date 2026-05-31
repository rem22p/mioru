package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

// TestListCustomerOrdersIsolation verifies two critical properties:
// 1. A customer sees only their own orders (IDOR/scoping guarantee).
// 2. Pagination — total reflects the correct customer, LIMIT/OFFSET works.
func TestListCustomerOrdersIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Seed two customers.
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "alice@example.com", HashedPW: "h1", FirstName: "Alice",
	}); err != nil {
		t.Fatalf("CreateCustomer A: %v", err)
	}
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "bob@example.com", HashedPW: "h2", FirstName: "Bob",
	}); err != nil {
		t.Fatalf("CreateCustomer B: %v", err)
	}

	a, err := s.GetCustomerByEmail(ctx, "alice@example.com")
	if err != nil || a == nil {
		t.Fatalf("GetCustomerByEmail A: cust=%v err=%v", a, err)
	}
	b, err := s.GetCustomerByEmail(ctx, "bob@example.com")
	if err != nil || b == nil {
		t.Fatalf("GetCustomerByEmail B: cust=%v err=%v", b, err)
	}

	// Insert 3 orders for Alice, 2 for Bob.
	for i := 0; i < 3; i++ {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO orders (customer_id, total_minor, status) VALUES ($1, $2, 'pending')`,
			a.ID, int64(1000+i*100),
		)
		if err != nil {
			t.Fatalf("insert Alice order: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := s.pool.Exec(ctx,
			`INSERT INTO orders (customer_id, total_minor, status) VALUES ($1, $2, 'processing')`,
			b.ID, int64(2000+i*100),
		)
		if err != nil {
			t.Fatalf("insert Bob order: %v", err)
		}
	}

	// Alice sees only her 3 orders.
	orders, total, err := s.ListCustomerOrders(ctx, a.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListCustomerOrders A: %v", err)
	}
	if total != 3 {
		t.Errorf("Alice total: want 3, got %d", total)
	}
	if len(orders) != 3 {
		t.Errorf("Alice orders len: want 3, got %d", len(orders))
	}
	for _, o := range orders {
		if o.CustomerID != a.ID {
			t.Errorf("Alice order %d has customer_id %d, want %d", o.ID, o.CustomerID, a.ID)
		}
	}

	// Bob sees only his 2 orders.
	orders2, total2, err := s.ListCustomerOrders(ctx, b.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListCustomerOrders B: %v", err)
	}
	if total2 != 2 {
		t.Errorf("Bob total: want 2, got %d", total2)
	}
	if len(orders2) != 2 {
		t.Errorf("Bob orders len: want 2, got %d", len(orders2))
	}
	for _, o := range orders2 {
		if o.CustomerID != b.ID {
			t.Errorf("Bob order %d has customer_id %d, want %d", o.ID, o.CustomerID, b.ID)
		}
	}

	// Pagination: Alice page 1 with per_page=2 → 2 items, total still 3.
	orders3, total3, err := s.ListCustomerOrders(ctx, a.ID, 1, 2)
	if err != nil {
		t.Fatalf("ListCustomerOrders A page=1 per=2: %v", err)
	}
	if total3 != 3 {
		t.Errorf("pagination total: want 3, got %d", total3)
	}
	if len(orders3) != 2 {
		t.Errorf("pagination len: want 2, got %d", len(orders3))
	}

	// Alice page 2 with per_page=2 → 1 remaining item.
	orders4, total4, err := s.ListCustomerOrders(ctx, a.ID, 2, 2)
	if err != nil {
		t.Fatalf("ListCustomerOrders A page=2 per=2: %v", err)
	}
	if total4 != 3 {
		t.Errorf("pagination page2 total: want 3, got %d", total4)
	}
	if len(orders4) != 1 {
		t.Errorf("pagination page2 len: want 1, got %d", len(orders4))
	}

	// Newest-first ordering: Alice has 3 orders inserted sequentially,
	// the one with highest ID (last inserted) should be first.
	if len(orders) > 1 && orders[0].ID < orders[1].ID {
		t.Errorf("orders not newest-first: [0].ID=%d < [1].ID=%d", orders[0].ID, orders[1].ID)
	}
}

package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

// TestSaveAndGetCustomerCartRoundTrip verifies cart persistence end-to-end:
// - round-trip: Save → Get returns the same items
// - replace semantics: second Save completely replaces previous items
// - isolation: customer A never sees customer B's cart
func TestSaveAndGetCustomerCartRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create products that the cart items will reference.
	pid1 := mustCreateProduct(t, s, model.Product{
		Slug: "cart-p1", CategoryID: 2, Brand: "Test", Name: "Cart P1", Price: 100,
		Status: "in_stock", InStock: true,
	})
	pid2 := mustCreateProduct(t, s, model.Product{
		Slug: "cart-p2", CategoryID: 2, Brand: "Test", Name: "Cart P2", Price: 200,
		Status: "in_stock", InStock: true,
	})
	pid3 := mustCreateProduct(t, s, model.Product{
		Slug: "cart-p3", CategoryID: 2, Brand: "Test", Name: "Cart P3", Price: 300,
		Status: "in_stock", InStock: true,
	})

	// Seed two customers.
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "cart-a@example.com", HashedPW: "h1", FirstName: "CartA",
	}); err != nil {
		t.Fatalf("CreateCustomer A: %v", err)
	}
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "cart-b@example.com", HashedPW: "h2", FirstName: "CartB",
	}); err != nil {
		t.Fatalf("CreateCustomer B: %v", err)
	}

	a, _ := s.GetCustomerByEmail(ctx, "cart-a@example.com")
	b, _ := s.GetCustomerByEmail(ctx, "cart-b@example.com")
	if a == nil || b == nil {
		t.Fatal("customers not found after create")
	}

	// ── Round-trip for customer A ──
	itemsA := []CartItem{
		{ProductID: int(pid1), SizeLabel: "M", Quantity: 2},
		{ProductID: int(pid2), SizeLabel: "L", Quantity: 1},
	}
	if err := s.SaveCustomerCart(ctx, a.ID, itemsA); err != nil {
		t.Fatalf("SaveCustomerCart A: %v", err)
	}

	got, err := s.GetCustomerCart(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetCustomerCart A: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("round-trip A: want 2 items, got %d", len(got))
	}
	if got[0].ProductID != int(pid1) || got[0].SizeLabel != "M" || got[0].Quantity != 2 {
		t.Errorf("round-trip A[0]: got %+v, want {%d M 2}", got[0], pid1)
	}
	if got[1].ProductID != int(pid2) || got[1].SizeLabel != "L" || got[1].Quantity != 1 {
		t.Errorf("round-trip A[1]: got %+v, want {%d L 1}", got[1], pid2)
	}

	// ── Replace semantics: save new set, old items vanish ──
	itemsA2 := []CartItem{
		{ProductID: int(pid3), SizeLabel: "S", Quantity: 5},
	}
	if err := s.SaveCustomerCart(ctx, a.ID, itemsA2); err != nil {
		t.Fatalf("SaveCustomerCart A (replace): %v", err)
	}
	got2, err := s.GetCustomerCart(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetCustomerCart A after replace: %v", err)
	}
	if len(got2) != 1 {
		t.Fatalf("replace: want 1 item, got %d", len(got2))
	}
	if got2[0].ProductID != int(pid3) {
		t.Errorf("replace: want product %d, got %d", pid3, got2[0].ProductID)
	}

	// ── Save empty cart ──
	if err := s.SaveCustomerCart(ctx, a.ID, nil); err != nil {
		t.Fatalf("SaveCustomerCart empty: %v", err)
	}
	got3, err := s.GetCustomerCart(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetCustomerCart empty: %v", err)
	}
	if len(got3) != 0 {
		t.Errorf("empty cart: want 0 items, got %d", len(got3))
	}

	// ── Isolation: customer B sees only their own cart ──
	itemsB := []CartItem{
		{ProductID: int(pid1), SizeLabel: "XL", Quantity: 10},
	}
	if err := s.SaveCustomerCart(ctx, b.ID, itemsB); err != nil {
		t.Fatalf("SaveCustomerCart B: %v", err)
	}

	gotB, err := s.GetCustomerCart(ctx, b.ID)
	if err != nil {
		t.Fatalf("GetCustomerCart B: %v", err)
	}
	if len(gotB) != 1 || gotB[0].ProductID != int(pid1) || gotB[0].Quantity != 10 {
		t.Errorf("isolation B: got %+v, want [{%d XL 10}]", gotB, pid1)
	}

	// A's cart is still empty (from the empty save above), not polluted by B.
	gotAafter, err := s.GetCustomerCart(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetCustomerCart A after B save: %v", err)
	}
	if len(gotAafter) != 0 {
		t.Errorf("isolation: A should be empty after A emptied, got %d items", len(gotAafter))
	}
}

// TestSaveAndGetCustomerFavoritesRoundTrip verifies favorites persistence:
// - round-trip: Save → Get returns the same IDs
// - replace semantics: second Save replaces previous set
// - isolation: customer A never sees customer B's favorites
func TestSaveAndGetCustomerFavoritesRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Create products that favorites will reference.
	pid1 := mustCreateProduct(t, s, model.Product{
		Slug: "fav-p1", CategoryID: 2, Brand: "Test", Name: "Fav P1", Price: 100,
		Status: "in_stock", InStock: true,
	})
	pid2 := mustCreateProduct(t, s, model.Product{
		Slug: "fav-p2", CategoryID: 2, Brand: "Test", Name: "Fav P2", Price: 200,
		Status: "in_stock", InStock: true,
	})
	pid3 := mustCreateProduct(t, s, model.Product{
		Slug: "fav-p3", CategoryID: 2, Brand: "Test", Name: "Fav P3", Price: 300,
		Status: "in_stock", InStock: true,
	})
	pid4 := mustCreateProduct(t, s, model.Product{
		Slug: "fav-p4", CategoryID: 2, Brand: "Test", Name: "Fav P4", Price: 400,
		Status: "in_stock", InStock: true,
	})

	// Seed two customers.
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "fav-a@example.com", HashedPW: "h1", FirstName: "FavA",
	}); err != nil {
		t.Fatalf("CreateCustomer A: %v", err)
	}
	if err := s.CreateCustomer(ctx, model.Customer{
		Email: "fav-b@example.com", HashedPW: "h2", FirstName: "FavB",
	}); err != nil {
		t.Fatalf("CreateCustomer B: %v", err)
	}

	a, _ := s.GetCustomerByEmail(ctx, "fav-a@example.com")
	b, _ := s.GetCustomerByEmail(ctx, "fav-b@example.com")

	// Round-trip for A.
	idsA := []int{int(pid1), int(pid2), int(pid3)}
	if err := s.SaveCustomerFavorites(ctx, a.ID, idsA); err != nil {
		t.Fatalf("SaveCustomerFavorites A: %v", err)
	}

	got, err := s.GetCustomerFavorites(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetCustomerFavorites A: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("round-trip A: want 3 IDs, got %d", len(got))
	}

	// Replace: save overlapping set.
	idsA2 := []int{int(pid2), int(pid3), int(pid4)}
	if err := s.SaveCustomerFavorites(ctx, a.ID, idsA2); err != nil {
		t.Fatalf("SaveCustomerFavorites A (replace): %v", err)
	}
	got2, err := s.GetCustomerFavorites(ctx, a.ID)
	if err != nil {
		t.Fatalf("GetCustomerFavorites A after replace: %v", err)
	}
	if len(got2) != 3 {
		t.Fatalf("replace A: want 3 IDs, got %d (%v)", len(got2), got2)
	}

	// Save empty → clears.
	if err := s.SaveCustomerFavorites(ctx, a.ID, nil); err != nil {
		t.Fatalf("SaveCustomerFavorites empty: %v", err)
	}
	got3, _ := s.GetCustomerFavorites(ctx, a.ID)
	if len(got3) != 0 {
		t.Errorf("empty favorites: want 0, got %d", len(got3))
	}

	// Isolation: B's favorites don't leak to A.
	idsB := []int{int(pid1)}
	s.SaveCustomerFavorites(ctx, b.ID, idsB)
	gotA, _ := s.GetCustomerFavorites(ctx, a.ID)
	gotB, _ := s.GetCustomerFavorites(ctx, b.ID)
	if len(gotA) != 0 {
		t.Errorf("isolation: A should have 0, got %d", len(gotA))
	}
	if len(gotB) != 1 || gotB[0] != int(pid1) {
		t.Errorf("isolation: B should have [%d], got %v", pid1, gotB)
	}
}

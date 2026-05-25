package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

// TestHarnessSeededCategories proves the harness migrates the seeded category
// tree: the four root categories must be present after a reset.
func TestHarnessSeededCategories(t *testing.T) {
	s := testStore(t)

	roots, err := s.GetCategories(context.Background())
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("root categories = %d, want 4 (Одежда, Обувь, Сумки, Аксессуары)", len(roots))
	}
}

// TestHarnessCreateAndGetProduct proves the harness gives a clean, writable DB:
// a created product (with sizes) reads back intact through the store.
func TestHarnessCreateAndGetProduct(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	id, err := s.CreateProduct(ctx, model.Product{
		Slug:       "harness-tee",
		CategoryID: 2, // Футболки / поло
		Brand:      "ACME",
		Name:       "Harness Tee",
		Price:      1999,
		Status:     "in_stock",
		InStock:    true,
		Care:       []string{"machine wash"},
		Sizes:      []string{"M", "L"},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if id == 0 {
		t.Fatal("CreateProduct returned id 0")
	}

	got, err := s.GetProduct(ctx, "harness-tee")
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if got.Name != "Harness Tee" || got.Brand != "ACME" || got.Price != 1999 {
		t.Errorf("product mismatch: name=%q brand=%q price=%d", got.Name, got.Brand, got.Price)
	}
	if got.CategoryName != "Футболки / поло" {
		t.Errorf("category_name = %q, want %q", got.CategoryName, "Футболки / поло")
	}
	if len(got.Sizes) != 2 {
		t.Errorf("sizes = %v, want 2", got.Sizes)
	}
}

// TestHarnessResetIsolation proves resetTables wipes data between tests: each
// fresh testStore starts with zero products even after another test inserted one.
func TestHarnessResetIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	_, total, err := s.ListProducts(ctx, model.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected empty DB after reset, got %d products", total)
	}
}

package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

// TestCollabBrandsRoundTrip pins KAN-14: a collaboration product stores its
// brands as an array, the display name is derived as "A x B", and both brands
// are returned to the client.
func TestCollabBrandsRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	id := mustCreateProduct(t, s, model.Product{
		Slug: "collab-1", CategoryID: 2, Brands: []string{"Bape", "Mastermind"},
		Name: "Collab Tee", Price: 100, Status: "in_stock", InStock: true,
	})
	if id == 0 {
		t.Fatal("expected a product id")
	}

	got, err := s.GetProduct(ctx, "collab-1")
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if got.Brand != "Bape x Mastermind" {
		t.Errorf("display Brand = %q, want %q", got.Brand, "Bape x Mastermind")
	}
	if len(got.Brands) != 2 || got.Brands[0] != "Bape" || got.Brands[1] != "Mastermind" {
		t.Errorf("Brands = %v, want [Bape Mastermind]", got.Brands)
	}
}

// TestCollabBrandsFilterMatchesEitherBrand pins the manager's requirement:
// picking EITHER brand of a collaboration surfaces the product.
func TestCollabBrandsFilterMatchesEitherBrand(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "collab-tee", CategoryID: 2, Brands: []string{"Bape", "Mastermind"},
		Name: "Collab Tee", Price: 100, Status: "in_stock", InStock: true,
	})
	mustCreateProduct(t, s, model.Product{
		Slug: "solo-tee", CategoryID: 2, Brands: []string{"Nike"},
		Name: "Solo Tee", Price: 100, Status: "in_stock", InStock: true,
	})

	for _, brand := range []string{"Bape", "Mastermind"} {
		got, total, err := s.ListProducts(ctx, model.ProductFilter{Brands: []string{brand}})
		if err != nil {
			t.Fatalf("ListProducts(brand=%s): %v", brand, err)
		}
		if total != 1 || len(got) != 1 || got[0].Slug != "collab-tee" {
			t.Errorf("filter brand=%s: total=%d slugs=%v, want only collab-tee",
				brand, total, slugsOf(got))
		}
	}

	// Multi-select: either brand still matches (overlap semantics, not AND).
	_, total, err := s.ListProducts(ctx, model.ProductFilter{Brands: []string{"Bape", "Nike"}})
	if err != nil {
		t.Fatalf("ListProducts(brands=[Bape Nike]): %v", err)
	}
	if total != 2 {
		t.Errorf("brands=[Bape Nike]: total=%d, want 2 (overlap)", total)
	}
}

// TestCollabBrandsFacetsSplit pins KAN-14: facets list collaboration brands
// individually, not as a single "Bape x Mastermind" string.
func TestCollabBrandsFacetsSplit(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "collab-shoes", CategoryID: 2, Brands: []string{"Nike", "Off-White"},
		Name: "Collab Shoes", Price: 100, Status: "in_stock", InStock: true,
	})
	mustCreateProduct(t, s, model.Product{
		Slug: "solo-hat", CategoryID: 2, Brands: []string{"Adidas"},
		Name: "Solo Hat", Price: 50, Status: "in_stock", InStock: true,
	})

	facets, err := s.ListProductFacets(ctx, model.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProductFacets: %v", err)
	}
	if !contains(facets.Brands, "Nike") || !contains(facets.Brands, "Off-White") || !contains(facets.Brands, "Adidas") {
		t.Errorf("facets.Brands = %v, want Nike, Off-White and Adidas listed individually", facets.Brands)
	}
	if contains(facets.Brands, "Nike x Off-White") {
		t.Errorf("facets.Brands = %v, must not contain the joined display name", facets.Brands)
	}
}

func slugsOf(ps []model.Product) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Slug
	}
	return out
}

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

// TestCollabBrandsSortByBrand pins the storefront sort contract across the
// column change: "sort=brand" was served by the dropped `brand` column, so it
// has to be answered by the derived display name now.
func TestCollabBrandsSortByBrand(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "sort-z", CategoryID: 2, Brands: []string{"Zara"},
		Name: "Z", Price: 100, Status: "in_stock", InStock: true,
	})
	mustCreateProduct(t, s, model.Product{
		Slug: "sort-a", CategoryID: 2, Brands: []string{"Adidas", "Bape"},
		Name: "A", Price: 100, Status: "in_stock", InStock: true,
	})

	got, _, err := s.ListProducts(ctx, model.ProductFilter{Sort: "brand", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("ListProducts(sort=brand): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d products, want 2", len(got))
	}
	// "Adidas x Bape" sorts before "Zara" — the display name is the key.
	if got[0].Brand != "Adidas x Bape" || got[1].Brand != "Zara" {
		t.Errorf("order = [%q %q], want [\"Adidas x Bape\" \"Zara\"]", got[0].Brand, got[1].Brand)
	}

	desc, _, err := s.ListProducts(ctx, model.ProductFilter{Sort: "-brand", Page: 1, PerPage: 20})
	if err != nil {
		t.Fatalf("ListProducts(sort=-brand): %v", err)
	}
	if desc[0].Brand != "Zara" {
		t.Errorf("descending order starts with %q, want \"Zara\"", desc[0].Brand)
	}
}

// TestCollabBrandsFacetsSkipEmpty pins that an empty array element never
// reaches the facet list — the 028 backfill can produce one from a legacy
// value like "Bape x " (string_to_array leaves a trailing empty element).
func TestCollabBrandsFacetsSkipEmpty(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "facet-empty", CategoryID: 2, Brands: []string{"Bape", ""},
		Name: "FE", Price: 100, Status: "in_stock", InStock: true,
	})

	facets, err := s.ListProductFacets(ctx, model.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProductFacets: %v", err)
	}
	for _, b := range facets.Brands {
		if b == "" {
			t.Fatalf("empty brand surfaced in facets: %q", facets.Brands)
		}
	}
}

// TestCollabBrandsAdminFilterMatchesDisplayName pins #86 R2: a brand filter
// carrying the joined display name ("Bape x Mastermind") also matches the
// collab product — the admin list filter is free-text.
func TestCollabBrandsAdminFilterMatchesDisplayName(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "collab-display", CategoryID: 2, Brands: []string{"Bape", "Mastermind"},
		Name: "Collab Tee", Price: 100, Status: "in_stock", InStock: true,
	})

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Brand: "Bape x Mastermind"})
	if err != nil {
		t.Fatalf("ListProducts(display name): %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].Slug != "collab-display" {
		t.Errorf("display-name filter: total=%d slugs=%v, want collab-display", total, slugsOf(got))
	}
}

// TestCollabBrandsChipMatchIsExact pins the storefront semantics: picking
// the "Bape" chip must not surface "Bape Kids" — the substring branch is
// for the admin free-text filter (filter.Brand) only.
func TestCollabBrandsChipMatchIsExact(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "bape-kids", CategoryID: 2, Brands: []string{"Bape Kids"},
		Name: "Bape Kids Tee", Price: 100, Status: "in_stock", InStock: true,
	})

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Brands: []string{"Bape"}})
	if err != nil {
		t.Fatalf("ListProducts(chip): %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Errorf("chip \"Bape\" surfaced %d products, want 0 (substring must not leak into chips)", total)
	}
}

// TestCollabBrandsFreeTextEscapesWildcards pins the admin filter: % and _
// in the query stay literal characters, not LIKE wildcards.
func TestCollabBrandsFreeTextEscapesWildcards(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "nike-air", CategoryID: 2, Brands: []string{"Nike"},
		Name: "Nike Air", Price: 100, Status: "in_stock", InStock: true,
	})

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Brand: "N_ke"})
	if err != nil {
		t.Fatalf("ListProducts(wildcard): %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Errorf("query \"N_ke\" surfaced %d products, want 0 (underscore must stay literal)", total)
	}
}

// TestCollabBrandsFreeTextEscapesPercent pins the % wildcard too.
func TestCollabBrandsFreeTextEscapesPercent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "adidas-50", CategoryID: 2, Brands: []string{"Adidas 50"},
		Name: "Adidas 50", Price: 100, Status: "in_stock", InStock: true,
	})

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Brand: "50%"})
	if err != nil {
		t.Fatalf("ListProducts(percent): %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Errorf("query \"50%%\" surfaced %d products, want 0 (percent must stay literal)", total)
	}
}

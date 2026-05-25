package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

// mustCreateProduct inserts a product and fails the test on error.
func mustCreateProduct(t *testing.T, s *PostgresStore, p model.Product) int64 {
	t.Helper()
	id, err := s.CreateProduct(context.Background(), p)
	if err != nil {
		t.Fatalf("CreateProduct(%s): %v", p.Slug, err)
	}
	return id
}

// TestListProductsAttachesRelatedData verifies ListProducts returns each
// product with its sizes, size chart, and images attached in the right order,
// and that results follow the requested sort. This is the regression guard for
// the batch (N+1 → 5 queries) refactor.
func TestListProductsAttachesRelatedData(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{
		Slug: "alpha", CategoryID: 2, Brand: "ACME", Name: "Alpha", Price: 100,
		Status: "in_stock", InStock: true,
		Sizes:  []string{"S", "M", "L"},
		Images: []model.ProductImage{{URL: "/a1.jpg"}, {URL: "/a2.jpg"}},
		SizeChart: []model.SizeChartRow{
			{Label: "S"}, {Label: "M"},
		},
	})
	mustCreateProduct(t, s, model.Product{
		Slug: "bravo", CategoryID: 2, Brand: "ACME", Name: "Bravo", Price: 200,
		Status: "in_stock", InStock: true,
		Sizes:  []string{"M"},
	})

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Sort: "name"})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Fatalf("len(products) = %d, want 2", len(got))
	}

	// Sort "name" ascending → Alpha, Bravo.
	if got[0].Slug != "alpha" || got[1].Slug != "bravo" {
		t.Fatalf("order = [%s, %s], want [alpha, bravo]", got[0].Slug, got[1].Slug)
	}

	alpha := got[0]
	if want := []string{"S", "M", "L"}; !equalStrings(alpha.Sizes, want) {
		t.Errorf("alpha.Sizes = %v, want %v", alpha.Sizes, want)
	}
	if len(alpha.Images) != 2 || alpha.Images[0].URL != "/a1.jpg" || alpha.Images[1].URL != "/a2.jpg" {
		t.Errorf("alpha.Images = %+v, want [/a1.jpg /a2.jpg] in order", alpha.Images)
	}
	if len(alpha.SizeChart) != 2 || alpha.SizeChart[0].Label != "S" || alpha.SizeChart[1].Label != "M" {
		t.Errorf("alpha.SizeChart = %+v, want labels [S M]", alpha.SizeChart)
	}

	// A product without related rows must carry empty (non-nil) slices so the
	// JSON contract stays [] rather than null.
	bravo := got[1]
	if bravo.SizeChart == nil || bravo.Images == nil {
		t.Errorf("bravo empty relations must be non-nil slices: chart=%v images=%v", bravo.SizeChart, bravo.Images)
	}
	if len(bravo.SizeChart) != 0 || len(bravo.Images) != 0 {
		t.Errorf("bravo should have no chart/images, got chart=%v images=%v", bravo.SizeChart, bravo.Images)
	}
}

// TestListProductsPagination verifies page/per_page slice the result set while
// total reflects the full match count.
func TestListProductsPagination(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Names chosen so ascending sort is p1..p5.
	for _, n := range []string{"p1", "p2", "p3", "p4", "p5"} {
		mustCreateProduct(t, s, model.Product{
			Slug: n, CategoryID: 2, Name: n, Price: 100, Status: "in_stock", InStock: true,
		})
	}

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Sort: "name", Page: 2, PerPage: 2})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Slug != "p3" || got[1].Slug != "p4" {
		t.Errorf("page 2 = [%s, %s], want [p3, p4]", got[0].Slug, got[1].Slug)
	}
}

// TestListProductsFilters verifies category, brand, and search filtering.
func TestListProductsFilters(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{Slug: "tee-acme", CategoryID: 2, Brand: "ACME", Name: "Cotton Tee", Price: 100, Status: "in_stock", InStock: true})
	mustCreateProduct(t, s, model.Product{Slug: "tee-nike", CategoryID: 2, Brand: "Nike", Name: "Sport Tee", Price: 100, Status: "in_stock", InStock: true})
	mustCreateProduct(t, s, model.Product{Slug: "sneaker-nike", CategoryID: 12, Brand: "Nike", Name: "Runner", Price: 100, Status: "in_stock", InStock: true})

	tests := []struct {
		name      string
		filter    model.ProductFilter
		wantSlugs []string
	}{
		{"by category", model.ProductFilter{CategoryID: 12, Sort: "name"}, []string{"sneaker-nike"}},
		{"by brand", model.ProductFilter{Brand: "Nike", Sort: "name"}, []string{"sneaker-nike", "tee-nike"}},
		{"by search name", model.ProductFilter{Search: "Cotton", Sort: "name"}, []string{"tee-acme"}},
		{"by search slug", model.ProductFilter{Search: "sneaker", Sort: "name"}, []string{"sneaker-nike"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := s.ListProducts(ctx, tt.filter)
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if total != len(tt.wantSlugs) {
				t.Errorf("total = %d, want %d", total, len(tt.wantSlugs))
			}
			var slugs []string
			for _, p := range got {
				slugs = append(slugs, p.Slug)
			}
			if !equalStrings(slugs, tt.wantSlugs) {
				t.Errorf("slugs = %v, want %v", slugs, tt.wantSlugs)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

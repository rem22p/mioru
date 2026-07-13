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
		Sizes: []model.ProductSize{model.ProductSize{Label: "S"}, model.ProductSize{Label: "M"}, model.ProductSize{Label: "L"}},
		Images: []model.ProductImage{{URL: "/a1.jpg"}, {URL: "/a2.jpg"}},
		SizeChart: []model.SizeChartRow{
			{Label: "S"}, {Label: "M"},
		},
	})
	mustCreateProduct(t, s, model.Product{
		Slug: "bravo", CategoryID: 2, Brand: "ACME", Name: "Bravo", Price: 200,
		Status: "in_stock", InStock: true,
		Sizes: []model.ProductSize{model.ProductSize{Label: "M"}},
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
	if want := []model.ProductSize{{Label: "S"}, {Label: "M"}, {Label: "L"}}; !equalSizeLabels(alpha.Sizes, want) {
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
		// Fuzzy: "cotn" is a typo of "Cotton", trigram similarity ≈ 0.3 > 0.2 threshold.
		{"by fuzzy typo", model.ProductFilter{Search: "cotn", Sort: "name"}, []string{"tee-acme"}},
		// Clamp: search >200 chars is truncated to 200, query still works on prefix.
		{"clamped search", model.ProductFilter{Search: strings.Repeat("x", 250) + "Cotton", Sort: "name"}, []string{}},
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

// TestListProductsMultiFilters verifies brand/color/size/price multi-value
// filters narrow the result set the way the storefront filter UI expects.
func TestListProductsMultiFilters(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{Slug: "red-s", CategoryID: 2, Brand: "ACME", Name: "Red S", Color: "red", Price: 100, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "S"}}})
	mustCreateProduct(t, s, model.Product{Slug: "red-m", CategoryID: 2, Brand: "ACME", Name: "Red M", Color: "red", Price: 200, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "M"}}})
	mustCreateProduct(t, s, model.Product{Slug: "blue-m", CategoryID: 2, Brand: "Nike", Name: "Blue M", Color: "blue", Price: 300, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "M"}}})
	mustCreateProduct(t, s, model.Product{Slug: "green-l", CategoryID: 2, Brand: "Nike", Name: "Green L", Color: "green", Price: 400, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "L"}}})

	tests := []struct {
		name      string
		filter    model.ProductFilter
		wantSlugs []string
	}{
		{
			"brands multi", model.ProductFilter{Brands: []string{"ACME", "Nike"}, Sort: "name"},
			[]string{"blue-m", "green-l", "red-m", "red-s"},
		},
		{
			"colors multi", model.ProductFilter{Colors: []string{"red", "blue"}, Sort: "name"},
			[]string{"blue-m", "red-m", "red-s"},
		},
		{
			"size single", model.ProductFilter{Sizes: []string{"M"}, Sort: "name"},
			[]string{"blue-m", "red-m"},
		},
		{
			"price range", model.ProductFilter{PriceMin: 200, PriceMax: 300, Sort: "name"},
			[]string{"blue-m", "red-m"},
		},
		{
			"combined: red OR blue, size M, price>=200",
			model.ProductFilter{Colors: []string{"red", "blue"}, Sizes: []string{"M"}, PriceMin: 200, Sort: "name"},
			[]string{"blue-m", "red-m"},
		},
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

// TestListProductFacets verifies facets enumerate distinct brand/color/size
// values within the filter scope and that empty values are dropped.
func TestListProductFacets(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{Slug: "p1", CategoryID: 2, Brand: "ACME", Name: "P1", Color: "red", Price: 100, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "S"}, model.ProductSize{Label: "M"}}})
	mustCreateProduct(t, s, model.Product{Slug: "p2", CategoryID: 2, Brand: "Nike", Name: "P2", Color: "blue", Price: 200, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "M"}, model.ProductSize{Label: "L"}}})
	mustCreateProduct(t, s, model.Product{Slug: "p3", CategoryID: 12, Brand: "Puma", Name: "P3", Color: "", Price: 300, Status: "in_stock", InStock: true, Sizes: []model.ProductSize{model.ProductSize{Label: "XL"}}})

	// Without scope: brand/color/size from all categories, empty color dropped.
	all, err := s.ListProductFacets(ctx, model.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProductFacets: %v", err)
	}
	if !equalStrings(all.Brands, []string{"ACME", "Nike", "Puma"}) {
		t.Errorf("brands = %v, want [ACME Nike Puma]", all.Brands)
	}
	if !equalStrings(all.Colors, []string{"blue", "red"}) {
		t.Errorf("colors = %v, want [blue red] (empty dropped)", all.Colors)
	}
	if !equalStrings(all.Sizes, []string{"L", "M", "S", "XL"}) {
		t.Errorf("sizes = %v, want [L M S XL]", all.Sizes)
	}

	// Scoped by category 2: Puma/p3 must be excluded.
	cat2, err := s.ListProductFacets(ctx, model.ProductFilter{CategoryID: 2})
	if err != nil {
		t.Fatalf("ListProductFacets cat=2: %v", err)
	}
	if !equalStrings(cat2.Brands, []string{"ACME", "Nike"}) {
		t.Errorf("cat=2 brands = %v, want [ACME Nike]", cat2.Brands)
	}
	if !equalStrings(cat2.Sizes, []string{"L", "M", "S"}) {
		t.Errorf("cat=2 sizes = %v, want [L M S]", cat2.Sizes)
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

func equalSizeLabels(a, b []model.ProductSize) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Label != b[i].Label {
			return false
		}
	}
	return true
}

// TestListProductsByStatus verifies the storefront catalog honours the
// status filter (in_stock | preorder | out_of_stock). It is the regression
// guard for the catalog "В наличии / Под заказ" toggle wired by the
// feat/catalog-status-toggle change.
func TestListProductsByStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{Slug: "is-1", CategoryID: 2, Name: "IS1", Price: 100, Status: "in_stock", InStock: true})
	mustCreateProduct(t, s, model.Product{Slug: "is-2", CategoryID: 2, Name: "IS2", Price: 100, Status: "in_stock", InStock: true})
	mustCreateProduct(t, s, model.Product{Slug: "po-1", CategoryID: 2, Name: "PO1", Price: 100, Status: "preorder", InStock: false, StockQty: 0})
	mustCreateProduct(t, s, model.Product{Slug: "po-2", CategoryID: 2, Name: "PO2", Price: 100, Status: "preorder", InStock: false, StockQty: 0})

	// Filter: in_stock → exactly 2
	got, total, err := s.ListProducts(ctx, model.ProductFilter{Status: "in_stock", Sort: "name"})
	if err != nil {
		t.Fatalf("ListProducts(in_stock): %v", err)
	}
	if total != 2 {
		t.Errorf("in_stock total = %d, want 2", total)
	}
	for _, p := range got {
		if p.Status != "in_stock" {
			t.Errorf("in_stock filter returned %s, want only in_stock", p.Status)
		}
	}

	// Filter: preorder → exactly 2
	got, total, err = s.ListProducts(ctx, model.ProductFilter{Status: "preorder", Sort: "name"})
	if err != nil {
		t.Fatalf("ListProducts(preorder): %v", err)
	}
	if total != 2 {
		t.Errorf("preorder total = %d, want 2", total)
	}
	for _, p := range got {
		if p.Status != "preorder" {
			t.Errorf("preorder filter returned %s, want only preorder", p.Status)
		}
	}

	// Empty filter → all 4
	_, total, err = s.ListProducts(ctx, model.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProducts(no filter): %v", err)
	}
	if total < 4 {
		t.Errorf("unfiltered total = %d, want >= 4", total)
	}
}

// TestListProductFacetsByStatus documents the design decision: status is a
// 2-state toggle (in_stock | preorder), not a multi-value chip. Unlike brand/
// color/size — where the facets endpoint drops the user's selection so every
// option stays visible — status is kept in the facets query. The user expects
// the size chart and brand list to reflect "products in the currently toggled
// bucket", not "every product globally". This test exercises both code paths
// (filtered and unfiltered) so the SQL does not regress.
func TestListProductFacetsByStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	mustCreateProduct(t, s, model.Product{Slug: "is-facets", CategoryID: 2, Brand: "FACETS-IS", Name: "ISF", Price: 100, Status: "in_stock", InStock: true})
	mustCreateProduct(t, s, model.Product{Slug: "po-facets", CategoryID: 2, Brand: "FACETS-PO", Name: "POF", Price: 100, Status: "preorder", InStock: false, StockQty: 0})

	// Filtered: status=in_stock → only the in_stock brand should appear
	filtered, err := s.ListProductFacets(ctx, model.ProductFilter{Status: "in_stock"})
	if err != nil {
		t.Fatalf("ListProductFacets(in_stock): %v", err)
	}
	if !contains(filtered.Brands, "FACETS-IS") {
		t.Errorf("in_stock facets missing FACETS-IS brand, got %v", filtered.Brands)
	}
	if contains(filtered.Brands, "FACETS-PO") {
		t.Errorf("in_stock facets should NOT include FACETS-PO, got %v", filtered.Brands)
	}

	// Unfiltered → both brands appear
	unfiltered, err := s.ListProductFacets(ctx, model.ProductFilter{})
	if err != nil {
		t.Fatalf("ListProductFacets(no filter): %v", err)
	}
	if !contains(unfiltered.Brands, "FACETS-IS") || !contains(unfiltered.Brands, "FACETS-PO") {
		t.Errorf("unfiltered facets missing seeded brands, got %v", unfiltered.Brands)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// TestCreateProductRejectsUnknownStatus verifies the CHECK constraint on
// products.status rejects anything outside {in_stock, preorder, out_of_stock}.
// The constraint is the data-integrity backstop — the handler also validates,
// but a DB-level guarantee means a buggy code path can never poison the column.
func TestCreateProductRejectsUnknownStatus(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	_, err := s.CreateProduct(ctx, model.Product{
		Slug: "bad-status", CategoryID: 2, Name: "BAD", Price: 100,
		Status: "ghost", InStock: true,
	})
	if err == nil {
		t.Fatal("expected CreateProduct to reject unknown status, got nil")
	}
}

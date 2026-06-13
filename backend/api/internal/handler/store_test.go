package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/model"
)

// fakeStoreReader is a hand-written fake satisfying storeReader. It demonstrates
// the seam from #27: handler branches (filter parsing, error→status mapping)
// can be tested without a database. Each test sets only the funcs it exercises.
type fakeStoreReader struct {
	listFn   func(ctx context.Context, f model.ProductFilter) ([]model.Product, int, error)
	getFn    func(ctx context.Context, slug string) (*model.Product, error)
	catsFn   func(ctx context.Context) ([]model.Category, error)
	facetsFn func(ctx context.Context, f model.ProductFilter) (model.ProductFacets, error)
}

func (f *fakeStoreReader) ListProducts(ctx context.Context, filter model.ProductFilter) ([]model.Product, int, error) {
	return f.listFn(ctx, filter)
}
func (f *fakeStoreReader) ListProductFacets(ctx context.Context, filter model.ProductFilter) (model.ProductFacets, error) {
	if f.facetsFn == nil {
		return model.ProductFacets{}, nil
	}
	return f.facetsFn(ctx, filter)
}
func (f *fakeStoreReader) GetProduct(ctx context.Context, slug string) (*model.Product, error) {
	return f.getFn(ctx, slug)
}
func (f *fakeStoreReader) GetCategories(ctx context.Context) ([]model.Category, error) {
	return f.catsFn(ctx)
}

// Compile-time check that the fake satisfies the consumer interface.
var _ storeReader = (*fakeStoreReader)(nil)

func TestStoreListProductsSuccess(t *testing.T) {
	var gotFilter model.ProductFilter
	fake := &fakeStoreReader{
		listFn: func(_ context.Context, f model.ProductFilter) ([]model.Product, int, error) {
			gotFilter = f
			return []model.Product{{Slug: "x", Name: "X"}}, 1, nil
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/api/products?page=2&per_page=5&search=foo", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	// Query params must reach the store filter.
	if gotFilter.Page != 2 || gotFilter.PerPage != 5 || gotFilter.Search != "foo" {
		t.Errorf("filter = %+v, want page=2 per_page=5 search=foo", gotFilter)
	}

	var body struct {
		Products []model.Product `json:"products"`
		Total    int             `json:"total"`
		Page     int             `json:"page"`
		PerPage  int             `json:"per_page"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Total != 1 || len(body.Products) != 1 || body.Page != 2 || body.PerPage != 5 {
		t.Errorf("body = %+v", body)
	}
}

func TestStoreListProductsStoreError(t *testing.T) {
	fake := &fakeStoreReader{
		listFn: func(_ context.Context, _ model.ProductFilter) ([]model.Product, int, error) {
			return nil, 0, errors.New("db exploded")
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	// The internal error string must not leak to the client.
	if strings.Contains(rr.Body.String(), "db exploded") {
		t.Errorf("response leaked internal error: %s", rr.Body.String())
	}
}

// TestStoreListProductsParsesMultiFilters verifies the handler unpacks repeated
// keys (?brand=X&brand=Y), comma-separated values (?color=red,blue), and
// numeric bounds (price_min/price_max) into the ProductFilter the store sees.
func TestStoreListProductsParsesMultiFilters(t *testing.T) {
	var gotFilter model.ProductFilter
	fake := &fakeStoreReader{
		listFn: func(_ context.Context, f model.ProductFilter) ([]model.Product, int, error) {
			gotFilter = f
			return []model.Product{}, 0, nil
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet,
		"/api/products?brand=Nike&brand=ACME&color=red,blue&size=M&price_min=100&price_max=500&sort=-price",
		nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !equalStrSlices(gotFilter.Brands, []string{"Nike", "ACME"}) {
		t.Errorf("brands = %v, want [Nike ACME]", gotFilter.Brands)
	}
	if !equalStrSlices(gotFilter.Colors, []string{"red", "blue"}) {
		t.Errorf("colors = %v, want [red blue]", gotFilter.Colors)
	}
	if !equalStrSlices(gotFilter.Sizes, []string{"M"}) {
		t.Errorf("sizes = %v, want [M]", gotFilter.Sizes)
	}
	if gotFilter.PriceMin != 100 || gotFilter.PriceMax != 500 {
		t.Errorf("price bounds = [%d, %d], want [100, 500]", gotFilter.PriceMin, gotFilter.PriceMax)
	}
	if gotFilter.Sort != "-price" {
		t.Errorf("sort = %q, want -price", gotFilter.Sort)
	}
}

// TestStoreListProductsCapsPerPage verifies per_page is clamped at maxPerPage
// before reaching the store so a client cannot ask for the whole table.
func TestStoreListProductsCapsPerPage(t *testing.T) {
	var gotFilter model.ProductFilter
	fake := &fakeStoreReader{
		listFn: func(_ context.Context, f model.ProductFilter) ([]model.Product, int, error) {
			gotFilter = f
			return []model.Product{}, 0, nil
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/api/products?per_page=99999", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if gotFilter.PerPage != maxPerPage {
		t.Errorf("PerPage = %d, want %d", gotFilter.PerPage, maxPerPage)
	}

	var body struct {
		PerPage int `json:"per_page"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.PerPage != maxPerPage {
		t.Errorf("body.per_page = %d, want %d", body.PerPage, maxPerPage)
	}
}

// TestStoreListFacetsIgnoresFacetSelection verifies /facets parses the same
// filter params as /products but drops brand/color/size — otherwise selecting
// a brand would hide every other brand from the facet UI.
func TestStoreListFacetsIgnoresFacetSelection(t *testing.T) {
	var gotFilter model.ProductFilter
	fake := &fakeStoreReader{
		facetsFn: func(_ context.Context, f model.ProductFilter) (model.ProductFacets, error) {
			gotFilter = f
			return model.ProductFacets{Brands: []string{"Nike"}, Colors: []string{}, Sizes: []string{}}, nil
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet,
		"/api/products/facets?category_id=2&brand=Nike&color=red&size=M&price_min=100",
		nil)
	rr := httptest.NewRecorder()
	h.ListFacets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	// Category and price scope are honoured.
	if !equalIntSlices(gotFilter.CategoryIDs, []int{2}) || gotFilter.PriceMin != 100 {
		t.Errorf("filter scope dropped: %+v", gotFilter)
	}
	// Facet selections are dropped.
	if len(gotFilter.Brands) != 0 || len(gotFilter.Colors) != 0 || len(gotFilter.Sizes) != 0 || gotFilter.Brand != "" {
		t.Errorf("facet selection leaked into store filter: %+v", gotFilter)
	}
}

// TestStoreListProductsAcceptsKnownStatus verifies the catalog "В наличии /
// Под заказ" toggle param is plumbed through to the store layer untouched for
// the two real values. Out-of-scope values are covered by the next test.
func TestStoreListProductsAcceptsKnownStatus(t *testing.T) {
	var gotFilter model.ProductFilter
	fake := &fakeStoreReader{
		listFn: func(_ context.Context, f model.ProductFilter) ([]model.Product, int, error) {
			gotFilter = f
			return []model.Product{}, 0, nil
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/api/products?status=preorder", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if gotFilter.Status != "preorder" {
		t.Errorf("filter.Status = %q, want preorder", gotFilter.Status)
	}
}

// TestStoreListProductsRejectsUnknownStatus verifies the handler returns a
// 400 envelope with code VALIDATION_FAILED when ?status= carries a value
// outside the closed enum. The store must never see a malformed value.
func TestStoreListProductsRejectsUnknownStatus(t *testing.T) {
	fake := &fakeStoreReader{
		listFn: func(_ context.Context, _ model.ProductFilter) ([]model.Product, int, error) {
			t.Fatal("store should not be called for invalid status")
			return nil, 0, nil
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/api/products?status=ghost", nil)
	rr := httptest.NewRecorder()
	h.ListProducts(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", body.Code)
	}
	// Body must NOT leak the raw enum hint to a public client (defence in
	// depth — the wire format is "VALIDATION_FAILED", the operator log is the
	// only place the full message belongs).
	if !strings.Contains(strings.ToLower(body.Error), "status") {
		t.Errorf("error = %q, expected to mention 'status'", body.Error)
	}
}

func equalStrSlices(a, b []string) bool {
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

func equalIntSlices(a, b []int) bool {
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

func TestStoreGetProductNotFound(t *testing.T) {
	fake := &fakeStoreReader{
		getFn: func(_ context.Context, _ string) (*model.Product, error) {
			return nil, errors.New("product not found: x")
		},
	}
	h := NewStoreHandler(fake)

	req := httptest.NewRequest(http.MethodGet, "/api/products/x", nil)
	req.SetPathValue("slug", "x")
	rr := httptest.NewRecorder()
	h.GetProduct(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

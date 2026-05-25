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
	listFn func(ctx context.Context, f model.ProductFilter) ([]model.Product, int, error)
	getFn  func(ctx context.Context, slug string) (*model.Product, error)
	catsFn func(ctx context.Context) ([]model.Category, error)
}

func (f *fakeStoreReader) ListProducts(ctx context.Context, filter model.ProductFilter) ([]model.Product, int, error) {
	return f.listFn(ctx, filter)
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

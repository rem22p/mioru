package handler_test

import (
	"context"
	"net/http"
	"testing"

	"mioru/internal/model"
)

// facetProduct inserts an in-stock product with an explicit brand/color so a
// test can build a distinct-brand catalog. seedProduct hardcodes
// Brand:"TestBrand"/Color:"red", which is useless for facet assertions, so the
// distinct-brand tests go through the store directly.
func facetProduct(t *testing.T, e *env, slug, brand, color string) {
	t.Helper()
	_, err := e.st.CreateProduct(context.Background(), model.Product{
		Slug: slug, CategoryID: 1, Brand: brand, Name: "Test " + slug,
		Price: 500, Color: color, Status: "in_stock", InStock: true,
		StockQty: 5, CreatedBy: "test", Sizes: []string{"M"},
	})
	if err != nil {
		t.Fatalf("facetProduct %s: %v", slug, err)
	}
}

// facetsResponse mirrors model.ProductFacets' wire shape (json tags
// brands/colors/sizes) so the test depends on the public contract, not the Go
// field names.
type facetsResponse struct {
	Brands []string `json:"brands"`
	Colors []string `json:"colors"`
	Sizes  []string `json:"sizes"`
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// TestIntegrationFacetsHappy verifies the public facets endpoint enumerates the
// distinct brands/colors present in the catalog.
func TestIntegrationFacetsHappy(t *testing.T) {
	e := newEnv(t)
	facetProduct(t, e, "acme-1", "Acme", "red")
	facetProduct(t, e, "globex-1", "Globex", "blue")

	rr := e.do(t, http.HandlerFunc(e.storeH.ListFacets), http.MethodGet, "/api/products/facets", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListFacets: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp facetsResponse
	decode(t, rr, &resp)
	if !contains(resp.Brands, "Acme") || !contains(resp.Brands, "Globex") {
		t.Errorf("brands = %v, want both Acme and Globex", resp.Brands)
	}
	if len(resp.Colors) == 0 {
		t.Errorf("colors = %v, want non-empty", resp.Colors)
	}
}

// TestIntegrationFacetsDropsOwnSelection is the core invariant: selecting one
// brand must NOT hide the sibling brands from the facet list, otherwise a
// chip-style filter UI can never offer the other options. The handler drops the
// brand filter before querying facets, so picking brand=Acme must still return
// both Acme and Globex.
func TestIntegrationFacetsDropsOwnSelection(t *testing.T) {
	e := newEnv(t)
	facetProduct(t, e, "acme-1", "Acme", "red")
	facetProduct(t, e, "globex-1", "Globex", "blue")

	rr := e.do(t, http.HandlerFunc(e.storeH.ListFacets), http.MethodGet, "/api/products/facets?brand=Acme", reqOpts{})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListFacets?brand=Acme: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp facetsResponse
	decode(t, rr, &resp)
	if !contains(resp.Brands, "Acme") || !contains(resp.Brands, "Globex") {
		t.Errorf("brands = %v, want BOTH Acme and Globex (selecting a brand must not drop sibling brands)", resp.Brands)
	}
}

// TestIntegrationFacetsBadStatus confirms the facets endpoint rejects an unknown
// status value through the same parseProductFilter validation as ListProducts:
// an out-of-enum status is a 400 with VALIDATION_FAILED, never a silent "all".
func TestIntegrationFacetsBadStatus(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, http.HandlerFunc(e.storeH.ListFacets), http.MethodGet, "/api/products/facets?status=bogus", reqOpts{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("ListFacets?status=bogus: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

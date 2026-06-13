package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"mioru/internal/model"
)

// storeReader is the subset of the store consumed by the public storefront
// handlers. Defined here (where it is used) to keep the seam small and let tests
// supply a fake; *store.PostgresStore satisfies it.
type storeReader interface {
	ListProducts(ctx context.Context, filter model.ProductFilter) ([]model.Product, int, error)
	ListProductFacets(ctx context.Context, filter model.ProductFilter) (model.ProductFacets, error)
	GetProduct(ctx context.Context, slug string) (*model.Product, error)
	GetCategories(ctx context.Context) ([]model.Category, error)
}

// maxPerPage caps the page size a client may request. The store layer also
// caps internally (defence in depth) but the handler clamps first so the
// returned per_page in the body matches what actually shipped.
const maxPerPage = 100

// multiQuery returns all values for the given key, supporting both repeated
// (?brand=A&brand=B) and comma-separated (?brand=A,B) forms. Empty values are
// dropped so a stray comma can't smuggle in a "match-everything" filter.
func multiQuery(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// StoreHandler handles public (no-auth) storefront endpoints.
type StoreHandler struct {
	store storeReader
}

// NewStoreHandler creates a new StoreHandler.
func NewStoreHandler(s storeReader) *StoreHandler {
	return &StoreHandler{store: s}
}

// ListProducts handles GET /api/products
// Supports query params: category_id, search, brand, color, size, price_min,
// price_max, sort, page, per_page. brand/color/size accept repeated keys
// (?brand=A&brand=B) or comma-separated values (?brand=A,B).
func (h *StoreHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProductFilter(r)
	if err != nil {
		jsonErrorCode(w, err.Error(), http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	products, total, err := h.store.ListProducts(r.Context(), filter)
	if err != nil {
		log.Printf("ListProducts: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if products == nil {
		products = []model.Product{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"products": products,
		"total":    total,
		"page":     filter.Page,
		"per_page": filter.PerPage,
	})
}

// ListFacets handles GET /api/products/facets and returns the distinct
// brand/color/size values available within the given filter scope. The same
// query params as ListProducts are honoured (so a facet query for "category=X
// price>=100" returns only the brands/colors/sizes that produce non-empty
// results under those constraints), with one deliberate exception: brand,
// color, and size are *ignored* when building facets — otherwise selecting a
// brand would immediately hide every other brand from the UI.
func (h *StoreHandler) ListFacets(w http.ResponseWriter, r *http.Request) {
	filter, err := parseProductFilter(r)
	if err != nil {
		jsonErrorCode(w, err.Error(), http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	filter.Brand = ""
	filter.Brands = nil
	filter.Colors = nil
	filter.Sizes = nil
	// status is also kept (unlike brand/color/size) — it is a TOGGLE, not a
	// narrowing facet, and the user is expected to see products from the same
	// status bucket when computing facet counts. Dropping it would make the
	// size chart look broken when toggled to "Под заказ".

	facets, err := h.store.ListProductFacets(r.Context(), filter)
	if err != nil {
		log.Printf("ListFacets: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facets)
}

// parseProductFilter reads the storefront filter/sort/pagination params off
// the request, clamping per_page to maxPerPage so a client cannot ask the DB
// to ship a million rows. Returns a non-nil error when ?status= is set to a
// value outside the closed enum {in_stock, preorder, out_of_stock}.
func parseProductFilter(r *http.Request) (model.ProductFilter, error) {
	q := r.URL.Query()

	filter := model.ProductFilter{
		Page:    1,
		PerPage: 20,
	}

	for _, raw := range multiQuery(q["category_id"]) {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			filter.CategoryIDs = append(filter.CategoryIDs, v)
		}
	}
	filter.Search = strings.TrimSpace(q.Get("search"))
	filter.Brands = multiQuery(q["brand"])
	filter.Colors = multiQuery(q["color"])
	filter.Sizes = multiQuery(q["size"])
	if v, err := strconv.Atoi(q.Get("price_min")); err == nil && v > 0 {
		filter.PriceMin = v
	}
	if v, err := strconv.Atoi(q.Get("price_max")); err == nil && v > 0 {
		filter.PriceMax = v
	}
	// status is a closed enum; an unknown value would silently become "" (=
	// "all products") and confuse the catalog toggle. Reject it explicitly.
	if raw := q.Get("status"); raw != "" {
		switch raw {
		case "in_stock", "preorder", "out_of_stock":
			filter.Status = raw
		default:
			// Don't echo the raw user input or the full enum in a public
			// error message — the wire-format contract is "valid or
			// rejected", and a generic code is enough for the client
			// to surface. A leaked enum tells an attacker the exact
			// set to probe.
			return filter, fmt.Errorf("invalid status value")
		}
	}
	filter.Sort = q.Get("sort")
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		filter.Page = v
	}
	if v, err := strconv.Atoi(q.Get("per_page")); err == nil && v > 0 {
		filter.PerPage = v
	}
	if filter.PerPage > maxPerPage {
		filter.PerPage = maxPerPage
	}
	return filter, nil
}

// GetProduct handles GET /api/products/{slug}
// Returns the full product with sizes, size chart, and images.
func (h *StoreHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	p, err := h.store.GetProduct(r.Context(), slug)
	if err != nil {
		jsonError(w, "product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// ListCategories handles GET /api/categories
// Returns the category tree (not flat) for store navigation.
func (h *StoreHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.store.GetCategories(r.Context())
	if err != nil {
		log.Printf("ListCategories: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if cats == nil {
		cats = []model.Category{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

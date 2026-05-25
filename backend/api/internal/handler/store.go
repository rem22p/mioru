package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"mioru/internal/model"
	"mioru/internal/store"
)

// StoreHandler handles public (no-auth) storefront endpoints.
type StoreHandler struct {
	store *store.PostgresStore
}

// NewStoreHandler creates a new StoreHandler.
func NewStoreHandler(s *store.PostgresStore) *StoreHandler {
	return &StoreHandler{store: s}
}

// ListProducts handles GET /api/products
// Supports query params: category_id, search, brand, sort, page, per_page.
func (h *StoreHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := model.ProductFilter{
		Page:    1,
		PerPage: 20,
	}

	if v, err := strconv.Atoi(q.Get("category_id")); err == nil && v > 0 {
		filter.CategoryID = v
	}
	filter.Search = q.Get("search")
	filter.Brand = q.Get("brand")
	filter.Sort = q.Get("sort")
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		filter.Page = v
	}
	if v, err := strconv.Atoi(q.Get("per_page")); err == nil && v > 0 {
		filter.PerPage = v
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

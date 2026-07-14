package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"mioru/internal/middleware"
	"mioru/internal/model"
)

// productStore is the subset of the store consumed by the admin product
// handlers. Defined here (where it is used) to keep the seam small and let tests
// supply a fake; *store.PostgresStore satisfies it.
type productStore interface {
	ListProducts(ctx context.Context, filter model.ProductFilter) ([]model.Product, int, error)
	GetProduct(ctx context.Context, slug string) (*model.Product, error)
	CreateProduct(ctx context.Context, p model.Product) (int64, error)
	UpdateProduct(ctx context.Context, slug string, p model.Product) error
	DeleteProduct(ctx context.Context, slug string) error
	UpdateProductRanks(ctx context.Context, ranks map[int64]int) error
	GetCategories(ctx context.Context) ([]model.Category, error)
	GetCategoriesFlat(ctx context.Context) ([]model.Category, error)
}

// ProductHandler handles product CRUD and file uploads.
type ProductHandler struct {
	store     productStore
	uploadDir string
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(s productStore, uploadDir string) *ProductHandler {
	return &ProductHandler{store: s, uploadDir: uploadDir}
}

// List handles GET /api/admin/products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("admin ListProducts: %v", err)
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

// Create handles POST /api/admin/products (multipart form)
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "failed to parse form (too large or malformed)", http.StatusBadRequest)
		return
	}

	p, err := parseProductFromForm(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.CreatedBy = username

	// Upload images
	imageURLs, err := h.saveUploadedImages(r, "images")
	if err != nil {
		log.Printf("Create product saveUploadedImages: %v", err)
		jsonError(w, "failed to save images", http.StatusInternalServerError)
		return
	}
	for i, url := range imageURLs {
		p.Images = append(p.Images, model.ProductImage{URL: url, SortOrder: i})
	}

	id, err := h.store.CreateProduct(r.Context(), p)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			jsonError(w, "product with this slug already exists", http.StatusConflict)
			return
		}
		log.Printf("CreateProduct: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	created, err := h.store.GetProduct(r.Context(), p.Slug)
	if err != nil {
		log.Printf("Create product re-fetch: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"product": created,
	})
}

// Get handles GET /api/admin/products/{slug}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	p, err := h.store.GetProduct(r.Context(), slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		log.Printf("admin GetProduct: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// Update handles PUT /api/admin/products/{slug} (multipart form)
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "failed to parse form (too large or malformed)", http.StatusBadRequest)
		return
	}

	p, err := parseProductFromForm(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Upload new images
	imageURLs, err := h.saveUploadedImages(r, "images")
	if err != nil {
		log.Printf("Update product saveUploadedImages: %v", err)
		jsonError(w, "failed to save images", http.StatusInternalServerError)
		return
	}

	// Preserve existing images (sent as text fields from the frontend).
	// Defence-in-depth: cap, format whitelist, deterministic SortOrder.
	// Admin-only route (RequireAdmin), but still: preserved images used
	// to get SortOrder=0 (zero value) and were therefore shuffled to
	// the front by the later `len(p.Images) + i` block. Now they get
	// a deterministic sort based on their position in the form.
	if r.MultipartForm != nil {
		preserved := r.MultipartForm.Value["existing_images[]"]
		if len(preserved) > 20 {
			jsonError(w, "existing_images: max 20 entries", http.StatusBadRequest)
			return
		}
		for i, url := range preserved {
			if len(url) > 500 {
				jsonError(w, fmt.Sprintf("existing_images[%d]: url too long", i), http.StatusBadRequest)
				return
			}
			if !strings.HasPrefix(url, "/uploads/") && !strings.HasPrefix(url, "https://") {
				jsonError(w, fmt.Sprintf("existing_images[%d]: must start with /uploads/ or https://", i), http.StatusBadRequest)
				return
			}
			p.Images = append(p.Images, model.ProductImage{URL: url, SortOrder: i})
		}
	}

	for i, url := range imageURLs {
		p.Images = append(p.Images, model.ProductImage{URL: url, SortOrder: len(p.Images) + i})
	}

	if err := h.store.UpdateProduct(r.Context(), slug, p); err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			jsonError(w, "product with this slug already exists", http.StatusConflict)
			return
		}
		log.Printf("UpdateProduct: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	updated, err := h.store.GetProduct(r.Context(), p.Slug)
	if err != nil {
		log.Printf("Update product re-fetch: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// Delete handles DELETE /api/admin/products/{slug}
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteProduct(r.Context(), slug); err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		log.Printf("DeleteProduct: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// Categories handles GET /api/admin/categories
// Use ?flat=1 to get a flat list with parent_id (for form dropdowns).
func (h *ProductHandler) Categories(w http.ResponseWriter, r *http.Request) {
	flat := r.URL.Query().Get("flat") == "1"

	var cats []model.Category
	var err error
	if flat {
		cats, err = h.store.GetCategoriesFlat(r.Context())
	} else {
		cats, err = h.store.GetCategories(r.Context())
	}
	if err != nil {
		log.Printf("admin Categories: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if cats == nil {
		cats = []model.Category{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

// UpdateRanks handles PUT /api/admin/products/rank
func (h *ProductHandler) UpdateRanks(w http.ResponseWriter, r *http.Request) {
	var entries []struct {
		ID   int64 `json:"id"`
		Rank int   `json:"rank"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	ranks := make(map[int64]int, len(entries))
	for _, e := range entries {
		ranks[e.ID] = e.Rank
	}
	if err := h.store.UpdateProductRanks(r.Context(), ranks); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

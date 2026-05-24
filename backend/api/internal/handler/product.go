package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
)

// ProductHandler handles product CRUD and file uploads.
type ProductHandler struct {
	store     *store.SQLiteStore
	uploadDir string
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(s *store.SQLiteStore, uploadDir string) *ProductHandler {
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

	products, total, err := h.store.ListProducts(filter)
	if err != nil {
		jsonError(w, "internal error: "+err.Error(), http.StatusInternalServerError)
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

// Create handles POST /api/admin/products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)

	var p model.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	p.Slug = strings.TrimSpace(p.Slug)
	if p.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	if p.CategoryID <= 0 {
		jsonError(w, "category_id is required", http.StatusBadRequest)
		return
	}

	p.CreatedBy = username

	id, err := h.store.CreateProduct(p)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonError(w, "product with this slug already exists", http.StatusConflict)
			return
		}
		jsonError(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the created product to return full data
	created, err := h.store.GetProduct(p.Slug)
	if err != nil {
		jsonError(w, "created but failed to fetch: "+err.Error(), http.StatusInternalServerError)
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

	p, err := h.store.GetProduct(slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		jsonError(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// Update handles PUT /api/admin/products/{slug}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}

	var p model.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	p.Slug = strings.TrimSpace(p.Slug)
	if p.Slug == "" {
		jsonError(w, "slug is required", http.StatusBadRequest)
		return
	}
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	if p.CategoryID <= 0 {
		jsonError(w, "category_id is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateProduct(slug, p); err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonError(w, "product with this slug already exists", http.StatusConflict)
			return
		}
		jsonError(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated product
	updated, err := h.store.GetProduct(p.Slug)
	if err != nil {
		jsonError(w, "updated but failed to fetch: "+err.Error(), http.StatusInternalServerError)
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

	if err := h.store.DeleteProduct(slug); err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, "product not found", http.StatusNotFound)
			return
		}
		jsonError(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// Categories handles GET /api/admin/categories
func (h *ProductHandler) Categories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.store.GetCategories()
	if err != nil {
		jsonError(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if cats == nil {
		cats = []model.Category{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

// Upload handles POST /api/admin/upload
// Accepts a multipart file upload and saves it to the upload directory.
func (h *ProductHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// Limit upload size to 10 MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "file too large (max 10MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExt := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true,
	}
	if !allowedExt[ext] {
		jsonError(w, "only image files allowed (jpg, jpeg, png, gif, webp, svg)", http.StatusBadRequest)
		return
	}

	// Generate unique filename to avoid collisions
	safeName := uniquePrefix() + ext

	// Ensure upload directory exists
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		jsonError(w, "internal error: cannot create upload dir", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(h.uploadDir, safeName)
	dest, err := os.Create(destPath)
	if err != nil {
		jsonError(w, "internal error: cannot create file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(destPath)
		jsonError(w, "internal error: cannot write file", http.StatusInternalServerError)
		return
	}

	url := "/uploads/" + safeName
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// uniquePrefix generates a unique prefix for uploaded filenames.
func uniquePrefix() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
}

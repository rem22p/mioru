package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	store     *store.PostgresStore
	uploadDir string
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(s *store.PostgresStore, uploadDir string) *ProductHandler {
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
	for i, url := range imageURLs {
		p.Images = append(p.Images, model.ProductImage{URL: url, SortOrder: i})
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

	// Validate the extension, then sniff the real bytes — never trust the
	// client-supplied filename or Content-Type (allowlist + sniff in image.go;
	// SVG excluded as a stored-XSS vector). Rewind before saving.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExt(ext) {
		jsonError(w, "only image files allowed (jpg, jpeg, png, gif, webp)", http.StatusBadRequest)
		return
	}
	if err := validateImageContent(file); err != nil {
		jsonError(w, "file content is not a supported image", http.StatusBadRequest)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
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

// parseProductFromForm extracts a Product from multipart form values.
func parseProductFromForm(r *http.Request) (model.Product, error) {
	p := model.Product{
		Slug:        strings.TrimSpace(r.FormValue("slug")),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: strings.TrimSpace(r.FormValue("description")),
		Brand:       strings.TrimSpace(r.FormValue("brand")),
		Color:       strings.TrimSpace(r.FormValue("color")),
		Model:       strings.TrimSpace(r.FormValue("model")),
		Fit:         strings.TrimSpace(r.FormValue("fit")),
		Material:    strings.TrimSpace(r.FormValue("material")),
	}

	if v, err := strconv.Atoi(r.FormValue("price")); err == nil {
		p.Price = v
	}
	if v, err := strconv.Atoi(r.FormValue("xp_reward")); err == nil {
		p.XPReward = v
	}
	if v, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64); err == nil {
		p.CategoryID = v
	}
	p.InStock = r.FormValue("in_stock") == "true" || r.FormValue("in_stock") == "1"

	p.Status = strings.TrimSpace(r.FormValue("status"))
	if p.Status == "" {
		if p.InStock {
			p.Status = "in_stock"
		} else {
			p.Status = "none"
		}
	}
	if v, err := strconv.Atoi(r.FormValue("stock_quantity")); err == nil {
		p.StockQty = v
	}

	if p.Slug == "" {
		return p, fmt.Errorf("slug is required")
	}
	if p.Name == "" {
		return p, fmt.Errorf("name is required")
	}
	if p.CategoryID <= 0 {
		return p, fmt.Errorf("category_id is required")
	}

	// Sizes
	if r.Form["sizes[]"] != nil {
		p.Sizes = r.Form["sizes[]"]
	} else if r.Form["sizes"] != nil {
		p.Sizes = r.Form["sizes"]
	}

	// Care instructions
	if r.Form["care[]"] != nil {
		p.Care = r.Form["care[]"]
	}

	// Size chart — parse indexed fields like size_chart[0][label], size_chart[0][chest], etc.
	chartMap := make(map[int]*model.SizeChartRow)
	for key, values := range r.Form {
		if !strings.HasPrefix(key, "size_chart[") {
			continue
		}
		// Parse "size_chart[0][label]" → index=0, field="label"
		rest := strings.TrimPrefix(key, "size_chart[")
		closeBracket := strings.Index(rest, "]")
		if closeBracket < 0 {
			continue
		}
		idx, err := strconv.Atoi(rest[:closeBracket])
		if err != nil {
			continue
		}
		fieldPart := rest[closeBracket+1:]
		if !strings.HasPrefix(fieldPart, "[") || !strings.HasSuffix(fieldPart, "]") {
			continue
		}
		field := fieldPart[1 : len(fieldPart)-1]
		if len(values) == 0 {
			continue
		}
		val := values[0]

		if chartMap[idx] == nil {
			chartMap[idx] = &model.SizeChartRow{}
		}
		row := chartMap[idx]

		switch field {
		case "label":
			row.Label = val
		case "chest":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Chest = &f
			}
		case "waist":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Waist = &f
			}
		case "hips":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Hips = &f
			}
		case "length":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Length = &f
			}
		case "foot_length":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.FootLength = &f
			}
		case "wrist":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				row.Wrist = &f
			}
		}
	}
	for i := 0; ; i++ {
		if row, ok := chartMap[i]; ok {
			p.SizeChart = append(p.SizeChart, *row)
		} else {
			break
		}
	}

	return p, nil
}

// saveUploadedImages saves all files from a multipart form field and returns their URLs.
func (h *ProductHandler) saveUploadedImages(r *http.Request, fieldName string) ([]string, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil
	}

	files := r.MultipartForm.File[fieldName]
	if len(files) == 0 {
		return nil, nil
	}

	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		return nil, err
	}

	var urls []string
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		if !allowedImageExt(ext) {
			log.Printf("saveUploadedImages: %q skipped: extension %q not allowed", fh.Filename, ext)
			continue
		}

		file, err := fh.Open()
		if err != nil {
			log.Printf("saveUploadedImages: open %q: %v", fh.Filename, err)
			continue
		}

		// Sniff the real bytes (blocks SVG/HTML renamed to .png), then rewind.
		if err := validateImageContent(file); err != nil {
			log.Printf("saveUploadedImages: %q rejected: %v", fh.Filename, err)
			file.Close()
			continue
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			log.Printf("saveUploadedImages: seek %q: %v", fh.Filename, err)
			file.Close()
			continue
		}

		safeName := uniquePrefix() + ext
		destPath := filepath.Join(h.uploadDir, safeName)
		dest, err := os.Create(destPath)
		if err != nil {
			log.Printf("saveUploadedImages: create %q: %v", destPath, err)
			file.Close()
			continue
		}

		if _, err := io.Copy(dest, file); err != nil {
			// Don't return a URL for a partially written file; clean it up.
			log.Printf("saveUploadedImages: write %q: %v", destPath, err)
			dest.Close()
			file.Close()
			os.Remove(destPath)
			continue
		}
		dest.Close()
		file.Close()

		urls = append(urls, "/uploads/"+safeName)
	}

	return urls, nil
}

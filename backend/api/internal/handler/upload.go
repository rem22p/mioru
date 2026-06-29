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
	"strings"
	"time"
)

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
		jsonError(w, "only image files allowed (png, jpg, webp)", http.StatusBadRequest)
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

	// Generate thumbnail (400×300).
	thumbName := "thumb_" + safeName
	thumbPath := filepath.Join(h.uploadDir, thumbName)
	if err := generateThumbnail(destPath, thumbPath, 400, 300); err != nil {
		log.Printf("Upload: thumbnail generation failed for %s: %v", safeName, err)
		// Thumbnail is best-effort — original is already saved.
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

		// Generate thumbnail (400×300).
		thumbName := "thumb_" + safeName
		thumbPath := filepath.Join(h.uploadDir, thumbName)
		if err := generateThumbnail(destPath, thumbPath, 400, 300); err != nil {
			log.Printf("saveUploadedImages: thumbnail failed for %s: %v", safeName, err)
		}

		urls = append(urls, "/uploads/"+safeName)
	}

	return urls, nil
}

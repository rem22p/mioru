package handler

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"mioru/internal/thumbnail"
)

func TestGenerateThumbnail(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst := filepath.Join(dir, "thumb.png")

	// Create a 100×80 PNG.
	createTestPNG(t, src, 100, 80)

	if err := generateThumbnail(src, dst, 40, 40); err != nil {
		t.Fatalf("generateThumbnail: %v", err)
	}

	// Verify the output exists and has correct dimensions.
	f, err := os.Open(dst)
	if err != nil {
		t.Fatalf("open thumbnail: %v", err)
	}
	defer f.Close()

	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}

	// 100×80 in a 40×40 box → width-limited: 40×32 (scale = 0.4)
	if cfg.Width != 40 {
		t.Errorf("width = %d, want 40", cfg.Width)
	}
	if cfg.Height != 32 {
		t.Errorf("height = %d, want 32", cfg.Height)
	}
}

func TestGenerateThumbnailRejectsTooLarge(t *testing.T) {
	saved := thumbnail.MaxPixels
	thumbnail.MaxPixels = 100 // tiny limit for testing
	defer func() { thumbnail.MaxPixels = saved }()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst := filepath.Join(dir, "thumb.png")

	// 20×20 = 400 pixels > 100 → should be rejected.
	createTestPNG(t, src, 20, 20)

	err := generateThumbnail(src, dst, 10, 10)
	if err == nil {
		t.Fatal("expected rejection for image exceeding pixel limit, got nil")
	}
}

func TestGenerateThumbnailSourceNotFound(t *testing.T) {
	err := generateThumbnail("/nonexistent/path.png", "/tmp/out.png", 100, 100)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

// createTestPNG writes a solid-color PNG of the given dimensions to path.
func createTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test PNG: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
}

package thumbnail_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"mioru/internal/thumbnail"
)

func TestGenerateFitsAspectRatio(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst := filepath.Join(dir, "thumb.png")

	// 100×80 PNG → fits a 40×40 box width-limited → 40×32.
	createTestPNG(t, src, 100, 80)

	if err := thumbnail.Generate(src, dst, 40, 40); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	cfg, err := decodeConfig(t, dst)
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Width != 40 {
		t.Errorf("width = %d, want 40", cfg.Width)
	}
	if cfg.Height != 32 {
		t.Errorf("height = %d, want 32", cfg.Height)
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	// regen-thumbs and the live upload handler share this
	// function, so the output for a given source must be
	// byte-identical across calls — otherwise an admin re-running
	// regen-thumbs would invalidate every cached thumbnail.
	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst1 := filepath.Join(dir, "thumb1.png")
	dst2 := filepath.Join(dir, "thumb2.png")
	createTestPNG(t, src, 137, 91)

	if err := thumbnail.Generate(src, dst1, 400, 300); err != nil {
		t.Fatalf("Generate #1: %v", err)
	}
	if err := thumbnail.Generate(src, dst2, 400, 300); err != nil {
		t.Fatalf("Generate #2: %v", err)
	}

	a, err := os.ReadFile(dst1)
	if err != nil {
		t.Fatalf("read dst1: %v", err)
	}
	b, err := os.ReadFile(dst2)
	if err != nil {
		t.Fatalf("read dst2: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("two Generate calls produced different bytes for the same source:\n  a=%x\n  b=%x", a[:min(64, len(a))], b[:min(64, len(b))])
	}
}

func TestGenerateRejectsTooLarge(t *testing.T) {
	saved := thumbnail.MaxPixels
	thumbnail.MaxPixels = 100
	defer func() { thumbnail.MaxPixels = saved }()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.png")
	dst := filepath.Join(dir, "thumb.png")
	createTestPNG(t, src, 20, 20) // 400 px > 100

	if err := thumbnail.Generate(src, dst, 10, 10); err == nil {
		t.Fatal("expected rejection for image exceeding MaxPixels, got nil")
	}
}

func TestGenerateSourceNotFound(t *testing.T) {
	if err := thumbnail.Generate("/nonexistent/path.png", filepath.Join(t.TempDir(), "out.png"), 100, 100); err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

// TestGenerateDecodesJPEG: an image/jpeg source decodes correctly via the
// image.Decode path (registers via _ "image/jpeg" in the package). Output
// must still be PNG, and dimensions must match the JPEG source aspect ratio.
func TestGenerateDecodesJPEG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.jpg")
	dst := filepath.Join(dir, "thumb.png")

	createTestJPEG(t, src, 200, 100) // 2:1 aspect

	if err := thumbnail.Generate(src, dst, 50, 50); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	cfg, err := decodeConfig(t, dst)
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	// 200×100 in a 50×50 box → width-limited: 50×25.
	if cfg.Width != 50 {
		t.Errorf("width = %d, want 50", cfg.Width)
	}
	if cfg.Height != 25 {
		t.Errorf("height = %d, want 25", cfg.Height)
	}
}

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

func createTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill with a non-zero color so JPEG's lossy encoder still produces
	// readable bytes (all-zero RGBA gets compressed to almost nothing).
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test JPEG: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode test JPEG: %v", err)
	}
}

func decodeConfig(t *testing.T, path string) (image.Config, error) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()
	return png.DecodeConfig(f)
}

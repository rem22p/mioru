package handler

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// maxPixels is the maximum allowed image size in pixels (width × height)
// for decompression-bomb protection. 50 megapixels is generous for a
// product photo (e.g. 10000×5000) while blocking degenerate inputs that
// claim tiny file size but massive pixel dimensions.
// Package-level var so tests can temporarily lower the limit.
var maxPixels = 50_000_000

// generateThumbnail reads a PNG from srcPath, resizes it to the target
// dimensions maintaining aspect ratio within the box (letterbox), and writes
// the result to dstPath. Both width and height act as maximum bounds — the
// image is scaled to fit while preserving its proportions.
func generateThumbnail(srcPath, dstPath string, maxWidth, maxHeight int) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Decompression-bomb guard: read dimensions without allocating pixels.
	cfg, err := png.DecodeConfig(src)
	if err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if cfg.Width*cfg.Height > maxPixels {
		return fmt.Errorf("image too large: %d×%d (max %d pixels)", cfg.Width, cfg.Height, maxPixels)
	}
	// Rewind to start for full decode.
	if _, err := src.Seek(0, 0); err != nil {
		return fmt.Errorf("seek: %w", err)
	}

	img, err := png.Decode(src)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	// Compute scale to fit within maxWidth × maxHeight.
	scale := float64(maxWidth) / float64(srcW)
	if hScale := float64(maxHeight) / float64(srcH); hScale < scale {
		scale = hScale
	}
	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if err := png.Encode(out, dst); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	// Close must be checked — a flush error means truncated file.
	return out.Close()
}

// Package thumbnail is the single source of truth for the
// PNG-thumbnail generation pipeline used both by the live upload
// handler (internal/handler/upload.go, customer.go) and by the
// standalone regen-thumbs migration tool (cmd/regen-thumbs).
//
// Both call sites previously carried byte-for-byte copies of the
// same algorithm; the regen-thumbs comment even warned "Any drift
// here is a bug". The shared package makes the algorithm a single
// source of truth so drift becomes structurally impossible.
//
// Behaviour:
//   - Reads a PNG from srcPath, validates against the decompression-
//     bomb guard (MaxPixels), decodes, fits-with-aspect into the
//     (maxWidth, maxHeight) box via Catmull-Rom scaling on a zeroed
//     RGBA canvas (preserves alpha), encodes back to PNG at dstPath.
//   - Returns wrapped errors so callers can distinguish failure modes
//     (bad config vs decode vs encode vs io).
package thumbnail

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

// MaxPixels is the maximum allowed source image size in pixels
// (width × height) for decompression-bomb protection. 50 megapixels
// is generous for a product photo (e.g. 10000×5000) while blocking
// degenerate inputs that claim tiny file size but massive pixel
// dimensions.
//
// Package-level var so tests can temporarily lower the limit. Both
// the handler test and the regen-thumbs migration tool rely on the
// same number so they cannot drift apart silently.
var MaxPixels = 50_000_000

// Generate reads a PNG from srcPath, resizes it to fit within the
// (maxWidth, maxHeight) box while preserving aspect ratio, and
// writes the result to dstPath. Both width and height are maximum
// bounds — the image is scaled to fit, never cropped.
//
// Errors are wrapped so the caller can errors.Is / errors.As on
// the underlying cause if needed.
func Generate(srcPath, dstPath string, maxWidth, maxHeight int) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Decompression-bomb guard: read dimensions without allocating
	// pixels, then rewind for full decode.
	cfg, err := png.DecodeConfig(src)
	if err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	if cfg.Width*cfg.Height > MaxPixels {
		return fmt.Errorf("image too large: %d×%d (max %d pixels)", cfg.Width, cfg.Height, MaxPixels)
	}
	if _, err := src.Seek(0, 0); err != nil {
		return fmt.Errorf("seek: %w", err)
	}

	img, err := png.Decode(src)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()

	scale := float64(maxWidth) / float64(srcW)
	if hScale := float64(maxHeight) / float64(srcH); hScale < scale {
		scale = hScale
	}
	dstW := int(float64(srcW) * scale)
	dstH := int(float64(srcH) * scale)

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

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
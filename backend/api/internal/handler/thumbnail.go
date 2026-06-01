package handler

import (
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

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

	img, err := png.Decode(src)
	if err != nil {
		return err
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

	return png.Encode(out, dst)
}

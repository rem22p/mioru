package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"
)

// errUnsupportedImage is returned when an uploaded file's real bytes do not
// sniff as a supported raster image.
var errUnsupportedImage = errors.New("file content is not a supported image")

// allowedImageExts is the upload extension allowlist. SVG is intentionally
// excluded: it is an XML document that can embed <script>, so a stored SVG
// becomes stored XSS.
//
// JPEG and WebP are the two real-world image formats we also accept:
//   - iPhone Safari (iOS 14+) submits camera photos as image/jpeg even when
//     the device still records them as HEIC internally — but some browsers
//     and ancient iOS versions still surface .heic with image/heic, which the
//     stdlib can't decode, so HEIC is intentionally not in this list (the
//     browser would 400 on the corresponding content-type sniff).
//   - WebP is widely supported by the golang.org/x/image/webp decoder; admins
//     often export product photos as WebP for the smaller payload.
var allowedImageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

// allowedImageMIMEs is the set of content types accepted after sniffing the real
// file bytes (raster images only). Matched against http.DetectContentType, which
// only recognises the formats each Go decoder registers — so JPEG and PNG come
// back as image/jpeg / image/png; WebP comes back as image/webp when the import
// side-effect of golang.org/x/image/webp has run.
//
// Adding a decoder (e.g. for HEIC) means adding its MIME here AND to the
// thumbnail pipeline so the saved file can be decoded for thumbnail generation.
var allowedImageMIMEs = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

// allowedImageExt reports whether ext (with leading dot) is an accepted image
// extension. Case-insensitive.
func allowedImageExt(ext string) bool {
	return allowedImageExts[strings.ToLower(ext)]
}

// validateImageContent sniffs the first bytes of r and returns errUnsupportedImage
// unless they identify a supported raster image. It never trusts the client
// filename or Content-Type, so it blocks SVG/HTML payloads renamed to .png.
// It consumes bytes from r; callers reading the file afterwards must seek it
// back to the start before saving.
func validateImageContent(r io.Reader) error {
	sniff := make([]byte, 512)
	n, err := io.ReadFull(r, sniff)
	// Short files are fine — the magic bytes live in the first few bytes — so
	// only a genuine read error (not EOF / unexpected EOF) is fatal here.
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err
	}
	if !allowedImageMIMEs[http.DetectContentType(sniff[:n])] {
		return errUnsupportedImage
	}
	return nil
}

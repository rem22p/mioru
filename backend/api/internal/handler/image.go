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
var allowedImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

// allowedImageMIMEs is the set of content types accepted after sniffing the real
// file bytes (raster images only).
var allowedImageMIMEs = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
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

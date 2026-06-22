// Package handler keeps a thin re-export of the shared thumbnail
// generator so the handler's existing call sites
// (customer.go, upload.go) continue to compile without touching
// every reference. New code should call thumbnail.Generate
// directly; this file is a backward-compat shim and will be
// deleted once the handler tests move into the thumbnail package.
package handler

import "mioru/internal/thumbnail"

// generateThumbnail is a thin wrapper around thumbnail.Generate
// preserving the original handler-internal call signature.
func generateThumbnail(srcPath, dstPath string, maxWidth, maxHeight int) error {
	return thumbnail.Generate(srcPath, dstPath, maxWidth, maxHeight)
}
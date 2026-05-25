package handler

import (
	"bytes"
	"strings"
	"testing"
)

func TestAllowedImageExt(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".jpg", true},
		{".jpeg", true},
		{".png", true},
		{".gif", true},
		{".webp", true},
		{".PNG", true}, // case-insensitive
		{".svg", false},
		{".svg ", false},
		{".html", false},
		{".exe", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := allowedImageExt(tt.ext); got != tt.want {
			t.Errorf("allowedImageExt(%q) = %v, want %v", tt.ext, got, tt.want)
		}
	}
}

func TestValidateImageContent(t *testing.T) {
	// Real magic-byte prefixes; the rest of the file is irrelevant to sniffing.
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{"png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR"), false},
		{"jpeg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01"), false},
		{"gif", []byte("GIF89a\x01\x00\x01\x00"), false},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 16)...), false},
		{"svg with script", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), true},
		{"xml svg", []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`), true},
		{"html", []byte("<!DOCTYPE html><html><body>x</body></html>"), true},
		{"plain text", []byte("just some text, not an image at all"), true},
		{"empty", []byte{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateImageContent(bytes.NewReader(tt.data))
			if tt.wantErr && err == nil {
				t.Errorf("validateImageContent(%s) = nil, want error", tt.name)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateImageContent(%s) = %v, want nil", tt.name, err)
			}
		})
	}
}

// TestValidateImageContentRenamedSVG is the core security case: an SVG payload a
// caller might pass with a .png filename must still be rejected on content.
func TestValidateImageContentRenamedSVG(t *testing.T) {
	payload := `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(document.domain)"/>`
	if err := validateImageContent(strings.NewReader(payload)); err == nil {
		t.Fatal("SVG payload accepted as image — stored XSS vector")
	}
}

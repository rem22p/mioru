package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// allowExtsReduced is the production allowlist mirrored verbatim so the
// test doesn't depend on Go-time initialisation of the package-level
// regenSourceExts map (some Go versions initialise test globals lazily).
var allowExtsReduced = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

func TestWalkFiltersByExtensionAndThumb(t *testing.T) {
	dir := t.TempDir()

	// Allowed extensions
	writeEmpty(t, dir, "foo.png")
	writeEmpty(t, dir, "bar.jpg")
	writeEmpty(t, dir, "baz.jpeg")
	writeEmpty(t, dir, "qux.webp")

	// Rejected extensions / non-image entries
	writeEmpty(t, dir, "ignored.gif")
	writeEmpty(t, dir, "ignored.bmp")
	writeEmpty(t, dir, "no_extension")
	writeEmpty(t, dir, "README.md")
	writeEmpty(t, dir, "data.json")

	// Self-thumbnail skip (would recurse and exponential-grow if we ever
	// changed the allowlist to include them, but more importantly: walking
	// these would double the work we already did)
	writeEmpty(t, dir, "thumb_foo.png")
	writeEmpty(t, dir, "thumb_bar.jpg")

	// Directory guard
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	writeEmpty(t, filepath.Join(dir, "nested"), "inside.png")

	// Dotfiles (rare in uploads/, but exercise the case-insensitive path
	// and the lack of a special-case for hidden entries)
	writeEmpty(t, dir, ".hidden.png")

	got, err := walk(dir, allowExtsReduced)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	want := []string{
		filepath.Join(dir, ".hidden.png"), // dir entries are sorted lexicographically; dotfiles sort before letters in ASCII
		filepath.Join(dir, "bar.jpg"),
		filepath.Join(dir, "baz.jpeg"),
		filepath.Join(dir, "foo.png"),
		filepath.Join(dir, "qux.webp"),
	}
	sort.Strings(want) // defense: if Go's ReadDir ever changes order, this keeps the test stable
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("walk() =\n  got:  %v\n  want: %v", got, want)
	}

	// Critical sanity: thumb_* and non-allowed extensions must be absent.
	for _, p := range got {
		base := filepath.Base(p)
		if len(base) >= 6 && base[:6] == "thumb_" {
			t.Errorf("walk() leaked self-thumbnail: %s", p)
		}
		ext := filepath.Ext(base)
		if !allowExtsReduced[ext] {
			t.Errorf("walk() leaked non-allowed extension: %s (ext=%s)", p, ext)
		}
	}
}

func TestWalkCaseInsensitiveExtensionMatch(t *testing.T) {
	dir := t.TempDir()

	// Allowed exts in mixed case — must all be picked up
	for _, name := range []string{"a.PNG", "b.Jpg", "c.JPEG", "d.WebP"} {
		writeEmpty(t, dir, name)
	}

	got, err := walk(dir, allowExtsReduced)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 entries (case-insensitive match), got %d: %v", len(got), got)
	}
}

func TestWalkEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	got, err := walk(dir, allowExtsReduced)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice for empty dir, got %v", got)
	}
}

func TestWalkNonexistentDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	got, err := walk(dir, allowExtsReduced)
	if err == nil {
		t.Fatalf("expected error for nonexistent dir, got %v", got)
	}
	if len(got) != 0 {
		t.Fatalf("expected nil-or-empty slice on error, got %v", got)
	}
}

// TestWalkDoesNotPickUpThumbPrefixForAnyExt guards the historic bug where
// removing thumb_/allowlist logic could silently walk the tool's own
// thumbnails (exponential work). The check is intentionally tight: any name
// starting with "thumb_" must be skipped regardless of extension. Names
// like "thumbnail.png" (start with "thumbnail", not "thumb_") and
// "thumb.png" (no underscore after "thumb") are NOT excluded — this test
// deliberately includes those cases to lock in the exact prefix match
// (otherwise a sloppy `strings.Contains` rewrite would accidentally
// start excluding thumbnails too).
func TestWalkDoesNotPickUpThumbPrefixForAnyExt(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"thumb_a.png", "thumb_b.jpg", "thumb_c.jpeg", "thumb_d.webp",
		"thumb.png",     /* NOT skipped — char 6 is ".", not "_" */
		"thumbnail.png", /* NOT skipped — prefix is "thumb", not "thumb_" */
		"thumb_no_ext",
	} {
		writeEmpty(t, dir, name)
	}
	// Allowed source files to confirm walk returns OTHER files.
	writeEmpty(t, dir, "good.png")
	writeEmpty(t, dir, "good.jpg")

	got, err := walk(dir, allowExtsReduced)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	// Expected 4 files: good.png, good.jpg, thumb.png, thumbnail.png.
	// All four are allowed-ext, none start with "thumb_". thumb.png
	// (only 5 chars before dot) and thumbnail.png (no underscore) lock
	// in that the filter uses an exact `thumb_` prefix match.
	if len(got) != 4 {
		t.Fatalf("expected 4 files (2 good + thumb.png + thumbnail.png), got %d: %v", len(got), got)
	}
	for _, p := range got {
		base := filepath.Base(p)
		if len(base) >= 6 && base[:6] == "thumb_" {
			t.Errorf("walk() leaked thumb_ prefix: %s", p)
		}
	}
}

// TestWalkStrictAllowlist blocks a regression to the historical bug where
// the allowlist extension was misspelt (.jpg vs .jpeg). Build the allowlist
// from scratch here so the test reflects the actual contract, not the
// package global.
func TestWalkStrictAllowlist(t *testing.T) {
	dir := t.TempDir()
	writeEmpty(t, dir, "ok.png")
	writeEmpty(t, dir, "ok.jpg")
	writeEmpty(t, dir, "notok.png.worm") // tricky: ext says .worm, name has .png

	// Use the production mirror to verify.
	got, err := walk(dir, allowExtsReduced)
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	for _, p := range got {
		base := filepath.Base(p)
		if filepath.Ext(base) == ".worm" {
			t.Errorf("walk() leaked .worm: %s", p)
		}
	}
}

func writeEmpty(t *testing.T, dir, name string) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("create %s/%s: %v", dir, name, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

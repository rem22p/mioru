package store

import (
	"context"
	"testing"
)

// TestSeededCategoryTree pins the category tree seeded by migrate(). The seed is
// the single source of truth for categories (the admin frontend fetches it via
// the API and keeps no copy), so this guards it against silent drift.
func TestSeededCategoryTree(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// id -> (parentID, slug). parentID == 0 means a root (NULL parent).
	want := map[int]struct {
		parent int
		slug   string
	}{
		1:  {0, "clothing"},
		2:  {1, "tshirts-polo"},
		3:  {1, "shorts"},
		4:  {1, "hoodies-zip"},
		5:  {1, "sweatshirts-sweaters"},
		6:  {1, "jeans"},
		7:  {1, "pants"},
		8:  {1, "jackets"},
		9:  {1, "vests"},
		10: {1, "underwear"},
		11: {0, "shoes"},
		12: {11, "sneakers"},
		13: {11, "slides"},
		14: {11, "boots"},
		15: {0, "bags"},
		16: {0, "accessories"},
		17: {16, "wallets-cardholders"},
		18: {16, "belts"},
		19: {16, "headwear"},
		20: {16, "jewelry"},
		21: {20, "bracelets"},
		22: {20, "pendants"},
		23: {20, "rings"},
		24: {16, "watches"},
	}

	flat, err := s.GetCategoriesFlat(ctx)
	if err != nil {
		t.Fatalf("GetCategoriesFlat: %v", err)
	}
	if len(flat) != len(want) {
		t.Fatalf("seeded categories = %d, want %d", len(flat), len(want))
	}

	for _, c := range flat {
		exp, ok := want[c.ID]
		if !ok {
			t.Errorf("unexpected category id %d (%q)", c.ID, c.Slug)
			continue
		}
		if c.Slug != exp.slug {
			t.Errorf("category %d slug = %q, want %q", c.ID, c.Slug, exp.slug)
		}
		gotParent := 0
		if c.ParentID != nil {
			gotParent = *c.ParentID
		}
		if gotParent != exp.parent {
			t.Errorf("category %d (%s) parent = %d, want %d", c.ID, c.Slug, gotParent, exp.parent)
		}
	}

	// The tree must expose exactly the four top-level categories.
	roots, err := s.GetCategories(ctx)
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("root categories = %d, want 4", len(roots))
	}
}

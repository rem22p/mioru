package store

import (
	"context"
	"testing"

	"mioru/internal/model"
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

	// Category 15 (Bags / Сумки) must include "size" in its criteria so the
	// admin product form shows the size picker and clients can select a size
	// before checkout (migration 020_bags_size_criteria.sql).
	for _, c := range flat {
		if c.ID == 15 {
			found := false
			for _, crit := range c.Criteria {
				if crit == "size" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("category 15 (bags) criteria does not include 'size'; migration 020 may not have been applied")
			}
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

// TestCategoryCoverImage verifies the cover_image subquery:
//   - category with products → cover_image is the URL of the max-stock product's
//     first image (by sort_order)
//   - empty category → cover_image is nil
//   - child-category products contribute to parent cover (the subquery checks
//     both p.category_id = c.id AND p.category_id IN children)
func TestCategoryCoverImage(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	// Category 3 = "Шорты" (shorts), a leaf under "Одежда" (clothing, id=1).
	// Category 17 = "Кошельки/кардхолдеры" (wallets-cardholders), leaf under "Аксессуары".

	// Insert two products into category 3 with images. The one with higher
	// stock_quantity should become the cover.
	mustCreateProduct(t, s, model.Product{
		Slug: "shorts-low-stock", CategoryID: 3, Brand: "X", Name: "Low", Price: 100,
		Status: "in_stock", InStock: true, StockQty: 5,
		Images: []model.ProductImage{{URL: "/uploads/low-stock.png"}},
	})
	mustCreateProduct(t, s, model.Product{
		Slug: "shorts-high-stock", CategoryID: 3, Brand: "X", Name: "High", Price: 200,
		Status: "in_stock", InStock: true, StockQty: 100,
		Images: []model.ProductImage{{URL: "/uploads/high-stock.png"}},
	})

	flat, err := s.GetCategoriesFlat(ctx)
	if err != nil {
		t.Fatalf("GetCategoriesFlat: %v", err)
	}

	byID := make(map[int]*model.Category)
	for i := range flat {
		byID[flat[i].ID] = &flat[i]
	}

	// Category 3 should have cover_image = high-stock product's URL (raw, no thumb_).
	cat3 := byID[3]
	if cat3 == nil {
		t.Fatal("category 3 not found")
	}
	if cat3.CoverImage == nil {
		t.Fatal("category 3 has products → cover_image must not be nil")
	}
	if *cat3.CoverImage != "/uploads/high-stock.png" {
		t.Errorf("category 3 cover_image = %q, want /uploads/high-stock.png", *cat3.CoverImage)
	}

	// Category 17 (no products) → cover_image nil.
	cat17 := byID[17]
	if cat17 == nil {
		t.Fatal("category 17 not found")
	}
	if cat17.CoverImage != nil {
		t.Errorf("category 17 (no products) cover_image = %v, want nil", cat17.CoverImage)
	}

	// Category 1 (parent) should pick up the best image from its children
	// (including category 3). The highest-stock product overall among children
	// is the high-stock shorts (stock=100).
	cat1 := byID[1]
	if cat1 == nil {
		t.Fatal("category 1 not found")
	}
	if cat1.CoverImage == nil {
		t.Fatal("category 1 has children with products → cover_image must not be nil")
	}
	if *cat1.CoverImage != "/uploads/high-stock.png" {
		t.Errorf("category 1 cover_image = %q, want /uploads/high-stock.png", *cat1.CoverImage)
	}
}

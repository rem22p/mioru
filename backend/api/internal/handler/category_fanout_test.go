package handler

import (
	"context"
	"testing"

	"mioru/internal/model"
	"mioru/internal/storetest"
)

// TestCategoryFanOutFitsFilterCap ties maxCategoryValues to the taxonomy it
// actually has to carry. Picking a root category makes the storefront send
// every descendant id in one ?category_id= array (CatalogPage.tsx builds it
// from the category tree), so the cap is not a free number: once a root grows
// past it, that whole section of the catalog answers 400 and no code changed —
// only a seed migration did. The tree has already grown that way (021 took
// "Аксессуары" to 12), which is why this pins the two together instead of
// trusting the margin.
func TestCategoryFanOutFitsFilterCap(t *testing.T) {
	st := storetest.Fresh(t)

	cats, err := st.GetCategories(context.Background())
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(cats) == 0 {
		t.Fatal("seeded category tree is empty — the migration seed is the fixture here")
	}

	var size func(c model.Category) int
	size = func(c model.Category) int {
		n := 1
		for _, ch := range c.Children {
			n += size(ch)
		}
		return n
	}

	for _, root := range cats {
		if got := size(root); got > maxCategoryValues {
			t.Errorf("root %q expands to %d category ids, over maxCategoryValues=%d: "+
				"selecting it in the catalog would 400", root.Name, got, maxCategoryValues)
		}
	}
}

package store

import (
	"context"
	"testing"

	"mioru/internal/model"
)

func intPtr(v int) *int { return &v }

func mustCreateRanked(t *testing.T, s *PostgresStore, slug string) int64 {
	t.Helper()
	id, err := s.CreateProduct(context.Background(), model.Product{
		Slug: slug, CategoryID: 2, Brands: []string{"RankBrand"},
		Name: slug, Price: 100, Status: "in_stock", InStock: true,
	})
	if err != nil {
		t.Fatalf("CreateProduct(%s): %v", slug, err)
	}
	return id
}

// TestUpdateProductRanksMixedBatch pins #71 F3/F4: the batched rank save
// targets the right column per call, updates every row in one statement, and
// leaves the sibling rank column untouched.
func TestUpdateProductRanksMixedBatch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	ids := make(map[string]int64)
	for _, slug := range []string{"rank-a", "rank-b", "rank-c", "rank-d", "rank-e"} {
		ids[slug] = mustCreateRanked(t, s, slug)
	}

	// 3 in-stock ranks in one save (Preorder=false → main column).
	inStock := []model.RankEntry{
		{ID: ids["rank-a"], Rank: 1},
		{ID: ids["rank-b"], Rank: 2},
		{ID: ids["rank-c"], Rank: 3},
	}
	if err := s.UpdateProductRanks(ctx, inStock); err != nil {
		t.Fatalf("UpdateProductRanks(in_stock): %v", err)
	}

	// 2 preorder ranks in a separate save (Preorder=true → preorder column).
	preorder := []model.RankEntry{
		{ID: ids["rank-d"], Rank: 10, Preorder: true},
		{ID: ids["rank-e"], Rank: 20, Preorder: true},
	}
	if err := s.UpdateProductRanks(ctx, preorder); err != nil {
		t.Fatalf("UpdateProductRanks(preorder): %v", err)
	}

	// Assert both rank columns independently.
	want := map[string]struct{ main, pre *int }{
		"rank-a": {intPtr(1), nil},
		"rank-b": {intPtr(2), nil},
		"rank-c": {intPtr(3), nil},
		"rank-d": {nil, intPtr(10)},
		"rank-e": {nil, intPtr(20)},
	}
	for slug, w := range want {
		got, err := s.GetProduct(ctx, slug)
		if err != nil {
			t.Fatalf("GetProduct(%s): %v", slug, err)
		}
		if (got.PopularityRank == nil) != (w.main == nil) ||
			(got.PopularityRank != nil && *got.PopularityRank != *w.main) {
			t.Errorf("%s: popularity_rank = %v, want %v", slug, got.PopularityRank, w.main)
		}
		if (got.PopularityRankPreorder == nil) != (w.pre == nil) ||
			(got.PopularityRankPreorder != nil && *got.PopularityRankPreorder != *w.pre) {
			t.Errorf("%s: popularity_rank_preorder = %v, want %v", slug, got.PopularityRankPreorder, w.pre)
		}
	}
}

// TestListProductsSortPopular pins the popular sort contract: ranked products
// ascend by rank, unranked products land in the NULL tail. Ranks are set via
// the admin path (UpdateProductRanks) — CreateProduct does not carry them.
func TestListProductsSortPopular(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	id30 := mustCreateRanked(t, s, "pop-30")
	mustCreateRanked(t, s, "pop-null")
	id10 := mustCreateRanked(t, s, "pop-10")

	if err := s.UpdateProductRanks(ctx, []model.RankEntry{
		{ID: id30, Rank: 30},
		{ID: id10, Rank: 10},
	}); err != nil {
		t.Fatalf("UpdateProductRanks: %v", err)
	}

	got, total, err := s.ListProducts(ctx, model.ProductFilter{Sort: "popular"})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	wantOrder := []string{"pop-10", "pop-30", "pop-null"}
	for i, w := range wantOrder {
		if got[i].Slug != w {
			t.Fatalf("order[%d] = %s, want %s (full: %v)", i, got[i].Slug, w, slugsOf(got))
		}
	}
}

// TestUpdateProductRanksRejectsMixedBatch pins the store-side gate: the
// column comes from the first entry, so a mixed batch must error instead
// of silently writing every row into one column.
func TestUpdateProductRanksRejectsMixedBatch(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	a := mustCreateRanked(t, s, "rank-mix-a")
	b := mustCreateRanked(t, s, "rank-mix-b")

	err := s.UpdateProductRanks(ctx, []model.RankEntry{
		{ID: a, Rank: 1},
		{ID: b, Rank: 2, Preorder: true},
	})
	if err == nil {
		t.Fatalf("mixed batch accepted, want error")
	}
}

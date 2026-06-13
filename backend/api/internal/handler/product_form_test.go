package handler

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"mioru/internal/model"
)

// TestParseProductFromFormNormalizesLegacyStatus guards the wire-format
// contract for the admin product form. The admin SPA was written before the
// status CHECK constraint was tightened (feat/catalog-status-toggle), so its
// dropdown still emits the old values:
//
//	pre_order  → canonical "preorder"
//	none       → canonical "out_of_stock"
//
// The handler is the single source of truth for what "status" means on the
// wire — anything that lands in the DB must be one of the three canonical
// values, otherwise the CHECK constraint on products.status will reject the
// row and the admin will get a generic 500 ("internal error").
func TestParseProductFromFormNormalizesLegacyStatus(t *testing.T) {
	cases := []struct {
		name string
		body string
		want model.Product
	}{
		{
			name: "legacy pre_order + in_stock=0 → preorder",
			body: "slug=a&name=A&status=pre_order&in_stock=0&category_id=2",
			want: model.Product{Slug: "a", Name: "A", Status: "preorder", InStock: false, CategoryID: 2},
		},
		{
			name: "legacy none + in_stock=0 → out_of_stock",
			body: "slug=a&name=A&status=none&in_stock=0&category_id=2",
			want: model.Product{Slug: "a", Name: "A", Status: "out_of_stock", InStock: false, CategoryID: 2},
		},
		{
			name: "canonical preorder → preorder (no-op)",
			body: "slug=a&name=A&status=preorder&in_stock=0&category_id=2",
			want: model.Product{Slug: "a", Name: "A", Status: "preorder", InStock: false, CategoryID: 2},
		},
		{
			name: "canonical in_stock → in_stock (no-op)",
			body: "slug=a&name=A&status=in_stock&in_stock=1&category_id=2",
			want: model.Product{Slug: "a", Name: "A", Status: "in_stock", InStock: true, CategoryID: 2},
		},
		{
			name: "empty status + in_stock=0 (legacy) → out_of_stock",
			body: "slug=a&name=A&status=&in_stock=0&category_id=2",
			want: model.Product{Slug: "a", Name: "A", Status: "out_of_stock", InStock: false, CategoryID: 2},
		},
		{
			name: "empty status + in_stock=1 (legacy) → in_stock",
			body: "slug=a&name=A&status=&in_stock=1&category_id=2",
			want: model.Product{Slug: "a", Name: "A", Status: "in_stock", InStock: true, CategoryID: 2},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			// r.FormValue reads url.Values from r.PostForm, populated by
			// ParseForm. NewRequest doesn't call ParseForm, so we trigger it
			// manually here.
			if err := req.ParseForm(); err != nil {
				t.Fatal(err)
			}
			_ = url.Values{} // keep url import used (silent for future use)
			got, err := parseProductFromForm(req)
			if err != nil {
				t.Fatalf("parseProductFromForm: %v", err)
			}
			if got.Status != tc.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tc.want.Status)
			}
			if got.InStock != tc.want.InStock {
				t.Errorf("InStock = %v, want %v", got.InStock, tc.want.InStock)
			}
		})
	}
}

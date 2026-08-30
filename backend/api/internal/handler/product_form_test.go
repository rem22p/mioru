package handler

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"mioru/internal/model"
)

// TestParseProductFromFormMultiValueStock verifies that the
// size_stocks[] form field is parsed correctly alongside sizes[].
func TestParseProductFromFormMultiValueStock(t *testing.T) {
	tests := []struct {
		name   string
		sizes  []string
		stocks []string
		want   []model.ProductSize
	}{
		{
			name:   "single size with stock",
			sizes:  []string{"M"},
			stocks: []string{"5"},
			want:   []model.ProductSize{{Label: "M", StockQuantity: 5}},
		},
		{
			name:   "multiple sizes with stock",
			sizes:  []string{"S", "M", "L"},
			stocks: []string{"2", "3", "1"},
			want: []model.ProductSize{
				{Label: "S", StockQuantity: 2},
				{Label: "M", StockQuantity: 3},
				{Label: "L", StockQuantity: 1},
			},
		},
		{
			name:   "no stocks array — defaults to 0",
			sizes:  []string{"XL"},
			stocks: nil,
			want:   []model.ProductSize{{Label: "XL", StockQuantity: 0}},
		},
		{
			name:   "fewer stocks than sizes — remaining get 0",
			sizes:  []string{"A", "B", "C"},
			stocks: []string{"1"},
			want: []model.ProductSize{
				{Label: "A", StockQuantity: 1},
				{Label: "B", StockQuantity: 0},
				{Label: "C", StockQuantity: 0},
			},
		},
		{
			name:   "invalid stock value — treated as 0",
			sizes:  []string{"M"},
			stocks: []string{"abc"},
			want:   []model.ProductSize{{Label: "M", StockQuantity: 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form["slug"] = []string{strings.ToLower(strings.ReplaceAll(tt.name, " ", "-"))}
			form["category_id"] = []string{"2"}
			form["brand"] = []string{"Test"}
			form["name"] = []string{tt.name}
			form["price"] = []string{"100"}
			form["color"] = []string{"red"}
			for _, s := range tt.sizes {
				form.Add("sizes[]", s)
			}
			for _, s := range tt.stocks {
				form.Add("size_stocks[]", s)
			}

			req, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			_ = req.ParseForm()

			p, err := parseProductFromForm(req)
			if err != nil {
				t.Fatalf("parseProductFromForm: %v", err)
			}
			if len(p.Sizes) != len(tt.want) {
				t.Fatalf("got %d sizes, want %d", len(p.Sizes), len(tt.want))
			}
			for i, want := range tt.want {
				got := p.Sizes[i]
				if got.Label != want.Label {
					t.Errorf("size[%d].Label = %q, want %q", i, got.Label, want.Label)
				}
				if got.StockQuantity != want.StockQuantity {
					t.Errorf("size[%d].StockQuantity = %d, want %d", i, got.StockQuantity, want.StockQuantity)
				}
			}
		})
	}
}

// TestParseProductFromFormBrands pins KAN-14: brands arrive as a multipart
// brands[] array, are trimmed/deduped, and the legacy single "brand" field
// still parses as a one-element array.
func TestParseProductFromFormBrands(t *testing.T) {
	build := func(brands []string, legacy string) *http.Request {
		form := url.Values{}
		form["slug"] = []string{"brands-case"}
		form["category_id"] = []string{"2"}
		form["name"] = []string{"Brands Case"}
		form["price"] = []string{"100"}
		if legacy != "" {
			form["brand"] = []string{legacy}
		}
		for _, b := range brands {
			form.Add("brands[]", b)
		}
		req, _ := http.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_ = req.ParseForm()
		return req
	}

	t.Run("brands[] array is parsed and normalised", func(t *testing.T) {
		p, err := parseProductFromForm(build([]string{" Bape ", "Mastermind", "Bape", " "}, ""))
		if err != nil {
			t.Fatalf("parseProductFromForm: %v", err)
		}
		want := []string{"Bape", "Mastermind"}
		if len(p.Brands) != len(want) {
			t.Fatalf("Brands = %v, want %v", p.Brands, want)
		}
		for i := range want {
			if p.Brands[i] != want[i] {
				t.Errorf("Brands[%d] = %q, want %q", i, p.Brands[i], want[i])
			}
		}
	})

	t.Run("legacy brand field falls back to a single brand", func(t *testing.T) {
		p, err := parseProductFromForm(build(nil, "Nike"))
		if err != nil {
			t.Fatalf("parseProductFromForm: %v", err)
		}
		if len(p.Brands) != 1 || p.Brands[0] != "Nike" {
			t.Errorf("Brands = %v, want [Nike]", p.Brands)
		}
	})

	t.Run("brands[] wins over the legacy brand field", func(t *testing.T) {
		p, err := parseProductFromForm(build([]string{"Bape"}, "Nike"))
		if err != nil {
			t.Fatalf("parseProductFromForm: %v", err)
		}
		if len(p.Brands) != 1 || p.Brands[0] != "Bape" {
			t.Errorf("Brands = %v, want [Bape]", p.Brands)
		}
	})

	t.Run("a 60-character Cyrillic brand is accepted", func(t *testing.T) {
		// The bound is documented in characters; measuring bytes would cut
		// non-Latin brands at half the promised length.
		long := strings.Repeat("я", maxBrandLen)
		p, err := parseProductFromForm(build([]string{long}, ""))
		if err != nil {
			t.Fatalf("parseProductFromForm: %v", err)
		}
		if len(p.Brands) != 1 || p.Brands[0] != long {
			t.Errorf("Brands = %v, want the %d-character brand", p.Brands, maxBrandLen)
		}
	})

	t.Run("a brand longer than the bound is rejected in characters too", func(t *testing.T) {
		if _, err := parseProductFromForm(build([]string{strings.Repeat("я", maxBrandLen+1)}, "")); err == nil {
			t.Fatalf("expected an error for a %d-character brand", maxBrandLen+1)
		}
	})

	t.Run("normalisation leaves the parsed form untouched", func(t *testing.T) {
		req := build([]string{" Bape ", "Mastermind"}, "")
		if _, err := parseProductFromForm(req); err != nil {
			t.Fatalf("parseProductFromForm: %v", err)
		}
		raw := req.Form["brands[]"]
		if len(raw) != 2 || raw[0] != " Bape " || raw[1] != "Mastermind" {
			t.Errorf("r.Form[\"brands[]\"] = %q after parsing, want the values as sent", raw)
		}
	})

	t.Run("too many brands rejected", func(t *testing.T) {
		_, err := parseProductFromForm(build([]string{"A", "B", "C", "D", "E", "F"}, ""))
		if err == nil {
			t.Fatal("expected an error for 6 brands")
		}
	})

	t.Run("overlong brand rejected", func(t *testing.T) {
		long := strings.Repeat("x", maxBrandLen+1)
		_, err := parseProductFromForm(build([]string{long}, ""))
		if err == nil {
			t.Fatal("expected an error for an overlong brand")
		}
	})

	t.Run("empty brands array is allowed (brands default to empty)", func(t *testing.T) {
		p, err := parseProductFromForm(build([]string{"   "}, ""))
		if err != nil {
			t.Fatalf("parseProductFromForm: %v", err)
		}
		if len(p.Brands) != 0 {
			t.Errorf("Brands = %v, want empty", p.Brands)
		}
	})
}

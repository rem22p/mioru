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
		name     string
		sizes    []string
		stocks   []string
		want     []model.ProductSize
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

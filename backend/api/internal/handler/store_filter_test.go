package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func repeatValues(prefix string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%d", prefix, i)
	}
	return out
}

// TestParseProductFilterBrandCaps pins #88 S1: the public brand filter is
// bounded (20 values × 60 runes), so a crafted array can't make the DB work
// on an arbitrarily large overlap/ILIKE input.
func TestParseProductFilterBrandCaps(t *testing.T) {
	mk := func(brands []string) error {
		q := ""
		for _, b := range brands {
			if q != "" {
				q += "&"
			}
			q += "brand=" + b
		}
		_, err := parseProductFilter(httptest.NewRequest("GET", "/api/products?"+q, nil))
		return err
	}

	if err := mk([]string{"A", "B", "C"}); err != nil {
		t.Errorf("3 brands rejected: %v", err)
	}

	many := make([]string, 21)
	for i := range many {
		many[i] = "B"
	}
	if err := mk(many); err == nil {
		t.Errorf("21 brands accepted, want error")
	}

	if err := mk([]string{strings.Repeat("я", 61)}); err == nil {
		t.Errorf("61-rune brand accepted, want error")
	}
}

// TestParseProductFilterCaps pins #92 F4: the public color/size arrays are
// bounded the same way the brand array is (Maxim's S1 hardening) — a crafted
// oversized list is rejected, not shipped to the DB.
func TestParseProductFilterCaps(t *testing.T) {
	build := func(params map[string][]string) *http.Request {
		q := url.Values{}
		for k, vs := range params {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		return httptest.NewRequest(http.MethodGet, "/api/products?"+q.Encode(), nil)
	}

	tests := []struct {
		name    string
		params  map[string][]string
		wantErr string
	}{
		{
			name:    "21 color values rejected",
			params:  map[string][]string{"color": repeatValues("Цвет-", 21)},
			wantErr: "too many color values",
		},
		{
			name:    "21 size values rejected",
			params:  map[string][]string{"size": repeatValues("42-", 21)},
			wantErr: "too many size values",
		},
		{
			name:    "overlong color value rejected",
			params:  map[string][]string{"color": {strings.Repeat("ж", 41)}},
			wantErr: "color value too long",
		},
		{
			name:    "overlong size value rejected",
			params:  map[string][]string{"size": {strings.Repeat("4", 33)}},
			wantErr: "size value too long",
		},
		{
			name:    "21 brand values rejected (S1 regression)",
			params:  map[string][]string{"brand": repeatValues("Brand-", 21)},
			wantErr: "too many brand values",
		},
		{
			name: "20 colors and sizes pass",
			params: map[string][]string{
				"color": repeatValues("Цвет-", 20),
				"size":  repeatValues("42-", 20),
			},
		},
		{
			name: "exactly 40-rune color and 32-rune size pass",
			params: map[string][]string{
				"color": {strings.Repeat("ж", 40)},
				"size":  {strings.Repeat("4", 32)},
			},
		},
		{
			name: "CSV form counts toward the same caps",
			params: map[string][]string{
				"color": {strings.Join(repeatValues("Цвет-", 21), ",")},
			},
			wantErr: "too many color values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseProductFilter(build(tt.params))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

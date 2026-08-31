package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

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

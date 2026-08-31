package telegram

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"mioru/internal/model"
)


// captureLogger swaps slog.Default() for a buffer-backed handler
// for the duration of fn, returning whatever was written. Used by
// the new-failure-mode tests below to assert that the notifier
// *logs* its skip / error conditions instead of swallowing them
// silently (the original bug).
func captureLogger(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	fn()
	return buf.String()
}





// redirector is a tiny http.RoundTripper that rewrites every
// request's URL host to `target`. Used to point the notifier at
// an httptest server without exposing a test-only field on
// Notifier.
type redirector struct{ target string }

func (r redirector) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rebuild the URL with the test server's scheme+host.
	u, err := req.URL.Parse(r.target + req.URL.Path + "?" + req.URL.RawQuery)
	if err != nil {
		return nil, err
	}
	req.URL = u
	req.Host = u.Host
	return http.DefaultTransport.RoundTrip(req)
}




// full fallback path: an unescaped "." in the input must
// cause sanitizeForMarkdownV2 to log a warning and return
// the input with every backslash stripped (Telegram's
// TestFormatOrderMessageHTMLEmitsAdminLinks covers the new
// HTML-mode rendering path. When adminURL is set, the
// rendered message must contain:
//
//   * <a href="{adminURL}/customers/{id}">Name Surname</a>
//     for the customer
//   * <a href="{adminURL}/products/{id}">Product Name</a>
//     per cart item
//
// And when adminURL is empty the message must still ship
// (graceful degradation) with the names as plain text. We
// only assert on the *presence* of the tags here — the real
// defence for the link path is the live end-to-end test
// in the dev server, which is what we use to verify that
// Telegram actually returns a `text_link` entity for the
// generated markup.
func TestFormatOrderMessageHTMLEmitsAdminLinks(t *testing.T) {
	o := &model.Order{
		ID: 42, Type: "cart", TotalMinor: 5000, Status: "pending",
		Phone:          "+373777908542",
		City:           "Тирасполь",
		DeliveryMethod: "personal",
		PaymentMethod:  "card",
		Items: []model.OrderItem{{
			ProductID:   7,
			ProductName: "Худи Mioru Classic",
			SizeLabel:   "XL",
			Quantity:    1,
			PriceMinor:  5000,
		}},
	}
	c := &model.Customer{ID: 2, Email: "c@ex.com", FirstName: "Test", LastName: "Customer", Phone: ""}

	t.Run("with adminURL", func(t *testing.T) {
		n := NewNotifier("", nil, "", t.TempDir(), "https://admin.mioru.store", "https://mioru.store")
		got := n.formatOrderMessageHTML(o, c)
		wantCustomer := `<a href="https://admin.mioru.store/customers/2">Test Customer</a>`
		if !strings.Contains(got, wantCustomer) {
			t.Errorf("customer link not found in:\n%s\nwant: %s", got, wantCustomer)
		}
		wantProduct := `<a href="https://mioru.store/product/7">Худи Mioru Classic</a>`
		if !strings.Contains(got, wantProduct) {
			t.Errorf("product link not found in:\n%s\nwant: %s", got, wantProduct)
		}
	})

	t.Run("without adminURL", func(t *testing.T) {
		n := NewNotifier("", nil, "", t.TempDir(), "", "")
		got := n.formatOrderMessageHTML(o, c)
		if strings.Contains(got, `<a href="`) {
			t.Errorf("expected no link markup, got: %s", got)
		}
		if !strings.Contains(got, "Test Customer") {
			t.Errorf("customer name should still appear as plain text, got: %s", got)
		}
	})
}

// TestEscapeHTML covers the four-character HTML escape used
// in the HTML rendering path. We escape the four markup
// delimiters that Telegram's HTML parser reserves (&, <, >,
// ") and leave every other character alone. The pin is
// important because over-escaping would break link markup
// (e.g. `&amp;` inside an `href` attribute is fine for
// display, but `&` inside a URL is a real character) and
// under-escaping would let a customer who types
// `<script>alert(1)</script>` in the comment field inject
// markup into the message.
func TestEscapeHTML(t *testing.T) {
	cases := []struct{ in, want string }{
		{"hello", "hello"},
		{"<b>жирный</b>", "&lt;b&gt;жирный&lt;/b&gt;"},
		{"M&M <Sale>", "M&amp;M &lt;Sale&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		// Periods, parens, stars are all literal in HTML.
		{"0.00 лей (XL)", "0.00 лей (XL)"},
		{"*foo* _bar_", "*foo* _bar_"},
		// Empty stays empty.
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := escapeHTML(tc.in); got != tc.want {
				t.Errorf("escapeHTML(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestIndividualFieldsHTML pins #88 F5: a shoes order carries the category
// and the exact insole length (no %.1f rounding — «10.25» must survive),
// while a cart order renders no individual block at all.
func TestIndividualFieldsHTML(t *testing.T) {
	fl := func(v float64) *float64 { return &v }
	h := func(v float64) *float64 { return &v }

	t.Run("shoes order carries category and exact foot length", func(t *testing.T) {
		o := &model.Order{Type: "individual", Category: "shoes", FootLength: fl(10.25)}
		got := individualFieldsHTML(o)
		if !strings.Contains(got, "Категория:</b> Обувь") {
			t.Errorf("missing category line: %q", got)
		}
		if !strings.Contains(got, "Длина стельки:</b> 10.25 см") {
			t.Errorf("foot length rounded or missing, got: %q", got)
		}
	})

	t.Run("clothing order carries category and height/weight", func(t *testing.T) {
		o := &model.Order{Type: "individual", Category: "clothing", Height: h(175), Weight: h(70)}
		got := individualFieldsHTML(o)
		if !strings.Contains(got, "Категория:</b> Одежда") {
			t.Errorf("missing category line: %q", got)
		}
		if !strings.Contains(got, "Рост:</b> 175 см") || !strings.Contains(got, "Вес:</b> 70 кг") {
			t.Errorf("missing measurements, got: %q", got)
		}
	})

	t.Run("cart order renders no individual block", func(t *testing.T) {
		if got := individualFieldsHTML(&model.Order{Type: "cart", Category: "shoes"}); got != "" {
			t.Errorf("cart order leaked individual fields: %q", got)
		}
	})
}

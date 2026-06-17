package telegram

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/model"
)

// TestFormatOrderMessageUsesOrderPhone guards the contract that
// managers see the *order's* phone in the Telegram notification, not
// the customer's profile phone. The two can differ in three real
// scenarios:
//
//  1. Anonymous checkout — the order has no associated customer row
//     at all, so c.Phone is "" by construction. Without this fix the
//     Telegram message would be missing the contact line entirely.
//  2. Profile sync failure — even when c is non-nil, the best-effort
//     UpdateCustomer in customer.go::CreateOrder can fail. The order
//     still saves with o.Phone, but c.Phone would still be the old
//     value (possibly "").
//  3. Customer edited their profile to a different number since the
//     last order. We don't want a manager to call the *profile*
//     number, we want the number the customer used *for this order*.
//
// All three are "wrong number on Telegram" bugs that we closed by
// switching the formatOrderMessage source-of-truth from c.Phone to
// o.Phone. The contract is enforced by this test.
func TestFormatOrderMessageUsesOrderPhone(t *testing.T) {
	cases := []struct {
		name     string
		oPhone   string
		cPhone   string
		wantSeen bool
	}{
		{
			name:     "logged-in customer, phones match",
			oPhone:   "+373777908542",
			cPhone:   "+373777908542",
			wantSeen: true,
		},
		{
			name:     "order phone present, profile phone different (stale profile)",
			oPhone:   "+373777908542",
			cPhone:   "+37360000000",
			wantSeen: true, // we want the order's number, not the stale profile one
		},
		{
			name:     "anonymous checkout (nil customer profile)",
			oPhone:   "+373777908542",
			cPhone:   "", // no profile to fall back to
			wantSeen: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := &model.Order{
				ID: 1, Type: "cart", TotalMinor: 5000, Status: "pending",
				Phone:          tc.oPhone,
				City:           "Tiraspol",
				DeliveryMethod: "personal",
				PaymentMethod:  "cash",
			}
			c := &model.Customer{
				Email: "c@ex.com", FirstName: "T", LastName: "C", Phone: tc.cPhone,
			}
			got := formatOrderMessage(o, c)
			if tc.wantSeen && !strings.Contains(got, tc.oPhone) {
				t.Errorf("expected order phone %q to appear in Telegram message:\n%s",
					tc.oPhone, got)
			}
			// Defence-in-depth: the *profile* phone (when different
			// from the order phone) must NOT appear in the message.
			// If it does, a stale-profile bug has snuck back in.
			if tc.cPhone != "" && tc.cPhone != tc.oPhone &&
				strings.Contains(got, tc.cPhone) {
				t.Errorf("profile phone %q should not appear in Telegram message "+
					"when it differs from the order phone %q:\n%s",
					tc.cPhone, tc.oPhone, got)
			}
		})
	}
}

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

// TestNotifierSkipsWithWarnWhenBotTokenEmpty guards against the
// "notifications just don't arrive" bug: when the bot token isn't
// configured, OrderCreated must log a WARN line (not silently
// return) so the manager can see the wiring is missing in the
// server logs.
func TestNotifierSkipsWithWarnWhenBotTokenEmpty(t *testing.T) {
	n := NewNotifier("", []string{"123"}, "", t.TempDir())
	out := captureLogger(t, func() {
		n.OrderCreated(&model.Order{ID: 42}, &model.Customer{})
	})
	if !strings.Contains(out, "TELEGRAM_BOT_TOKEN not set") {
		t.Errorf("expected warn about missing bot token, got:\n%s", out)
	}
	if !strings.Contains(out, "order_id=42") {
		t.Errorf("expected order_id field, got:\n%s", out)
	}
}

// TestNotifierSkipsWithWarnWhenChatIDsEmpty is the chat_ids twin
// of the test above. Same bug, second env var.
func TestNotifierSkipsWithWarnWhenChatIDsEmpty(t *testing.T) {
	n := NewNotifier("telegram-test-token-1234567890", nil, "", t.TempDir())
	out := captureLogger(t, func() {
		n.OrderCreated(&model.Order{ID: 43}, &model.Customer{})
	})
	if !strings.Contains(out, "TELEGRAM_MANAGER_CHAT_IDS not set") {
		t.Errorf("expected warn about missing chat ids, got:\n%s", out)
	}
}

// TestNotifierHealthCheckReturnsErrorWhenTokenEmpty is the
// "don't even try Telegram if we have no token" half of the
// startup health check. The non-empty case is covered indirectly
// by `TestNotifierHealthCheck_Success` below via an httptest
// server.
func TestNotifierHealthCheckReturnsErrorWhenTokenEmpty(t *testing.T) {
	n := NewNotifier("", nil, "", t.TempDir())
	_, err := n.HealthCheck(context.Background())
	if err == nil {
		t.Fatalf("expected error when token is empty")
	}
	if !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN not set") {
		t.Errorf("error should mention the missing env var, got: %v", err)
	}
}

// TestNotifierHealthCheckReports401 guards the failure path: a
// token that's been revoked by BotFather comes back from getMe as
// 401 with `description: "Unauthorized"`. We must surface that
// description (with the token redacted) instead of returning a
// generic "request failed" so the operator can fix the right
// thing.
func TestNotifierHealthCheckReports401(t *testing.T) {
	const token = "telegram-test-token-1234567890"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized"}`))
	}))
	defer srv.Close()
	// The real NewNotifier pins the URL to api.telegram.org;
	// for the test we use a small wrapper that points at srv.URL.
	n := NewNotifier(token, nil, "", t.TempDir())
	n.client = srv.Client()
	// Swap the URL by replacing the client and routing through a
	// helper. Simpler: call HealthCheck with a redirector client
	// that rewrites api.telegram.org → srv.URL.
	n.client = &http.Client{
		Timeout: 5 * 1e9,
		Transport: redirector{target: srv.URL},
	}
	_, err := n.HealthCheck(context.Background())
	if err == nil {
		t.Fatalf("expected 401 error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected status 401 in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Unauthorized") {
		t.Errorf("expected description 'Unauthorized' in error, got: %v", err)
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("token leaked into error message: %v", err)
	}
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

// TestNotifierHealthCheckReturnsUsername covers the success path
// — the notifier's username is what the startup log line carries
// ("telegram notifier ready bot_username=...").
func TestNotifierHealthCheckReturnsUsername(t *testing.T) {
	const token = "telegram-test-token-1234567890"
	const want = "mioru_orders_bot"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"username":"` + want + `"}}`))
	}))
	defer srv.Close()
	n := NewNotifier(token, nil, "", t.TempDir())
	n.client = &http.Client{
		Timeout: 5 * 1e9,
		Transport: redirector{target: srv.URL},
	}
	got, err := n.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("username = %q, want %q", got, want)
	}
}

// TestIsMarkdownV2Safe covers the O(n) validator that the
// notifier runs over the rendered message before posting. It
// must return false for any *unescaped* MarkdownV2 reserved
// character (`.`, `(`, `)`, `!`, etc.) and true for the
// escaped forms + the `*` / `_` characters Telegram
// accepts as bold/italic markers. The regression that
// motivated this helper was an unescaped "." in a price
// ("0.00") that cost a 400 from Telegram and a silent
// dropped manager notification.
func TestIsMarkdownV2Safe(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"plain", "hello world", true},
		{"escaped dot", `0\.00 лей`, true},
		{"escaped paren", `\(размер XL\)`, true},
		{"unescaped dot", "0.00 лей", false},
		{"unescaped paren", "(размер XL)", false},
		{"unescaped bang", "привет!", false},
		{"unescaped hash", "#tag", false},
		{"bold marker", "*жирный*", true},
		{"italic marker", "_курсив_", true},
		{"mixed safe and escaped", "*Итого:* 0\\.00 лей", true},
		{"mixed with unescaped", "*Итого:* 0.00 лей", false},
		{"empty", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isMarkdownV2Safe(tc.in); got != tc.want {
				t.Errorf("isMarkdownV2Safe(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestSanitizeForMarkdownV2FallsBackToPlainText exercises the
// full fallback path: an unescaped "." in the input must
// cause sanitizeForMarkdownV2 to log a warning and return
// the input with every backslash stripped (Telegram's
// "plain text" mode treats backslashes as literal text, so
// stripping them is what we want). The warning is captured
// via the same captureLogger helper used by the other
// failure-mode tests.
func TestSanitizeForMarkdownV2FallsBackToPlainText(t *testing.T) {
	n := NewNotifier("telegram-test-token-1234567890", []string{"1"}, "", t.TempDir())
	in := `*Итого:* 0.00 лей \(escape me\)`
	out := captureLogger(t, func() {
		got := n.sanitizeForMarkdownV2(in)
		// Every MarkdownV2 special (`*`, `.`, `(`, `)`)
		// must end up bare in the plain-text output. The
		// original backslashes around the trailing "escape
		// me" are gone, but the parentheses are now literal
		// characters that Telegram will render as plain text.
		if got != `*Итого:* 0.00 лей (escape me)` {
			t.Errorf("plain-text fallback = %q, want %q", got, `*Итого:* 0.00 лей (escape me)`)
		}
	})
	if !strings.Contains(out, "falling back to plain text") {
		t.Errorf("expected warning log about plain-text fallback, got:\n%s", out)
	}
}

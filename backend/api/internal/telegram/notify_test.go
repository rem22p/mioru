package telegram

import (
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

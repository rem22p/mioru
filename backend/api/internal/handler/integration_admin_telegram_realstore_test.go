package handler_test

import (
	"context"
	"net/http"
	"testing"
)

// TestIntegrationAdminTelegramRealStoreMessagesRoundTrip exercises the
// real PostgresStore-backed telegram recorder and the
// Messages/Diagnose endpoints against a test database. The
// pre-existing stub-based tests (integration_admin_telegram_test.go)
// did not verify the SQL — they used in-memory structs. This
// test pins the actual Postgres queries.
//
// B2 (final gate): telegram_messages.go had no real-PG test.
// CLAUDE.md: "DB-touching logic is tested against a real
// PostgreSQL, not mocks".
func TestIntegrationAdminTelegramRealStoreMessagesRoundTrip(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "tg-real-store", "admin")

	// Record a send through the real store (not a stub).
	rowID, err := e.st.RecordTelegramSend(context.Background(),
		nil, "111", "hello world", "HTML")
	if err != nil {
		t.Fatalf("RecordTelegramSend: %v", err)
	}
	if rowID == 0 {
		t.Fatal("RecordTelegramSend returned 0 id")
	}

	// Mark it as sent.
	if err := e.st.MarkTelegramSent(context.Background(),
		rowID, 200, ptr(int64(42)), 150); err != nil {
		t.Fatalf("MarkTelegramSent: %v", err)
	}

	// Messages list should now contain the row.
	rr := e.do(t, e.wrapAdmin(e.adminTelegramH.Messages), http.MethodGet,
		"/api/admin/telegram/messages", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("Messages: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}

	var resp struct {
		Messages []map[string]any `json:"messages"`
		Total    int              `json:"total"`
	}
	decode(t, rr, &resp)
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1 (seeded one row)", resp.Total)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("messages count = %d, want 1", len(resp.Messages))
	}
	m := resp.Messages[0]
	if v, _ := m["status"].(string); v != "sent" {
		t.Errorf("status = %q, want sent", v)
	}
	if v, _ := m["parse_mode"].(string); v != "HTML" {
		t.Errorf("parse_mode = %q, want HTML", v)
	}

	// Diagnose endpoint (includes 24h stats).
	rr2 := e.do(t, e.wrapAdmin(e.adminTelegramH.Diagnose), http.MethodGet,
		"/api/admin/telegram/diagnose", reqOpts{sess: admin})
	if rr2.Code != http.StatusOK {
		t.Fatalf("Diagnose: want 200, got %d (%s)", rr2.Code, rr2.Body.String())
	}
	var diag struct {
		Last24hTotal  int `json:"last_24h_total"`
		Last24hSent   int `json:"last_24h_sent"`
		Last24hFailed int `json:"last_24h_failed"`
	}
	decode(t, rr2, &diag)
	if diag.Last24hTotal < 1 {
		t.Errorf("last_24h_total = %d, want >= 1", diag.Last24hTotal)
	}
	if diag.Last24hSent < 1 {
		t.Errorf("last_24h_sent = %d, want >= 1", diag.Last24hSent)
	}
	if diag.Last24hFailed != 0 {
		t.Errorf("last_24h_failed = %d, want 0", diag.Last24hFailed)
	}

	// Record a failed send and verify stats update.
	rowID2, err := e.st.RecordTelegramSend(context.Background(),
		nil, "222", "failed msg", "HTML")
	if err != nil {
		t.Fatalf("RecordTelegramSend #2: %v", err)
	}
	if err := e.st.MarkTelegramFailed(context.Background(),
		rowID2, ptrInt32(400), "bad request", 200); err != nil {
		t.Fatalf("MarkTelegramFailed: %v", err)
	}

	rr3 := e.do(t, e.wrapAdmin(e.adminTelegramH.Diagnose), http.MethodGet,
		"/api/admin/telegram/diagnose", reqOpts{sess: admin})
	if rr3.Code != http.StatusOK {
		t.Fatalf("Diagnose #2: want 200, got %d", rr3.Code)
	}
	var diag2 struct {
		Last24hFailed int `json:"last_24h_failed"`
	}
	decode(t, rr3, &diag2)
	if diag2.Last24hFailed < 1 {
		t.Errorf("last_24h_failed = %d, want >= 1 (seeded one failed row)", diag2.Last24hFailed)
	}
}

// ptr and ptrInt32 are test helpers.
func ptr[T any](v T) *T     { return &v }
func ptrInt32(v int32) *int32 { return &v }

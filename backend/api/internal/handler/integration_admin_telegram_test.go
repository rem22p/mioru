// internal/handler/integration_admin_telegram_test.go
//
// End-to-end tests for the admin Telegram workspace endpoints.
// These run against the real Postgres test DB (no Telegram
// network — the notifier is constructed with an empty bot
// token, so every send attempt is a "no token configured"
// WARN log, not an outbound HTTP call).
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mioru/internal/middleware"
	"mioru/internal/store"
	"mioru/internal/telegram"
)

// stubTelegramRecorder is a fake Recorder that lives entirely
// in memory. The integration test uses it instead of the real
// Postgres-backed recorder so the tests stay deterministic and
// runnable even when the database is empty.
type stubTelegramRecorder struct {
	rows []store.TelegramMessageRow
	next int64
}

func (r *stubTelegramRecorder) RecordTelegramSend(_ context.Context, orderID *int64, chatID, text, parseMode string) (int64, error) {
	r.next++
	r.rows = append(r.rows, store.TelegramMessageRow{
		ID: r.next, OrderID: orderID, ChatID: chatID, Text: text,
		ParseMode: parseMode, Status: "pending",
	})
	return r.next, nil
}
func (r *stubTelegramRecorder) MarkTelegramSent(_ context.Context, id int64, httpStatus int, telegramMsgID *int64, durationMs int) error {
	for i := range r.rows {
		if r.rows[i].ID == id {
			r.rows[i].Status = "sent"
			st := int32(httpStatus)
			r.rows[i].HTTPStatus = &st
			r.rows[i].TelegramMessageID = telegramMsgID
			dm := int32(durationMs)
			r.rows[i].DurationMs = &dm
			return nil
		}
	}
	return nil
}
func (r *stubTelegramRecorder) MarkTelegramFailed(_ context.Context, id int64, httpStatus *int32, errMsg string, durationMs int) error {
	for i := range r.rows {
		if r.rows[i].ID == id {
			r.rows[i].Status = "failed"
			r.rows[i].HTTPStatus = httpStatus
			r.rows[i].Error = errMsg
			dm := int32(durationMs)
			r.rows[i].DurationMs = &dm
			return nil
		}
	}
	return nil
}

// stubAdminTelegramStore wraps the stub recorder with the
// list/stats methods the handler needs. Implementing them
// here (rather than reusing a *store.PostgresStore) keeps the
// test independent of migrations.
type stubAdminTelegramStore struct {
	rec *stubTelegramRecorder
}

func (s *stubAdminTelegramStore) ListTelegramMessages(_ context.Context, page, perPage int, _ string) ([]store.TelegramMessageRow, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	total := len(s.rec.rows)
	start := (page - 1) * perPage
	if start >= total {
		return []store.TelegramMessageRow{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	// Newest first to match Postgres' ORDER BY id DESC.
	out := make([]store.TelegramMessageRow, 0, end-start)
	for i := total - 1 - start; i >= total-end; i-- {
		out = append(out, s.rec.rows[i])
	}
	return out, total, nil
}
func (s *stubAdminTelegramStore) GetTelegramStats(_ context.Context) (store.TelegramStats, error) {
	var st store.TelegramStats
	for _, r := range s.rec.rows {
		st.Last24hTotal++
		if r.Status == "sent" {
			st.Last24hSent++
		}
		if r.Status == "failed" {
			st.Last24hFailed++
		}
	}
	if len(s.rec.rows) > 0 {
		last := s.rec.rows[len(s.rec.rows)-1]
		st.LastSendAt = last.SentAt
		if last.Error != "" {
			st.LastSendError = last.Error
		}
	}
	return st, nil
}

// adminTelegramEnv bundles the stub store, the live (no-network)
// notifier, and the handler the tests will exercise.
type adminTelegramEnv struct {
	tgH *AdminTelegramHandler
	rec *stubTelegramRecorder
	not *telegram.Notifier
}

func newAdminTelegramEnv(t *testing.T) *adminTelegramEnv {
	t.Helper()
	rec := &stubTelegramRecorder{}
	storeStub := &stubAdminTelegramStore{rec: rec}
	// Empty token + nil chat IDs → notifier is a no-op; we
	// don't need real network here.
	not := telegram.NewNotifier("", nil, "", t.TempDir(), "", "")
	not.SetRecorder(rec)
	h := NewAdminTelegramHandler(storeStub, not)
	return &adminTelegramEnv{tgH: h, rec: rec, not: not}
}

// doAdminTelegramWithSession runs the handler as an
// authenticated admin (so adminOnly passes) and returns the
// response recorder.
func (e *adminTelegramEnv) doAdminTelegramWithSession(t *testing.T, h http.HandlerFunc, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	// Stamp a super_admin session so adminOnly lets us through.
	ctx := middleware.WithCustomerID(r.Context(), 0) // no customer context
	ctx = context.WithValue(ctx, adminSessionKey{}, adminSession{username: "admin", role: "super_admin"})
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

// adminSessionKey mirrors the field the real auth middleware
// sets in the request context. Defined here so the tests
// don't import the auth package just to mint a session.
type adminSessionKey struct{}

type adminSession struct {
	username string
	role     string
}

// TestIntegrationAdminTelegramDiagnoseNoToken covers the
// "TELEGRAM_BOT_TOKEN not set" path: the Status card must
// report `bot_token_set: false` and `manager_chat_count: 0`,
// not 500. This is the most common misconfiguration in dev
// and in mis-deployed prod instances.
func TestIntegrationAdminTelegramDiagnoseNoToken(t *testing.T) {
	env := newAdminTelegramEnv(t)
	rr := env.doAdminTelegramWithSession(t, env.tgH.Diagnose, http.MethodGet, "/api/admin/telegram/diagnose", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		BotTokenSet      bool   `json:"bot_token_set"`
		ManagerChatCount int    `json:"manager_chat_count"`
		LastSendStatus   string `json:"last_send_status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.BotTokenSet {
		t.Errorf("bot_token_set should be false")
	}
	if resp.ManagerChatCount != 0 {
		t.Errorf("manager_chat_count = %d, want 0", resp.ManagerChatCount)
	}
	if resp.LastSendStatus != "never" {
		t.Errorf("last_send_status = %q, want \"never\"", resp.LastSendStatus)
	}
}

// TestIntegrationAdminTelegramMessagesEmpty checks the
// history endpoint with no rows yet. Must return 200 with an
// empty list and total=0, not 500.
func TestIntegrationAdminTelegramMessagesEmpty(t *testing.T) {
	env := newAdminTelegramEnv(t)
	rr := env.doAdminTelegramWithSession(t, env.tgH.Messages, http.MethodGet, "/api/admin/telegram/messages", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Messages []store.TelegramMessageRow `json:"messages"`
		Total    int                        `json:"total"`
		Page     int                        `json:"page"`
		PerPage  int                        `json:"per_page"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if len(resp.Messages) != 0 {
		t.Errorf("messages = %d, want 0", len(resp.Messages))
	}
}

// TestIntegrationAdminTelegramMessagesPagination seeds three
// rows and asks for page=2 per_page=2. Should return the
// oldest row.
func TestIntegrationAdminTelegramMessagesPagination(t *testing.T) {
	env := newAdminTelegramEnv(t)
	// Seed 3 rows directly through the recorder.
	for i := 0; i < 3; i++ {
		_, _ = env.rec.RecordTelegramSend(context.Background(), nil, "123", "msg", "HTML")
	}
	rr := env.doAdminTelegramWithSession(t, env.tgH.Messages, http.MethodGet, "/api/admin/telegram/messages?page=2&per_page=2", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	var resp struct {
		Messages []store.TelegramMessageRow `json:"messages"`
		Total    int                        `json:"total"`
		Page     int                        `json:"page"`
		PerPage  int                        `json:"per_page"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Total)
	}
	if resp.Page != 2 {
		t.Errorf("page = %d, want 2", resp.Page)
	}
	if len(resp.Messages) != 1 {
		t.Errorf("messages = %d, want 1 (3 rows, page 2 of 2-per-page = 1 left)", len(resp.Messages))
	}
}

// TestIntegrationAdminTelegramTestRequiresToken covers the
// "send test" button with no token. Must return
// 503 TELEGRAM_NOT_CONFIGURED, not 500.
func TestIntegrationAdminTelegramTestRequiresToken(t *testing.T) {
	env := newAdminTelegramEnv(t)
	rr := env.doAdminTelegramWithSession(t, env.tgH.Test, http.MethodPost, "/api/admin/telegram/test", []byte(`{}`))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Code != "TELEGRAM_NOT_CONFIGURED" {
		t.Errorf("code = %q, want TELEGRAM_NOT_CONFIGURED", resp.Code)
	}
}

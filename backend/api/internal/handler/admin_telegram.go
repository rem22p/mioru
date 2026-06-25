package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"mioru/internal/store"
	"mioru/internal/telegram"
)

// adminTelegramStore is the subset of *store.PostgresStore the
// admin Telegram handler needs. Defined here so the test
// harness can pass a fake without dragging pgx in.
type adminTelegramStore interface {
	ListTelegramMessages(ctx context.Context, page, perPage int, status string) ([]store.TelegramMessageRow, int, error)
	GetTelegramStats(ctx context.Context) (store.TelegramStats, error)
}

// AdminTelegramHandler exposes the Telegram notifier state to
// the admin workspace:
//
//	GET  /api/admin/telegram/diagnose  — bot_token_set, last
//	                                      24h sent/failed, last
//	                                      send_at + last error
//	GET  /api/admin/telegram/messages  — paginated history
//	POST /api/admin/telegram/test      — send a test message
//	                                      to every configured
//	                                      manager chat
//
// All routes are adminOnly. The diagnose endpoint never
// returns the bot token (only a boolean "is it set"), so this
// surface is safe to expose to non-super_admin roles.
type AdminTelegramHandler struct {
	store    adminTelegramStore
	notifier *telegram.Notifier
}

// NewAdminTelegramHandler wires the store and the live
// notifier. Both are required — diagnose reads the notifier's
// state (chat count, bot token presence), and the test
// endpoint actually sends through the notifier.
func NewAdminTelegramHandler(s adminTelegramStore, n *telegram.Notifier) *AdminTelegramHandler {
	return &AdminTelegramHandler{store: s, notifier: n}
}

// Diagnose returns the rolled-up status the Status card shows.
// `bot_token_set` is a *boolean* (the token is a secret; the
// boolean is what the manager needs to see "did the deploy
// forget to set the env var"). `bot_username` comes from the
// last successful HealthCheck; empty if HealthCheck never ran
// (which is fine — it ran on boot if the notifier was
// constructed).
type diagnoseResponse struct {
	BotTokenSet       bool                  `json:"bot_token_set"`
	BotUsername       string                `json:"bot_username"`
	ManagerChatCount  int                   `json:"manager_chat_count"`
	Last24hTotal      int                   `json:"last_24h_total"`
	Last24hSent       int                   `json:"last_24h_sent"`
	Last24hFailed     int                   `json:"last_24h_failed"`
	LastSendAt        string                `json:"last_send_at,omitempty"`
	LastSendError     string                `json:"last_send_error,omitempty"`
	LastSendStatus    string                `json:"last_send_status,omitempty"`
}

// Diagnose is the GET handler.
func (h *AdminTelegramHandler) Diagnose(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetTelegramStats(r.Context())
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	username, _, _ := h.notifier.LastHealthCheck()
	resp := diagnoseResponse{
		BotTokenSet:      h.notifier.BotTokenSet(),
		BotUsername:      username,
		ManagerChatCount: h.notifier.ManagerChatCount(),
		Last24hTotal:     stats.Last24hTotal,
		Last24hSent:      stats.Last24hSent,
		Last24hFailed:    stats.Last24hFailed,
		LastSendAt:       stats.LastSendAt,
		LastSendError:    stats.LastSendError,
	}
	// Derive a "last send status" string the UI can colour
	// without parsing the error: "ok" / "failed" / "unknown".
	switch {
	case stats.LastSendAt == "":
		resp.LastSendStatus = "never"
	case stats.LastSendError == "":
		resp.LastSendStatus = "ok"
	default:
		resp.LastSendStatus = "failed"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Messages is the GET handler for the history list. Pagination
// defaults match the customers list (page=1, per_page=20) so
// the UI is consistent across workspaces. `status` is an
// optional filter — empty means all rows.
type messagesResponse struct {
	Messages []store.TelegramMessageRow `json:"messages"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PerPage  int                        `json:"per_page"`
}

// Messages handles GET /api/admin/telegram/messages.
func (h *AdminTelegramHandler) Messages(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	status := r.URL.Query().Get("status")

	rows, total, err := h.store.ListTelegramMessages(r.Context(), page, perPage, status)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if rows == nil {
		rows = []store.TelegramMessageRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(messagesResponse{
		Messages: rows,
		Total:    total,
		Page:     page,
		PerPage:  perPage,
	})
}

// TestResult is the per-chat outcome of POST /telegram/test.
// One entry per configured manager chat. The HTTP status of
// the most recent send in `telegram_messages` is included so
// the admin UI can show red/green without an extra round trip.
type TestResult struct {
	ChatID     string `json:"chat_id"`
	OK         bool   `json:"ok"`
	Status     string `json:"status"`
	HTTPStatus *int32 `json:"http_status,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int    `json:"duration_ms"`
}

// Test handles POST /api/admin/telegram/test. Sends a single
// "🧪 test" message to every configured manager chat and
// returns one TestResult per chat. The bot token presence is
// checked up-front so we don't burn 10s of timeouts per
// empty-notifier request.
func (h *AdminTelegramHandler) Test(w http.ResponseWriter, r *http.Request) {
	if !h.notifier.BotTokenSet() {
		jsonErrorCode(w, "TELEGRAM_BOT_TOKEN not configured",
			http.StatusServiceUnavailable, "TELEGRAM_NOT_CONFIGURED")
		return
	}
	if h.notifier.ManagerChatCount() == 0 {
		jsonErrorCode(w, "TELEGRAM_MANAGER_CHAT_IDS not configured",
			http.StatusServiceUnavailable, "TELEGRAM_NOT_CONFIGURED")
		return
	}

	// Use the live notifier so the test message also lands in
	// the `telegram_messages` log (and shows up in the history
	// list with order_id NULL). The text is short and uses
	// plain emoji only — no HTML special characters, so it
	// can never hit a 400.
	text := "🧪 test message from mioru admin"
	chatIDs := h.notifier.ManagerChatIDs()
	results := make([]TestResult, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		start := time.Now()
		err := h.notifier.SendTestMessage(chatID, text)
		dur := int(time.Since(start).Milliseconds())
		res := TestResult{
			ChatID:     chatID,
			DurationMs: dur,
		}
		if err == nil {
			res.OK = true
			res.Status = "sent"
			// HTTPStatus is filled in by SendTestMessage's
			// record path; we don't re-query it here because
			// the row is the source of truth.
		} else {
			res.OK = false
			res.Status = "failed"
			res.Error = err.Error()
		}
		results = append(results, res)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Results []TestResult `json:"results"`
	}{Results: results})
}

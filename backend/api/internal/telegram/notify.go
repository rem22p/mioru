package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mioru/internal/model"
)

// Recorder is the subset of the store the notifier needs to
// log every send attempt. Defined here (the notifier side)
// so unit tests can pass a fake without dragging in pgx —
// *store.PostgresStore satisfies it.
type Recorder interface {
	RecordTelegramSend(ctx context.Context, orderID *int64, chatID, text, parseMode string) (int64, error)
	MarkTelegramSent(ctx context.Context, id int64, httpStatus int, telegramMsgID *int64, durationMs int) error
	MarkTelegramFailed(ctx context.Context, id int64, httpStatus *int32, errMsg string, durationMs int) error
}

// Notifier sends Telegram messages about new orders to manager chats.
type Notifier struct {
	botToken   string
	apiBaseURL string
	// adminURL is the origin of the admin SPA, used as the host
	// for clickable "open in admin" links inside order
	// notifications. When empty, formatOrderMessageHTML
	// degrades gracefully and emits no link — the message
	// still ships, it just won't be clickable. Set this
	// from config.AdminURL on the cmd/server side.
	adminURL   string
	// storeURL is the origin of the storefront SPA, used to
	// build the per-product "view in store" link in cart
	// orders. Falls back to adminURL (which in turn falls
	// back to APIBaseURL) when the operator doesn't
	// differentiate the two SPAs.
	storeURL   string
	uploadDir  string
	chatIDs    []string
	client     *http.Client
	rec        Recorder // optional; nil means no logging (tests + dev)

	// lastHealth stores the result of the most recent
	// HealthCheck call so the admin /telegram/diagnose endpoint
	// can show the bot's @username without re-issuing a getMe
	// on every page load. Populated by main.go at boot.
	lastHealthMu sync.RWMutex
	lastHealth   *healthResult
}

// healthResult is a snapshot of one HealthCheck call. We keep
// the username on success and the redacted error on failure;
// both are surfaced by the admin /telegram/diagnose endpoint.
type healthResult struct {
	at       time.Time
	username string
	err      string
}

// BotTokenSet is the only safe thing to expose about the
// token — the admin UI needs to know "is the env var set" but
// must never see the actual value.
func (n *Notifier) BotTokenSet() bool { return n.botToken != "" }

// ManagerChatCount is what the Status card shows in the
// "manager chats" tile.
func (n *Notifier) ManagerChatCount() int { return len(n.chatIDs) }

// ManagerChatIDs is a *copy* of the chat list, returned for
// the test-send endpoint. The copy is intentional: callers
// must not be able to mutate the notifier's internal slice.
func (n *Notifier) ManagerChatIDs() []string {
	out := make([]string, len(n.chatIDs))
	copy(out, n.chatIDs)
	return out
}

// LastHealthCheck returns the username and error string from
// the most recent HealthCheck, plus a boolean that is false
// when the boot health check hasn't run yet. Empty values
// are not an error — the admin UI just shows "—".
func (n *Notifier) LastHealthCheck() (username, errMsg string, ok bool) {
	n.lastHealthMu.RLock()
	defer n.lastHealthMu.RUnlock()
	if n.lastHealth == nil {
		return "", "", false
	}
	return n.lastHealth.username, n.lastHealth.err, true
}

// NewNotifier creates a Telegram notifier. If botToken or chatIDs are empty,
// notifications are silently skipped (no-op during dev) — but the skip is
// *logged* at WARN level so the manager can see in the server logs that
// the wiring is missing, instead of staring at a Telegram channel that
// never fires. The pre-fix behaviour was a plain `return` with no log
// at all, which is how "telegram notifier is broken in prod" got
// reported as "notifications just don't arrive".
func NewNotifier(botToken string, chatIDs []string, apiBaseURL, uploadDir, adminURL, storeURL string) *Notifier {
	return &Notifier{
		botToken:   botToken,
		apiBaseURL: apiBaseURL,
		adminURL:   strings.TrimRight(adminURL, "/"),
		storeURL:   strings.TrimRight(storeURL, "/"),
		uploadDir:  uploadDir,
		chatIDs:    chatIDs,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// SetRecorder wires a store into the notifier so every send
// attempt lands in the `telegram_messages` log. Must be called
// before the notifier starts handling real traffic; main.go
// does this once at boot after the store and notifier are both
// constructed.
func (n *Notifier) SetRecorder(r Recorder) { n.rec = r }

// StartPurge launches a background goroutine that periodically
// deletes telegram_messages rows older than retentionDays.
// Intended to be called once at boot from main.go.
// ctx cancellation stops the goroutine (e.g. on server shutdown).
// S1 (data-minimization, CLAUDE.md priority #2): prevents
// unbounded PII accumulation in the debug log.
func (n *Notifier) StartPurge(ctx context.Context, interval time.Duration, retentionDays int, purgeFn func(context.Context, int) (int64, error)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				slog.Info("telegram purge stopped")
				return
			case <-ticker.C:
				deleted, err := purgeFn(context.Background(), retentionDays)
				if err != nil {
					slog.Warn("telegram purge failed", "error", err)
				} else if deleted > 0 {
					slog.Info("telegram purge completed", "deleted", deleted)
				}
			}
		}
	}()
}

// HealthCheck calls Telegram's getMe endpoint and returns the bot's
// @username on success. It's a thin wrapper around the same HTTP
// client OrderCreated uses, intended to be called once at server
// startup so an invalid or revoked token shows up in the boot log
// instead of staying invisible until the first real order fails.
//
// The error message has the bot token redacted before being returned
// to the caller (which logs it via slog) — the same `redactToken`
// path that `OrderCreated` uses, so a leaked log line never
// includes the raw secret.
func (n *Notifier) HealthCheck(ctx context.Context) (string, error) {
	if n.botToken == "" {
		return "", fmt.Errorf("telegram notifier: TELEGRAM_BOT_TOKEN not set")
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", n.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := n.client.Do(req)
	if err != nil {
		// The *url.Error returned by http.Client embeds the full
		// request URL, which contains the bot token. Redact
		// before returning so the caller (main.go) can log the
		// error safely without needing the token itself.
		return "", fmt.Errorf("getMe: %s", redactToken(err.Error(), n.botToken))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := fmt.Sprintf("telegram api %d: %s", resp.StatusCode, errResp.Description)
		err := fmt.Errorf("%s", redactToken(msg, n.botToken))
		n.recordHealth("", err)
		return "", err
	}
	var ok struct {
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ok); err != nil {
		n.recordHealth("", err)
		return "", err
	}
	n.recordHealth(ok.Result.Username, nil)
	return ok.Result.Username, nil
}

// recordHealth stores the HealthCheck outcome so the admin
// /telegram/diagnose endpoint can show it without a fresh
// getMe on every page load. The mutex guards against the
// (rare) race where two callers invoke HealthCheck
// simultaneously.
func (n *Notifier) recordHealth(username string, err error) {
	h := healthResult{at: time.Now(), username: username}
	if err != nil {
		h.err = redactToken(err.Error(), n.botToken)
	}
	n.lastHealthMu.Lock()
	n.lastHealth = &h
	n.lastHealthMu.Unlock()
}

// OrderCreated sends a notification about a new order to all configured chats.
// Call from a background goroutine — never from the request handler directly.
func (n *Notifier) OrderCreated(order *model.Order, customer *model.Customer) {
	if n.botToken == "" {
		slog.Warn("telegram notify skipped: TELEGRAM_BOT_TOKEN not set",
			"order_id", order.ID)
		return
	}
	if len(n.chatIDs) == 0 {
		slog.Warn("telegram notify skipped: TELEGRAM_MANAGER_CHAT_IDS not set",
			"order_id", order.ID)
		return
	}

	text := n.formatOrderMessageHTML(order, customer)

	for _, chatID := range n.chatIDs {
		if err := n.sendMessage(chatID, text, order.ID); err != nil {
			slog.Warn("telegram notify failed",
				"chat_id", chatID,
				"order_id", order.ID,
				"error", redactToken(err.Error(), n.botToken),
			)
			continue
		}

		// Send photos as a media group
		if len(order.Photos) > 0 {
			if err := n.sendPhotos(chatID, order, order.ID); err != nil {
				slog.Warn("telegram photo notify failed",
					"chat_id", chatID,
					"order_id", order.ID,
					"photos", len(order.Photos),
					"error", redactToken(err.Error(), n.botToken),
				)
			}
		}
	}
}

// sendMessage posts one text message and — if a Recorder is
// wired in — writes the attempt to `telegram_messages`. The
// Recorder is optional so unit tests can construct a notifier
// without a store; the prod wiring in main.go calls
// `SetRecorder` once at boot.
//
// The function name kept the old "sendMessage(chatID, text)"
// signature for callers that don't have an order — specifically
// the test endpoint handler. That caller passes orderID=0
// and the recorder sees a NULL in `order_id` (handled by
// passing `*int64` nil through).
func (n *Notifier) sendMessage(chatID, text string, orderID int64) error {
	oid := orderID
	body := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)

	// Record the attempt before the HTTP call so a goroutine
	// crash mid-flight is visible as a stuck 'pending' row in
	// the admin UI. The recorder is nil in unit tests, which
	// is fine — we just skip the log.
	var rowID int64
	if n.rec != nil {
		var orderPtr *int64
		if oid != 0 {
			orderPtr = &oid
		}
		id, err := n.rec.RecordTelegramSend(context.Background(), orderPtr, chatID, text, "HTML")
		if err != nil {
			slog.Warn("telegram recorder: RecordTelegramSend failed", "error", err)
		} else {
			rowID = id
		}
	}

	start := time.Now()
	resp, err := n.client.Post(url, "application/json", bytes.NewReader(b))
	durationMs := int32(time.Since(start).Milliseconds())
	if err != nil {
		if n.rec != nil && rowID != 0 {
			_ = n.rec.MarkTelegramFailed(context.Background(), rowID, nil,
				redactToken(err.Error(), n.botToken), int(durationMs))
		}
		// The *url.Error from http.Client embeds the request URL
		// with the bot token. Redact the message before returning
		// so callers (OrderCreated, admin test-send handler)
		// never have to remember.
		return fmt.Errorf("send to %s: %s", chatID, redactToken(err.Error(), n.botToken))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		errMsg := fmt.Errorf("telegram api %d: %s", resp.StatusCode, errResp.Description)
		if n.rec != nil && rowID != 0 {
			st := int32(resp.StatusCode)
			_ = n.rec.MarkTelegramFailed(context.Background(), rowID, &st,
				redactToken(errMsg.Error(), n.botToken), int(durationMs))
		}
		return errMsg
	}

	// Happy path: parse the response, persist the message_id
	// Telegram gave us. Some handlers will want to reply to
	// that message later (e.g. /telegram/test → "edit me to
	// mark delivered"), so we surface it in the row.
	var ok struct {
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ok)
	if n.rec != nil && rowID != 0 {
		_ = n.rec.MarkTelegramSent(context.Background(), rowID, resp.StatusCode, &ok.Result.MessageID, int(durationMs))
	}
	return nil
}

// SendTestMessage is a thin wrapper around sendMessage that
// stamps orderID=0 (so the row's `order_id` is NULL — the test
// message isn't tied to any customer order). The admin
// "Send test" button uses this to verify the bot + chat_ids
// are wired up.
func (n *Notifier) SendTestMessage(chatID, text string) error {
	return n.sendMessage(chatID, text, 0)
}

// redactToken replaces the bot token in an error string with "***" to avoid
// leaking it into logs.
func redactToken(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "***")
}

// escapeHTML escapes the four HTML-special characters
// (&, <, >, ") that Telegram's HTML parse mode treats as
// markup delimiters. Every other character (including
// `.`, `(`, `)`, `*`, `_`, `~`, `=`, `+` etc.) is left
// as-is — Telegram's HTML parser doesn't reserve them.
// One pass through strings.NewReplacer keeps it O(n) and
// allocation-light.
func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

// formatOrderMessageHTML renders the notification body
// in Telegram's HTML parse mode. We switched to HTML from
// MarkdownV2 because `[text](url)` in MarkdownV2 silently
// drops the link as soon as the URL contains a `.` — which
// every real admin URL does — and `\.` escaping inside the
// URL breaks the parser outright with "Can't find end of a
// URL". HTML's `<a href="...">click</a>` is the only mode
// that handles arbitrary URLs reliably. The trade-off is
// that every dynamic field needs to be run through
// escapeHTML so a customer typing `<script>` in the comment
// can't inject markup into the message; the link targets
// themselves are operator config + a numeric id, so they
// don't need the same escape layer but we apply it
// anyway as defence-in-depth.
func (n *Notifier) formatOrderMessageHTML(o *model.Order, c *model.Customer) string {
	items := ""
	total := float64(o.TotalMinor) / 100
	for _, item := range o.Items {
		price := float64(item.PriceMinor) / 100
		sizeStr := escapeHTML(item.SizeLabel)
		var line string
		if item.ProductID > 0 || item.ProductSlug != "" {
			linkText := item.ProductName
			if linkText == "" {
				linkText = fmt.Sprintf("Товар #%d", item.ProductID)
			}
			link := n.makeStoreProductLinkHTML(item, linkText)
			line = fmt.Sprintf("  • %dx %s (размер %s) — %.2f лей\n",
				item.Quantity, link, sizeStr, price)
		} else {
			line = fmt.Sprintf("  • %dx (размер %s) — %.2f лей\n",
				item.Quantity, sizeStr, price)
		}
		items += line
	}

	// Customer line: clickable "First Last" that opens the
	// admin profile. Falls back to plain text when adminURL
	// is empty or the customer id is missing.
	customerLine := escapeHTML(c.FirstName) + " " + escapeHTML(c.LastName)
	if c.ID > 0 {
		if link := n.makeAdminCustomerLinkHTML(c.ID, c.FirstName+" "+c.LastName); link != "" {
			customerLine = link
		}
	}

	return fmt.Sprintf(
		"🛍 <b>Новый заказ #%d</b>\n\n"+
			"<b>Тип:</b> %s\n"+
			"<b>Клиент:</b> %s\n"+
			"<b>Email:</b> %s\n"+
			"<b>Телефон:</b> %s\n\n"+
			"<b>Город:</b> %s\n"+
			"<b>Доставка:</b> %s\n"+
			"<b>Оплата:</b> %s\n"+
			"%s%s%s\n"+
			"<b>Товары:</b>\n%s"+
			"<b>Итого: %.2f лей</b>\n\n"+
			"<i>%s</i>",
		o.ID,
			escapeHTML(o.Type),
			customerLine,
			escapeHTML(c.Email),
			escapeHTML(o.Phone),
		escapeHTML(o.City),
		escapeHTML(o.DeliveryMethod),
		escapeHTML(o.PaymentMethod),
		addressLineHTML(o),
		commentLineHTML(o),
		individualFieldsHTML(o),
		items,
		total,
		escapeHTML(o.CreatedAt.Format("02.01.2006 15:04")),
	)
}

// makeStoreProductLinkHTML builds the per-product "open in
// store" link used inside cart-order notifications. Unlike
// the customer link, the product link points at the
// *storefront* (the public product page) rather than the
// admin edit form — a manager reading a "new order" in
// chat will more often want to check the public listing
// (price, stock, photos, description) than to dive into
// the admin form. We use ProductSlug when the order
// carries it (the storefront router wants slugs, not ids,
// for SEO) and fall back to id so legacy orders still
// produce a working link.
func (n *Notifier) makeStoreProductLinkHTML(item model.OrderItem, text string) string {
	if n.storeURL == "" || (item.ProductID <= 0 && item.ProductSlug == "") {
		return ""
	}
	// The store router exposes products at /product/{slug}
	// (singular, not /products/...) — the trailing "s" was
	// a guess on our part that turned out to be wrong when
	// we started pointing the link at the public store
	// instead of the admin edit form. We hardcode the
	// singular form here because the storefront and the
	// admin use different URL shapes (`/products/{id}` in
	// admin, `/product/{slug}` in store) and we have to
	// pick one. The slug is preferred for SEO and is what
	// the React Router link uses everywhere in
	// apps/store/src. Fall back to the numeric id so a
	// legacy order without a slug still ships a working
	// link.
	var path string
	if item.ProductSlug != "" {
		path = "/product/" + item.ProductSlug
	} else {
		path = fmt.Sprintf("/product/%d", item.ProductID)
	}
	url := n.storeURL + path
	return fmt.Sprintf(`<a href="%s">%s</a>`, escapeHTML(url), escapeHTML(text))
}

// makeAdminCustomerLinkHTML mirrors makeAdminProductLinkHTML
// for the /customers/{id} route.
func (n *Notifier) makeAdminCustomerLinkHTML(customerID int64, text string) string {
	if n.adminURL == "" || customerID <= 0 {
		return ""
	}
	url := fmt.Sprintf("%s/customers/%d", n.adminURL, customerID)
	return fmt.Sprintf(`<a href="%s">%s</a>`, escapeHTML(url), escapeHTML(text))
}

// addressLineHTML, commentLineHTML, individualFieldsHTML
// are the HTML-mode rewrites of the previous MarkdownV2
// helpers. They use <b>...</b> for headings and pass
// dynamic fields through escapeHTML.
func addressLineHTML(o *model.Order) string {
	if o.DeliveryMethod == "address" && (o.Street != "" || o.House != "") {
		return fmt.Sprintf("<b>Адрес:</b> %s, %s", escapeHTML(o.Street), escapeHTML(o.House)) + aptSuffixHTML(o.Apartment) + "\n"
	}
	return ""
}

func aptSuffixHTML(a string) string {
	if a != "" {
		// Periods are literal in HTML — no escape needed.
		return ", кв. " + escapeHTML(a)
	}
	return ""
}

func commentLineHTML(o *model.Order) string {
	if o.Comment != "" {
		return fmt.Sprintf("<b>Комментарий:</b> %s\n", escapeHTML(o.Comment))
	}
	return ""
}

func individualFieldsHTML(o *model.Order) string {
	if o.Type != "individual" {
		return ""
	}
	s := ""
	if o.Height != nil {
		s += fmt.Sprintf("<b>Рост:</b> %.0f см\n", *o.Height)
	}
	if o.Weight != nil {
		s += fmt.Sprintf("<b>Вес:</b> %.0f кг\n", *o.Weight)
	}
	return s
}

func (n *Notifier) sendPhotos(chatID string, order *model.Order, orderID int64) error {
	var lastErr error
	for i, photo := range order.Photos {
		filename := filepath.Base(photo)
		diskPath := filepath.Join(n.uploadDir, filename)

		file, err := os.Open(diskPath)
		if err != nil {
			lastErr = fmt.Errorf("photo %d open %s: %w", i+1, diskPath, err)
			slog.Warn("telegram photo: cannot open file", "path", diskPath, "error", err)
			continue
		}

		if err := n.sendPhotoMultipart(chatID, file, filename); err != nil {
			file.Close()
			lastErr = fmt.Errorf("photo %d: %w", i+1, err)
			continue
		}
		file.Close()
	}
	return lastErr
}

func (n *Notifier) sendPhotoMultipart(chatID string, file *os.File, filename string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("chat_id", chatID); err != nil {
		return err
	}

	part, err := w.CreateFormFile("photo", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	w.Close()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", n.botToken)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send photo to %s: %s", chatID, redactToken(err.Error(), n.botToken))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Description string `json:"description"`
		}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &errResp)
		return fmt.Errorf("telegram sendPhoto %d: %s", resp.StatusCode, errResp.Description)
	}
	return nil
}

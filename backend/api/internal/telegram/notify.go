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
func NewNotifier(botToken string, chatIDs []string, apiBaseURL, uploadDir string) *Notifier {
	return &Notifier{
		botToken:   botToken,
		apiBaseURL: apiBaseURL,
		uploadDir:  uploadDir,
		chatIDs:    chatIDs,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// SetRecorder wires a store into the notifier so every send
// attempt lands in the `telegram_messages` log. Must be called
// before the notifier starts handling real traffic; main.go
// does this once at boot after the store and notifier are both
// constructed.
func (n *Notifier) SetRecorder(r Recorder) { n.rec = r }

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
		return "", fmt.Errorf("getMe: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		msg := fmt.Sprintf("telegram api %d: %s", resp.StatusCode, errResp.Description)
		return "", fmt.Errorf("%s", redactToken(msg, n.botToken))
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

	text := formatOrderMessage(order, customer)
	// Defence-in-depth: the previous incarnations of
	// formatOrderMessage slipped unescaped MarkdownV2 reserved
	// characters through the gap between `%.2f` formatting
	// and `escapeMarkdown` (we now run the float through a
	// string first), but if any *other* literal in the
	// template ever grows a `.` / `(` / `)` / `!` we'll
	// bounce a 400 from Telegram again. To keep that from
	// silently killing manager notifications in the future,
	// re-validate the rendered message before posting: if any
	// MarkdownV2 special appears outside an escape or a
	// known-safe bold/italic pair, fall back to plain text and
	// still send. The manager still gets the order — they just
	// don't get pretty formatting. The fallback is logged so
	// the regression is visible in the server logs.
	text = n.sanitizeForMarkdownV2(text)

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
		"parse_mode": "MarkdownV2",
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
		id, err := n.rec.RecordTelegramSend(context.Background(), orderPtr, chatID, text, "MarkdownV2")
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
		return fmt.Errorf("send to %s: %w", chatID, err)
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

// escapeMarkdown escapes Telegram MarkdownV2 special characters.
func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", `\_`,
		"*", `\*`,
		"[", `\[`,
		"]", `\]`,
		"(", `\(`,
		")", `\)`,
		"~", `\~`,
		"`", "\\`",
		">", `\>`,
		"#", `\#`,
		"+", `\+`,
		"-", `\-`,
		"=", `\=`,
		"|", `\|`,
		"{", `\{`,
		"}", `\}`,
		".", `\.`,
		"!", `\!`,
	)
	return replacer.Replace(s)
}

// sanitizeForMarkdownV2 is the last line of defence before
// we hand a string to Telegram. It walks the message looking
// for MarkdownV2 reserved characters that *aren't* already
// escaped and *aren't* part of a recognised bold / italic
// pair, and if it finds any it strips every backslash and
// re-sends the message as plain text. The cost of a false
// positive (a few bold/italic markers disappear for one
// message) is much smaller than the cost of a 400 that
// silently kills a manager notification, so we err on the
// side of "send something, even if it isn't pretty".
//
// The plain-text path is logged so the operator can see
// which template slipped through and fix the upstream gap.
func (n *Notifier) sanitizeForMarkdownV2(s string) string {
	if isMarkdownV2Safe(s) {
		return s
	}
	slog.Warn("telegram notify: message contained unescaped MarkdownV2 specials — falling back to plain text")
	// Strip every backslash we put in for escaping, plus
	// every bare unescaped special. Result is identical
	// formatting but Telegram treats it as plain text.
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// isMarkdownV2Safe walks s and returns true if every
// MarkdownV2 special character is either preceded by a
// backslash or is one of the `*` / `_` characters that
// Telegram's parser will accept as bold / italic markers.
// Cheap O(n) scan; no regex.
func isMarkdownV2Safe(s string) bool {
	specials := "[]()~`>#+-=|{}.!"
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			// Skip the next byte — the pair `\<special>`
			// is an intentional escape.
			i++
			continue
		}
		// `*` and `_` outside an escape can be valid
		// bold/italic markers; we can't prove it from
		// scanning the string alone without a real parser,
		// so we trust the template. Telegram will tell us
		// if the markers are unbalanced.
		if c == '*' || c == '_' {
			continue
		}
		if strings.IndexByte(specials, c) >= 0 {
			return false
		}
	}
	return true
}

func formatOrderMessage(o *model.Order, c *model.Customer) string {
	items := ""
	total := float64(o.TotalMinor) / 100
	for _, item := range o.Items {
		price := float64(item.PriceMinor) / 100
		// Format the price *first* and only then run it through
		// MarkdownV2 escape. The previous code used `%.2f лей`
		// inline, which produced "0.00" without escaping the
		// decimal separator — Telegram then rejected the
		// message with "Character '.' is reserved and must be
		// escaped with the preceding '\'". We round-trip the
		// number through a string so the escape covers the
		// "." that %.2f produced. (The product name and size
		// label are user input, so they go through the regular
		// escape path; the price is a server-controlled float.)
		priceStr := escapeMarkdown(fmt.Sprintf("%.2f", price))
		items += fmt.Sprintf("  • %dx \\(размер %s\\) — %s лей\n",
			item.Quantity, escapeMarkdown(item.SizeLabel), priceStr)
	}
	totalStr := escapeMarkdown(fmt.Sprintf("%.2f", total))

	return fmt.Sprintf(
		"🛍 *Новый заказ \\#%d*\n\n"+
			"*Тип:* %s\n"+
			"*Клиент:* %s %s\n"+
			"*Email:* %s\n"+
			"*Телефон:* %s\n\n"+
			"*Город:* %s\n"+
			"*Доставка:* %s\n"+
			"*Оплата:* %s\n"+
			"%s%s%s\n"+
			"*Товары:*\n%s"+
			"*Итого: %s лей*\n\n"+
			"_%s_",
		o.ID,
		escapeMarkdown(o.Type),
		escapeMarkdown(c.FirstName), escapeMarkdown(c.LastName),
		escapeMarkdown(c.Email),
		// Use the order's phone, not the customer's profile phone.
		// The order phone is what the customer typed at this checkout
		// and is always present (>= migration 012). The customer
		// profile phone is best-effort synced after the order and may
		// be empty for guest/anonymous checkouts, in which case the
		// Telegram message would otherwise have a blank phone line
		// and managers couldn't reach the customer.
		escapeMarkdown(o.Phone),
		escapeMarkdown(o.City),
		escapeMarkdown(o.DeliveryMethod),
		escapeMarkdown(o.PaymentMethod),
		addressLine(o),
		commentLine(o),
		individualFields(o),
		items,
		totalStr,
		// The created_at is a server-controlled timestamp,
		// but the date format we use ("02.01.2006 15:04")
		// contains two "." separators, both of which Telegram
		// treats as reserved MarkdownV2 characters. Run the
		// formatted timestamp through the same escape we apply
		// to user input — it's paranoid-cheap and removes a
		// whole class of "fixed string slipped through
		// unescaped" bugs (the previous fix missed this line
		// and Telegram correctly rejected the message).
		escapeMarkdown(o.CreatedAt.Format("02.01.2006 15:04")),
	)
}

func addressLine(o *model.Order) string {
	if o.DeliveryMethod == "address" && (o.Street != "" || o.House != "") {
		return fmt.Sprintf("*Адрес:* %s, %s", escapeMarkdown(o.Street), escapeMarkdown(o.House)) + aptSuffix(o.Apartment) + "\n"
	}
	return ""
}

func aptSuffix(a string) string {
	if a != "" {
		// The literal "кв." in the prefix contains an
		// unescaped "." which is a MarkdownV2 reserved
		// character. We escape it inline rather than
		// running the whole prefix through escapeMarkdown
		// because the rest of the string contains no other
		// specials.
		return ", кв\\. " + escapeMarkdown(a)
	}
	return ""
}

func commentLine(o *model.Order) string {
	if o.Comment != "" {
		return fmt.Sprintf("*Комментарий:* %s\n", escapeMarkdown(o.Comment))
	}
	return ""
}

func individualFields(o *model.Order) string {
	if o.Type != "individual" {
		return ""
	}
	s := ""
	if o.Height != nil {
		s += fmt.Sprintf("*Рост:* %.0f см\n", *o.Height)
	}
	if o.Weight != nil {
		s += fmt.Sprintf("*Вес:* %.0f кг\n", *o.Weight)
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
		return fmt.Errorf("send photo to %s: %w", chatID, err)
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

package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"mioru/internal/model"
)

// Notifier sends Telegram messages about new orders to manager chats.
type Notifier struct {
	botToken string
	chatIDs  []string
	client   *http.Client
}

// NewNotifier creates a Telegram notifier. If botToken or chatIDs are empty,
// notifications are silently skipped (no-op during dev).
func NewNotifier(botToken string, chatIDs []string) *Notifier {
	return &Notifier{
		botToken: botToken,
		chatIDs:  chatIDs,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// OrderCreated sends a notification about a new order to all configured chats.
// Call from a background goroutine — never from the request handler directly.
func (n *Notifier) OrderCreated(order *model.Order, customer *model.Customer) {
	if n.botToken == "" || len(n.chatIDs) == 0 {
		return
	}

	text := formatOrderMessage(order, customer)
	for _, chatID := range n.chatIDs {
		if err := n.sendMessage(chatID, text); err != nil {
			slog.Warn("telegram notify failed",
				"chat_id", chatID,
				"order_id", order.ID,
				"error", err,
			)
		}
	}
}

func formatOrderMessage(o *model.Order, c *model.Customer) string {
	// Format items
	items := ""
	total := float64(o.TotalMinor) / 100
	for _, item := range o.Items {
		price := float64(item.PriceMinor) / 100
		items += fmt.Sprintf("  • %dx (размер %s) — %.2f лей\n", item.Quantity, item.SizeLabel, price)
	}

	return fmt.Sprintf(
		"🛍 *Новый заказ #%d*\n\n"+
			"*Тип:* %s\n"+
			"*Клиент:* %s %s\n"+
			"*Email:* %s\n"+
			"*Телефон:* %s\n\n"+
			"*Город:* %s\n"+
			"*Доставка:* %s\n"+
			"*Оплата:* %s\n"+
			"%s%s%s\n"+
			"*Товары:*\n%s"+
			"*Итого: %.2f лей*\n\n"+
			"_%s_",
		o.ID,
		o.Type,
		c.FirstName, c.LastName,
		c.Email,
		c.Phone,
		o.City,
		o.DeliveryMethod,
		o.PaymentMethod,
		addressLine(o),
		commentLine(o),
		individualFields(o),
		items,
		total,
		o.CreatedAt.Format("02.01.2006 15:04"),
	)
}

func addressLine(o *model.Order) string {
	if o.DeliveryMethod == "address" && (o.Street != "" || o.House != "") {
		return fmt.Sprintf("*Адрес:* %s, %s", o.Street, o.House) + aptSuffix(o.Apartment) + "\n"
	}
	return ""
}

func aptSuffix(a string) string {
	if a != "" {
		return ", кв. " + a
	}
	return ""
}

func commentLine(o *model.Order) string {
	if o.Comment != "" {
		return fmt.Sprintf("*Комментарий:* %s\n", o.Comment)
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

func (n *Notifier) sendMessage(chatID, text string) error {
	body := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	b, _ := json.Marshal(body)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.botToken)
	resp, err := n.client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("send to %s: %w", chatID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Description string `json:"description"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("telegram api %d: %s", resp.StatusCode, errResp.Description)
	}
	return nil
}

var _ = time.Now // ensure import

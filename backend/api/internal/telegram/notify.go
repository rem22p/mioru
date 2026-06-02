package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"mioru/internal/model"
)

// Notifier sends Telegram messages about new orders to manager chats.
type Notifier struct {
	botToken   string
	apiBaseURL string
	uploadDir  string
	chatIDs    []string
	client     *http.Client
}

// NewNotifier creates a Telegram notifier. If botToken or chatIDs are empty,
// notifications are silently skipped (no-op during dev).
func NewNotifier(botToken string, chatIDs []string, apiBaseURL, uploadDir string) *Notifier {
	return &Notifier{
		botToken:   botToken,
		apiBaseURL: apiBaseURL,
		uploadDir:  uploadDir,
		chatIDs:    chatIDs,
		client:     &http.Client{Timeout: 10 * time.Second},
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
			continue
		}

		// Send photos as a media group
		if len(order.Photos) > 0 {
			if err := n.sendPhotos(chatID, order); err != nil {
				slog.Warn("telegram photo notify failed",
					"chat_id", chatID,
					"order_id", order.ID,
					"photos", len(order.Photos),
					"error", err,
				)
			}
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

// sendPhotos sends each photo as a multipart file upload to Telegram.
// Telegram's sendPhoto by URL fails for api.mioru.store (likely network
// restriction), so we read the file from disk and send it directly.
func (n *Notifier) sendPhotos(chatID string, order *model.Order) error {
	var lastErr error
	for i, photo := range order.Photos {
		// photo is like "/uploads/xxx.png"
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

// sendPhotoMultipart sends a single photo to Telegram using multipart/form-data.
func (n *Notifier) sendPhotoMultipart(chatID string, file *os.File, filename string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// chat_id field
	if err := w.WriteField("chat_id", chatID); err != nil {
		return err
	}

	// photo file part
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

var _ = time.Now // ensure import

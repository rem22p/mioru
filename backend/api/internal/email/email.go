package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Service handles email sending via Resend HTTP API.
type Service struct {
	Sender  string
	BaseURL string
	apiKey  string
}

// NewService reads its config from the environment: APP_BASE_URL (base for the
// reset link — the admin app hosts /reset, so it defaults to the admin dev
// port), EMAIL_SENDER (the From address), and RESEND_API_KEY (the Resend API
// key; when empty, emails are logged instead of sent).
func NewService() *Service {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5174"
	}
	sender := os.Getenv("EMAIL_SENDER")
	if sender == "" {
		sender = "onboarding@resend.dev"
	}
	return &Service{
		Sender:  sender,
		BaseURL: baseURL,
		apiKey:  os.Getenv("RESEND_API_KEY"),
	}
}

// SendPasswordReset sends a password reset email.
func (s *Service) SendPasswordReset(to, token string) {
	resetURL := fmt.Sprintf("%s/reset/%s", s.BaseURL, token)
	s.sendResend(to, "Сброс пароля — Mioru", fmt.Sprintf(
		"Для сброса пароля перейдите по ссылке:\n%s\n\nСсылка действительна 1 час.\nЕсли вы не запрашивали сброс — проигнорируйте это письмо.",
		resetURL,
	))
}

func (s *Service) sendResend(to, subject, text string) {
	if s.apiKey == "" {
		log.Printf("[EMAIL] No API key, reset link: %s/reset/...", s.BaseURL)
		return
	}

	body := map[string]interface{}{
		"from":    s.Sender,
		"to":      []string{to},
		"subject": subject,
		"text":    text,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(data))
	if err != nil {
		log.Printf("[EMAIL] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[EMAIL] Failed to send: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("[EMAIL] Sent to %s via Resend API", to)
	} else {
		log.Printf("[EMAIL] Resend error %d: %s", resp.StatusCode, string(respBody))
	}
}

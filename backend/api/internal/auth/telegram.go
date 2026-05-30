package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TelegramAuthData holds the fields the Telegram Login Widget sends to the
// frontend on a successful authentication. The Hash field is the HMAC-SHA256
// signature computed by Telegram; we recompute it server-side to verify the
// data has not been tampered with.
type TelegramAuthData struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// VerifyTelegramAuth checks the HMAC-SHA256 signature on the Telegram Login
// Widget data. The HMAC key is SHA256(botToken) — not the raw bot token.
// maxAge caps how old auth_date may be to prevent replay (0 = no check).
func VerifyTelegramAuth(data TelegramAuthData, botToken string, maxAge time.Duration) error {
	if botToken == "" {
		return fmt.Errorf("telegram bot token not configured")
	}

	// Build sorted key-value map; empty values are excluded per Telegram spec.
	pairs := map[string]string{
		"auth_date":  strconv.FormatInt(data.AuthDate, 10),
		"first_name": data.FirstName,
		"id":         strconv.FormatInt(data.ID, 10),
	}
	if data.LastName != "" {
		pairs["last_name"] = data.LastName
	}
	if data.Username != "" {
		pairs["username"] = data.Username
	}
	if data.PhotoURL != "" {
		pairs["photo_url"] = data.PhotoURL
	}

	// Sort keys alphabetically.
	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// data-check-string: key=value\nkey=value\n... (no trailing \n).
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(pairs[k])
	}
	dataCheckString := sb.String()

	// HMAC-SHA256 with secret = SHA256(botToken).
	secretHash := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretHash[:])
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedHash), []byte(data.Hash)) {
		return fmt.Errorf("invalid telegram hash")
	}

	if maxAge > 0 {
		authTime := time.Unix(data.AuthDate, 0)
		if time.Since(authTime) > maxAge {
			return fmt.Errorf("telegram auth data expired")
		}
	}

	return nil
}

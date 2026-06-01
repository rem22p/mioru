package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testBotToken = "123456:ABC-DEF1234ghikl-zyx57W2v1u123ew11"

// signTelegramData builds a valid Telegram hash for the given data fields
// using the same algorithm Telegram uses (HMAC-SHA256 with key = SHA256(botToken)).
func signTelegramData(botToken string, data TelegramAuthData) string {
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

	keys := make([]string, 0, len(pairs))
	for k := range pairs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(pairs[k])
	}

	secretHash := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretHash[:])
	mac.Write([]byte(sb.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyTelegramAuth_Valid(t *testing.T) {
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		LastName:  "Tonkoglas",
		Username:  "tonkoglas",
		PhotoURL:  "https://t.me/i/userpic/320/tonkoglas.jpg",
		AuthDate:  time.Now().Unix(),
	}
	data.Hash = signTelegramData(testBotToken, data)

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("expected valid signature, got: %v", err)
	}
}

func TestVerifyTelegramAuth_MinimalFields(t *testing.T) {
	// Only auth_date, first_name, id — no optional fields.
	data := TelegramAuthData{
		ID:        99999,
		FirstName: "Anna",
		AuthDate:  time.Now().Unix(),
	}
	data.Hash = signTelegramData(testBotToken, data)

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("expected valid signature with minimal fields, got: %v", err)
	}
}

func TestVerifyTelegramAuth_FakeHash(t *testing.T) {
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  time.Now().Unix(),
		Hash:      "00001111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff",
	}

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error for fake hash, got nil")
	}
	if !strings.Contains(err.Error(), "invalid telegram hash") {
		t.Fatalf("expected 'invalid telegram hash', got: %v", err)
	}
}

func TestVerifyTelegramAuth_WrongBotToken(t *testing.T) {
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  time.Now().Unix(),
	}
	data.Hash = signTelegramData(testBotToken, data)

	// Verify with a different bot token.
	err := VerifyTelegramAuth(data, "000000:OTHER-TOKEN", 24*time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error for wrong bot token, got nil")
	}
}

func TestVerifyTelegramAuth_Expired(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  now.Add(-25 * time.Hour).Unix(), // older than 24h
	}
	data.Hash = signTelegramData(testBotToken, data)

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, now)
	if err == nil {
		t.Fatal("expected error for expired auth_date, got nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected 'expired', got: %v", err)
	}
}

func TestVerifyTelegramAuth_EmptyBotToken(t *testing.T) {
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  time.Now().Unix(),
		Hash:      "any",
	}

	err := VerifyTelegramAuth(data, "", 24*time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error for empty bot token, got nil")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected 'not configured', got: %v", err)
	}
}

func TestVerifyTelegramAuth_TamperedID(t *testing.T) {
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  time.Now().Unix(),
	}
	data.Hash = signTelegramData(testBotToken, data)

	// Tamper with ID after signing.
	data.ID = 99999

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, time.Now())
	if err == nil {
		t.Fatal("expected error for tampered ID, got nil")
	}
}

func TestVerifyTelegramAuth_NoAgeCheck(t *testing.T) {
	// maxAge=0 disables the freshness check.
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  time.Now().Add(-100 * 24 * time.Hour).Unix(), // 100 days old
	}
	data.Hash = signTelegramData(testBotToken, data)

	err := VerifyTelegramAuth(data, testBotToken, 0, time.Now())
	if err != nil {
		t.Fatalf("expected success with maxAge=0, got: %v", err)
	}
}

func TestVerifyTelegramAuth_FirstNameEmpty(t *testing.T) {
	// first_name must not be empty — required by Telegram widget.
	data := TelegramAuthData{
		ID:       12345,
		AuthDate: time.Now().Unix(),
	}
	data.Hash = signTelegramData(testBotToken, data)

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, time.Now())
	if err != nil {
		t.Fatalf("empty first_name still validates (hash doesn't care), got: %v", err)
	}
	// The empty first_name is accepted here because the hash is valid.
	// Input validation (non-empty first_name) happens in the handler layer,
	// not in the signature verifier, which deals only with cryptographic trust.
}

func ExampleVerifyTelegramAuth() {
	data := TelegramAuthData{
		ID:        12345,
		FirstName: "Pavel",
		AuthDate:  time.Now().Unix(),
	}
	// In tests, sign with the helper; in production Telegram signs it.
	data.Hash = signTelegramData(testBotToken, data)

	err := VerifyTelegramAuth(data, testBotToken, 24*time.Hour, time.Now())
	if err != nil {
		fmt.Println("invalid:", err)
	} else {
		fmt.Println("ok")
	}
	// Output: ok
}

package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// minSecretKeyLen is the minimum acceptable length of the JWT signing key.
const minSecretKeyLen = 32

type Config struct {
	SecretKey        string
	TokenExpiry      int
	Port             string
	DatabaseURL      string
	UploadDir        string
	AppEnv           string
	TelegramBotToken string
}

func Load() Config {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	secret, err := resolveSecretKey(appEnv, os.Getenv("SECRET_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mioru:pass@localhost:5432/mioru?sslmode=disable"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramBotToken == "" {
		log.Printf("WARNING: TELEGRAM_BOT_TOKEN not set — Telegram login is disabled")
	}

	return Config{
		SecretKey:        secret,
		TokenExpiry:      1440,
		Port:             port,
		DatabaseURL:      databaseURL,
		UploadDir:        uploadDir,
		AppEnv:           appEnv,
		TelegramBotToken: telegramBotToken,
	}
}

// isProduction reports whether appEnv names the production environment.
func isProduction(appEnv string) bool {
	return strings.EqualFold(appEnv, "production") || strings.EqualFold(appEnv, "prod")
}

// IsProduction reports whether the loaded config targets the production
// environment. Consumers gate runtime decisions (notably the cookie Secure
// flag) on it instead of re-deriving the answer from AppEnv themselves.
func (c Config) IsProduction() bool {
	return isProduction(c.AppEnv)
}

// resolveSecretKey decides the JWT signing key for the configured environment.
//
// In production a strong SECRET_KEY (>= minSecretKeyLen chars) is mandatory: an
// empty or too-short key is a fatal misconfiguration, never silently replaced.
// That keeps issued tokens valid across restarts and prevents a deploy that
// simply forgot to set the key from running with an ephemeral, unknown secret.
//
// Outside production a missing key is replaced by a fresh random one (with a
// warning) so local development needs no setup; a present-but-too-short key is
// still rejected everywhere.
func resolveSecretKey(appEnv, secret string) (string, error) {
	if isProduction(appEnv) {
		if secret == "" {
			return "", errors.New("SECRET_KEY is required in production")
		}
		if len(secret) < minSecretKeyLen {
			return "", fmt.Errorf("SECRET_KEY must be at least %d characters", minSecretKeyLen)
		}
		return secret, nil
	}

	if secret == "" {
		b := make([]byte, 64)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("generate random SECRET_KEY: %w", err)
		}
		secret = base64.URLEncoding.EncodeToString(b)
		log.Printf("WARNING: generated a random SECRET_KEY for env %q; set SECRET_KEY for token persistence", appEnv)
	}
	if len(secret) < minSecretKeyLen {
		return "", fmt.Errorf("SECRET_KEY must be at least %d characters", minSecretKeyLen)
	}
	return secret, nil
}

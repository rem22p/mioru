package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
)

type Config struct {
	SecretKey   string
	TokenExpiry int
	RedisAddr   string
	RedisPW     string
	Port        string
	DatabaseURL string
	UploadDir   string
}

func Load() Config {
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		// Generate cryptographically secure random secret
		b := make([]byte, 64)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatal("failed to generate SECRET_KEY: ", err)
		}
		secret = base64.URLEncoding.EncodeToString(b)
		log.Printf("WARNING: Generated random SECRET_KEY. Set SECRET_KEY env var for persistence.")
	}
	if len(secret) < 32 {
		log.Fatal("SECRET_KEY must be at least 32 characters")
	}

	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = os.Getenv("REDIS_ADDR")
	}
	if addr == "" {
		addr = "localhost:6379"
	}
	pw := os.Getenv("REDIS_PASSWORD")
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

	return Config{
		SecretKey:   secret,
		TokenExpiry: 1440,
		RedisAddr:   addr,
		RedisPW:     pw,
		Port:        port,
		DatabaseURL: databaseURL,
		UploadDir:   uploadDir,
	}
}

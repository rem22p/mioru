package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"mioru/internal/config"
	"mioru/internal/email"
	"mioru/internal/handler"
	"mioru/internal/middleware"
	"mioru/internal/store"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Re-generate random SECRET_KEY if not set in .env
	if os.Getenv("SECRET_KEY") == "" {
		log.Println("WARNING: SECRET_KEY not set in .env, generating random key")
	}

	cfg := config.Load()

	s := store.New(cfg.RedisAddr, cfg.RedisPW)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	emailSvc := email.NewService("onboarding@resend.dev")

	authH := handler.NewAuthHandler(s, emailSvc, cfg.SecretKey, cfg.TokenExpiry)
	noteH := handler.NewNoteHandler(s)
	wsH := handler.NewWSHandler(s, cfg.SecretKey)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth (no middleware)
	mux.HandleFunc("POST /api/auth/register", cors(authH.Register))
	mux.HandleFunc("POST /api/auth/login", cors(authH.Login))
	mux.HandleFunc("POST /api/auth/forgot-password", cors(authH.ForgotPassword))
	mux.HandleFunc("POST /api/auth/reset-password", cors(authH.ResetPassword))

	// User profile (with auth)
	mux.Handle("GET /api/users/me", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(authH.Me)))
	mux.Handle("PUT /api/users/me/profile", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(authH.UpdateProfile)))
	mux.Handle("PUT /api/users/me/password", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(authH.ChangePassword)))

	// Notes (with auth)
	mux.Handle("GET /api/notes", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(noteH.List)))
	mux.Handle("POST /api/notes", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(noteH.Create)))
	mux.Handle("PUT /api/notes/{id}", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(noteH.Update)))
	mux.Handle("DELETE /api/notes/{id}", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(noteH.Delete)))

	// WebSocket
	mux.HandleFunc("/ws/notes", wsH.HandleNotes)

	// CORS preflight
	mux.HandleFunc("OPTIONS /api/", func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
	})

	// Start Redis broadcast goroutine
	go wsH.BroadcastRedis()

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, securityHeaders(mux)))
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w, r)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func corsHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	allowed := map[string]bool{
		"http://localhost:5173":      true,
		"http://127.0.0.1:5173":     true,
		"http://localhost:8080":      true,
		"http://127.0.0.1:8080":     true,
		"https://admin.mioru.store":  true,
		"https://www.admin.mioru.store": true,
	}
	if allowed[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src https://fonts.googleapis.com https://fonts.gstatic.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"mioru/internal/config"
	"mioru/internal/email"
	"mioru/internal/handler"
	"mioru/internal/middleware"
	"mioru/internal/model"
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

	// Redis store (for notes, WebSocket, reset tokens)
	redisStore := store.New(cfg.RedisAddr, cfg.RedisPW)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisStore.Ping(ctx); err != nil {
		log.Fatal("Redis connection failed:", err)
	}

	// PostgreSQL store (for users, products, categories)
	pgStore, err := store.NewPostgresStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("PostgreSQL initialization failed:", err)
	}
	defer pgStore.Close()

	// Auto-migrate: if PostgreSQL users table is empty and Redis has users, migrate them
	autoMigrateUsers(redisStore, pgStore)

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Printf("WARNING: Failed to create upload dir %s: %v", cfg.UploadDir, err)
	}

	emailSvc := email.NewService("onboarding@resend.dev")

	// Handlers
	authH := handler.NewAuthHandler(pgStore, redisStore, emailSvc, cfg.SecretKey, cfg.TokenExpiry)
	noteH := handler.NewNoteHandler(redisStore)
	wsH := handler.NewWSHandler(redisStore, cfg.SecretKey)
	productH := handler.NewProductHandler(pgStore, cfg.UploadDir)

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

	// Store: Public product & category endpoints (no auth)
	storeH := handler.NewStoreHandler(pgStore)
	mux.HandleFunc("GET /api/products", cors(storeH.ListProducts))
	mux.HandleFunc("GET /api/products/{slug}", cors(storeH.GetProduct))
	mux.HandleFunc("GET /api/categories", cors(storeH.ListCategories))

	// Admin: Categories
	mux.HandleFunc("GET /api/admin/categories", cors(productH.Categories))

	// Admin: Products (with auth)
	mux.Handle("GET /api/admin/products", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(productH.List)))
	mux.Handle("POST /api/admin/products", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(productH.Create)))
	mux.Handle("GET /api/admin/products/{slug}", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(productH.Get)))
	mux.Handle("PUT /api/admin/products/{slug}", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(productH.Update)))
	mux.Handle("DELETE /api/admin/products/{slug}", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(productH.Delete)))

	// Admin: Upload (with auth)
	mux.Handle("POST /api/admin/upload", middleware.AuthMW(cfg.SecretKey)(http.HandlerFunc(productH.Upload)))

	// Admin: Migration (with auth)
	mux.Handle("POST /api/admin/migrate-users", middleware.AuthMW(cfg.SecretKey)(handler.MigrateUsersHandler(redisStore, pgStore)))

	// Serve uploaded files
	fileServer := http.FileServer(http.Dir(cfg.UploadDir))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", fileServer))

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

// autoMigrateUsers migrates users from Redis to PostgreSQL if the PostgreSQL users table is empty.
func autoMigrateUsers(redisStore *store.Store, pgStore *store.PostgresStore) {
	ctx := context.Background()
	empty, err := pgStore.IsUsersTableEmpty(ctx)
	if err != nil {
		log.Printf("WARNING: Failed to check users table: %v", err)
		return
	}
	if !empty {
		return
	}

	keys, err := redisStore.Keys(ctx, "user:*")
	if err != nil {
		log.Printf("WARNING: Failed to list Redis users: %v", err)
		return
	}
	if len(keys) == 0 {
		return
	}

	log.Printf("Auto-migrating %d users from Redis to PostgreSQL...", len(keys))
	migrated := 0
	for _, key := range keys {
		data, err := redisStore.GetRaw(ctx, key)
		if err != nil {
			log.Printf("Auto-migrate: failed to get key %s: %v", key, err)
			continue
		}

		var u model.User
		if err := json.Unmarshal(data, &u); err != nil {
			log.Printf("Auto-migrate: failed to unmarshal %s: %v", key, err)
			continue
		}

		if u.Role == "" {
			u.Role = "admin"
		}

		if err := pgStore.CreateUser(ctx, u); err != nil {
			log.Printf("Auto-migrate: failed to insert %s: %v", u.Username, err)
			continue
		}
		migrated++
	}
	log.Printf("Auto-migration complete: %d/%d users migrated", migrated, len(keys))
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
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w, r)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src https://fonts.googleapis.com https://fonts.gstatic.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

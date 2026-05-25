package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"mioru/internal/auth"
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

	// Build the CORS allowlist from CORS_ORIGINS (must be done after .env load).
	allowedOrigins = buildAllowedOrigins()

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

	// Bootstrap the first admin from env when the users table is empty
	// (registration is invite-only, so this solves the first-admin chicken-and-egg).
	bootstrapAdmin(pgStore)

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
	customerH := handler.NewCustomerHandler(pgStore, cfg.SecretKey, cfg.TokenExpiry)

	// getRole resolves an authenticated user's role from the DB for RequireAdmin.
	getRole := func(ctx context.Context, username string) (string, error) {
		u, err := pgStore.GetUser(ctx, username)
		if err != nil {
			return "", err
		}
		if u == nil {
			return "", nil
		}
		return u.Role, nil
	}

	// rl builds a per-IP, 1-minute fixed-window rate limiter (Redis-backed) for an
	// endpoint, used to throttle credential-guessing on auth routes.
	rateIncr := func(ctx context.Context, key string) (int, error) {
		return redisStore.RateLimit(ctx, key, time.Minute)
	}
	rl := func(name string, limit int) func(http.HandlerFunc) http.HandlerFunc {
		return middleware.RateLimit(name, limit, rateIncr)
	}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth (no middleware) — registration is invite-only: only an existing admin
	// can create new admin users.
	mux.Handle("POST /api/auth/register", middleware.AuthMW(cfg.SecretKey)(middleware.RequireAdmin(getRole)(http.HandlerFunc(authH.Register))))
	mux.HandleFunc("POST /api/auth/login", cors(rl("admin-login", 10)(authH.Login)))
	mux.HandleFunc("POST /api/auth/forgot-password", cors(rl("forgot", 5)(authH.ForgotPassword)))
	mux.HandleFunc("POST /api/auth/reset-password", cors(rl("reset", 10)(authH.ResetPassword)))

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

	// Store: Customer auth & profile (separate namespace, separate JWT)
	mux.HandleFunc("POST /api/store/auth/register", cors(rl("cust-register", 5)(customerH.Register)))
	mux.HandleFunc("POST /api/store/auth/login", cors(rl("cust-login", 10)(customerH.Login)))
	mux.Handle("GET /api/store/customers/me", middleware.CustomerAuthMW(cfg.SecretKey)(http.HandlerFunc(customerH.Me)))
	mux.Handle("PUT /api/store/customers/me", middleware.CustomerAuthMW(cfg.SecretKey)(http.HandlerFunc(customerH.UpdateProfile)))
	mux.Handle("PUT /api/store/customers/me/password", middleware.CustomerAuthMW(cfg.SecretKey)(http.HandlerFunc(customerH.ChangePassword)))

	// Store: Public product & category endpoints (no auth)
	storeH := handler.NewStoreHandler(pgStore)
	mux.HandleFunc("GET /api/products", cors(storeH.ListProducts))
	mux.HandleFunc("GET /api/products/{slug}", cors(storeH.GetProduct))
	mux.HandleFunc("GET /api/categories", cors(storeH.ListCategories))

	// adminOnly composes AuthMW + RequireAdmin (DB-backed role check) for /api/admin/*.
	adminOnly := func(h http.Handler) http.Handler {
		return middleware.AuthMW(cfg.SecretKey)(middleware.RequireAdmin(getRole)(h))
	}

	// Admin: Categories (admin only)
	mux.Handle("GET /api/admin/categories", adminOnly(http.HandlerFunc(productH.Categories)))

	// Admin: Products (admin only)
	mux.Handle("GET /api/admin/products", adminOnly(http.HandlerFunc(productH.List)))
	mux.Handle("POST /api/admin/products", adminOnly(http.HandlerFunc(productH.Create)))
	mux.Handle("GET /api/admin/products/{slug}", adminOnly(http.HandlerFunc(productH.Get)))
	mux.Handle("PUT /api/admin/products/{slug}", adminOnly(http.HandlerFunc(productH.Update)))
	mux.Handle("DELETE /api/admin/products/{slug}", adminOnly(http.HandlerFunc(productH.Delete)))

	// Admin: Upload (admin only)
	mux.Handle("POST /api/admin/upload", adminOnly(http.HandlerFunc(productH.Upload)))

	// Admin: Migration (admin only)
	mux.Handle("POST /api/admin/migrate-users", adminOnly(handler.MigrateUsersHandler(redisStore, pgStore)))

	// Serve uploaded files (hardened: no script execution, no MIME sniffing)
	fileServer := http.FileServer(http.Dir(cfg.UploadDir))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", uploadsSecurity(fileServer)))

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

// bootstrapAdmin creates the first admin from BOOTSTRAP_ADMIN_* env vars when the
// users table is empty. Registration is invite-only (admins create admins), so
// this resolves the chicken-and-egg for the very first admin.
func bootstrapAdmin(pgStore *store.PostgresStore) {
	username := os.Getenv("BOOTSTRAP_ADMIN_USERNAME")
	emailAddr := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if username == "" || emailAddr == "" || password == "" {
		return
	}

	ctx := context.Background()
	empty, err := pgStore.IsUsersTableEmpty(ctx)
	if err != nil {
		log.Printf("WARNING: bootstrap admin — failed to check users table: %v", err)
		return
	}
	if !empty {
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Printf("WARNING: bootstrap admin — failed to hash password: %v", err)
		return
	}

	u := model.User{
		Username:    username,
		Email:       emailAddr,
		HashedPW:    hash,
		DisplayName: username,
		AvatarColor: "#44944A",
		Role:        "admin",
	}
	if err := pgStore.CreateUser(ctx, u); err != nil {
		log.Printf("WARNING: bootstrap admin — failed to create admin %q: %v", username, err)
		return
	}
	log.Printf("Bootstrapped first admin user %q from BOOTSTRAP_ADMIN_* env", username)
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

// allowedOrigins is the CORS allowlist, populated in main() from CORS_ORIGINS.
var allowedOrigins map[string]bool

// buildAllowedOrigins parses the comma-separated CORS_ORIGINS env var into an
// allowlist. When it is unset, a built-in default list (dev ports + production
// store and admin domains) is used. Setting CORS_ORIGINS replaces the defaults
// entirely, so production must list every origin it intends to allow.
func buildAllowedOrigins() map[string]bool {
	m := map[string]bool{}
	if env := os.Getenv("CORS_ORIGINS"); strings.TrimSpace(env) != "" {
		for _, o := range strings.Split(env, ",") {
			if o = strings.TrimSpace(o); o != "" {
				m[o] = true
			}
		}
		return m
	}
	defaults := []string{
		"http://localhost:5173", "http://127.0.0.1:5173",
		"http://localhost:5174", "http://127.0.0.1:5174",
		"http://localhost:8080", "http://127.0.0.1:8080",
		"https://mioru.store", "https://www.mioru.store",
		"https://admin.mioru.store", "https://www.admin.mioru.store",
	}
	for _, o := range defaults {
		m[o] = true
	}
	return m
}

func corsHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	// Reflect the origin only if it is explicitly allowlisted. Combined with
	// Allow-Credentials, reflecting an arbitrary origin would let any site make
	// authenticated cross-origin requests, so the allowlist is mandatory.
	if origin != "" && allowedOrigins[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")
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
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// uploadsSecurity hardens responses for user-uploaded files: it blocks MIME
// sniffing and forbids any active content via a locked-down CSP, neutralizing a
// malicious payload that might slip past upload validation.
func uploadsSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		next.ServeHTTP(w, r)
	})
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"mioru/internal/config"
	"mioru/internal/cookieauth"
	"mioru/internal/email"
	"mioru/internal/handler"
	"mioru/internal/middleware"
	"mioru/internal/store"
	"mioru/internal/telegram"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Build the CORS allowlist from CORS_ORIGINS (must be done after .env load).
	allowedOrigins = buildAllowedOrigins()

	// config.Load resolves SECRET_KEY (fatal when missing/short in production,
	// random with a warning in development) and the rest of the runtime config.
	cfg := config.Load()

	// PostgreSQL store (users, products, categories, reset tokens). The first
	// admin is seeded from BOOTSTRAP_ADMIN_* inside migrate() on a clean DB.
	pgStore, err := store.NewPostgresStore(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("PostgreSQL initialization failed:", err)
	}
	defer pgStore.Close()

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Printf("WARNING: Failed to create upload dir %s: %v", cfg.UploadDir, err)
	}

	emailSvc := email.NewService()

	// Handlers. The `secure` flag is on iff we're in production (HTTPS only) —
	// in development the cookie must be sent over plain HTTP, so Secure stays
	// off or the browser silently drops it and the SPA fails to authenticate.
	secureCookies := cfg.IsProduction()
	authH := handler.NewAuthHandler(pgStore, emailSvc, cfg.SecretKey, cfg.TokenExpiry, secureCookies, cfg.CookieDomain)
	productH := handler.NewProductHandler(pgStore, cfg.UploadDir)
	adminOrderH := handler.NewAdminOrderHandler(pgStore)
	adminCustomerH := handler.NewAdminCustomerHandler(pgStore)
	customerH := handler.NewCustomerHandler(pgStore, cfg.SecretKey, cfg.TokenExpiry, secureCookies, cfg.TelegramBotToken, cfg.CookieDomain,
		cfg.UploadDir,
		telegram.NewNotifier(cfg.TelegramBotToken, cfg.TelegramManagerChatIDs, cfg.APIBaseURL, cfg.UploadDir))

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

	// authMW / customerAuthMW authenticate a request and enforce session
	// revocation: a token issued before the account's password_changed_at is
	// rejected, so changing or resetting a password logs out prior sessions. The
	// epoch lookup is a single indexed read per authenticated request.
	authMW := middleware.AuthMW(cfg.SecretKey, pgStore.UserPasswordChangedAt)
	customerAuthMW := middleware.CustomerAuthMW(cfg.SecretKey, pgStore.CustomerPasswordChangedAt)

	// rl builds a per-IP, 1-minute fixed-window rate limiter (in-process) for an
	// endpoint, used to throttle credential-guessing on auth routes.
	rateLimiter := middleware.NewMemoryRateLimiter(time.Minute)
	// Trust X-Forwarded-For/X-Real-IP for client identification only when running
	// behind a trusted reverse proxy (nginx sets TRUST_PROXY=true). Otherwise the
	// headers are spoofable and would let an attacker rotate fake IPs to evade the
	// per-IP rate limit.
	trustProxy, _ := strconv.ParseBool(os.Getenv("TRUST_PROXY"))
	rl := func(name string, limit int) func(http.HandlerFunc) http.HandlerFunc {
		return middleware.RateLimit(name, limit, trustProxy, rateLimiter.Incr)
	}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// adminCSRF / customerCSRF enforce the double-submit-cookie check on every
	// state-changing request that rides an authenticated session: the SPA must
	// echo the csrf cookie value back in the X-CSRF-Token header. Safe methods
	// (GET/HEAD/OPTIONS) are passed through, so read endpoints aren't wrapped.
	// Login / register / forgot / reset are unauthenticated and bootstrap the
	// session itself — wrapping them would be a chicken-and-egg, so they stay
	// out of the CSRF gate (rate limiting is the primary abuse guard there).
	adminCSRF := middleware.CSRF(cookieauth.AdminCSRFCookie)
	customerCSRF := middleware.CSRF(cookieauth.StoreCSRFCookie)

	// Auth (no middleware) — registration is invite-only: only an existing admin
	// can create new admin users.
	mux.Handle("POST /api/auth/register", authMW(adminCSRF(middleware.RequireSuperAdmin(getRole)(http.HandlerFunc(authH.Register)))))
	mux.HandleFunc("POST /api/auth/login", cors(rl("admin-login", 10)(authH.Login)))
	mux.HandleFunc("POST /api/auth/forgot-password", cors(rl("forgot", 5)(authH.ForgotPassword)))
	mux.HandleFunc("POST /api/auth/reset-password", cors(rl("reset", 10)(authH.ResetPassword)))
	mux.Handle("POST /api/auth/logout", authMW(adminCSRF(http.HandlerFunc(authH.Logout))))

	// User profile (with auth)
	mux.Handle("GET /api/users/me", authMW(http.HandlerFunc(authH.Me)))
	mux.Handle("PUT /api/users/me/profile", authMW(adminCSRF(http.HandlerFunc(authH.UpdateProfile))))
	mux.Handle("PUT /api/users/me/password", authMW(adminCSRF(http.HandlerFunc(authH.ChangePassword))))

	registerTestRoutes(mux, pgStore, cfg, os.Getenv)

	// Store: Customer auth & profile (separate namespace, separate JWT)
	mux.HandleFunc("POST /api/store/auth/register", cors(rl("cust-register", 5)(customerH.Register)))
	mux.HandleFunc("POST /api/store/auth/login", cors(rl("cust-login", 10)(customerH.Login)))
	mux.HandleFunc("POST /api/store/auth/telegram", cors(rl("telegram", 10)(customerH.TelegramLogin)))
	mux.Handle("POST /api/store/auth/logout", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.Logout))))
	mux.Handle("GET /api/store/customers/me", customerAuthMW(http.HandlerFunc(customerH.Me)))
	mux.Handle("PUT /api/store/customers/me", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.UpdateProfile))))
	mux.Handle("PUT /api/store/customers/me/password", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.ChangePassword))))
	mux.Handle("POST /api/store/customers/me/set-password", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.SetPassword))))
	mux.Handle("POST /api/store/customers/me/oauth", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.LinkOAuth))))
	mux.Handle("GET /api/store/customers/me/orders", customerAuthMW(http.HandlerFunc(customerH.ListOrders)))
	mux.Handle("POST /api/store/orders", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.CreateOrder))))
	mux.Handle("POST /api/store/orders/upload-photo", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.UploadOrderPhoto))))

	// Customer cart & favorites (auth required)
	mux.Handle("GET /api/store/customers/me/cart", customerAuthMW(http.HandlerFunc(customerH.GetCart)))
	mux.Handle("PUT /api/store/customers/me/cart", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.SaveCart))))
	mux.Handle("GET /api/store/customers/me/favorites", customerAuthMW(http.HandlerFunc(customerH.GetFavorites)))
	mux.Handle("PUT /api/store/customers/me/favorites", customerAuthMW(customerCSRF(http.HandlerFunc(customerH.SaveFavorites))))

	// Store: Public product & category endpoints (no auth)
	storeH := handler.NewStoreHandler(pgStore)
	mux.HandleFunc("GET /api/products", cors(storeH.ListProducts))
	mux.HandleFunc("GET /api/products/facets", cors(storeH.ListFacets))
	mux.HandleFunc("GET /api/products/{slug}", cors(storeH.GetProduct))
	mux.HandleFunc("GET /api/categories", cors(storeH.ListCategories))

	// adminOnly composes AuthMW + RequireAdmin (DB-backed role check) + CSRF
	// for /api/admin/*. The CSRF middleware passes GET/HEAD/OPTIONS through
	// unchecked, so read endpoints aren't affected; mutations require a
	// matching X-CSRF-Token header.
	adminOnly := func(h http.Handler) http.Handler {
		return authMW(middleware.RequireAdmin(getRole)(adminCSRF(h)))
	}

	// superAdminOnly gates routes that only the super_admin can access
	// (user management: list, create, delete).
	superAdminOnly := func(h http.Handler) http.Handler {
		return authMW(middleware.RequireSuperAdmin(getRole)(adminCSRF(h)))
	}

	// Admin: Users (super_admin only)
	mux.Handle("GET /api/admin/users", superAdminOnly(http.HandlerFunc(authH.ListUsers)))
	mux.Handle("DELETE /api/admin/users/{username}", superAdminOnly(http.HandlerFunc(authH.DeleteUser)))

	// Admin: Categories (admin only)
	mux.Handle("GET /api/admin/categories", adminOnly(http.HandlerFunc(productH.Categories)))

	// Admin: Products (admin only)
	mux.Handle("GET /api/admin/products", adminOnly(http.HandlerFunc(productH.List)))
	mux.Handle("POST /api/admin/products", adminOnly(http.HandlerFunc(productH.Create)))
	mux.Handle("GET /api/admin/products/{slug}", adminOnly(http.HandlerFunc(productH.Get)))
	mux.Handle("PUT /api/admin/products/{slug}", adminOnly(http.HandlerFunc(productH.Update)))
	mux.Handle("DELETE /api/admin/products/{slug}", adminOnly(http.HandlerFunc(productH.Delete)))

	// Admin: Orders (admin only)
	mux.Handle("GET /api/admin/orders", adminOnly(http.HandlerFunc(adminOrderH.ListAll)))
	mux.Handle("PATCH /api/admin/orders/{id}/status", adminOnly(http.HandlerFunc(adminOrderH.UpdateStatus)))
	mux.Handle("GET /api/admin/customers", adminOnly(http.HandlerFunc(adminCustomerH.List)))
	mux.Handle("GET /api/admin/customers/{id}", adminOnly(http.HandlerFunc(adminCustomerH.Detail)))

	// Admin: Upload (admin only)
	mux.Handle("POST /api/admin/upload", adminOnly(http.HandlerFunc(productH.Upload)))

	// Serve uploaded files (hardened: no script execution, no MIME sniffing)
	fileServer := http.FileServer(http.Dir(cfg.UploadDir))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", uploadsSecurity(fileServer)))

	// CORS preflight
	mux.HandleFunc("OPTIONS /api/", func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w, r)
		w.WriteHeader(http.StatusNoContent)
	})

	addr := ":" + cfg.Port
	srv := newServer(addr, securityHeaders(mux))

	// Serve in a goroutine so main can block on a shutdown signal and then drain
	// in-flight requests instead of letting systemd kill them mid-flight.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for a fatal serve error or a termination signal: systemd stop and
	// redeploy send SIGTERM, Ctrl-C sends SIGINT.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case sig := <-stop:
		log.Printf("Received %s, draining in-flight requests...", sig)
	}

	// Give in-flight requests a bounded window to finish; the systemd unit's
	// default TimeoutStopSec (90s) comfortably exceeds this drain budget.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown timed out: %v", err)
	}
	log.Println("Server stopped")
}

// newServer builds the API HTTP server with explicit timeouts. net/http's
// defaults are all zero (unbounded), which leaves the server open to
// Slowloris-style connection-exhaustion DoS. ReadHeaderTimeout is the primary
// slow-header guard; the wider Read/Write budgets accommodate the 32 MiB
// multipart upload path on slower connections, and IdleTimeout caps idle
// keep-alive connections.
func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
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
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	// Auth lives in HttpOnly cookies (cookie-only — Authorization header is no
	// longer accepted); mutations must echo the CSRF cookie back in X-CSRF-Token.
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, Idempotency-Key")
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w, r)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		// The API serves JSON only, so 'unsafe-inline' adds attack surface
		// (HTML/SVG error pages, future regressions) without upside. Keep
		// font-src/style-src directives for the rare HTML rendering path but
		// without inline styles. The locked-down /uploads/ CSP is set
		// separately by uploadsSecurity.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src https://fonts.googleapis.com https://fonts.gstatic.com; style-src 'self' https://fonts.googleapis.com")
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

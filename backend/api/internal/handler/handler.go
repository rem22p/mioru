package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/email"
	"mioru/internal/middleware"
	"mioru/internal/model"
)

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// userStore is the subset of the store consumed by the admin auth handlers.
// Defined here (where it is used) to keep the seam small and let tests supply a
// fake; *store.PostgresStore satisfies it.
type userStore interface {
	CreateUser(ctx context.Context, u model.User) error
	GetUser(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateUser(ctx context.Context, username string, updates map[string]string) error
	UpdatePassword(ctx context.Context, username, hashedPW string) error
	CreateResetToken(ctx context.Context, username, token string) error
	ConsumeResetToken(ctx context.Context, token string) (string, error)
}

type AuthHandler struct {
	store  userStore
	email  *email.Service
	secret string
	expiry int
	// secure controls the Secure attribute on the auth/CSRF cookies. true in
	// production, false in dev (so the cookie is not silently dropped over
	// plain HTTP outside the localhost-secure-context browser exception).
	secure bool
}

func NewAuthHandler(pgStore userStore, emailSvc *email.Service, secret string, expiry int, secure bool) *AuthHandler {
	return &AuthHandler{store: pgStore, email: emailSvc, secret: secret, expiry: expiry, secure: secure}
}

type registerReq struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginUserResp is the body returned by Login: a small public profile of the
// authenticated user. The actual session lives in HttpOnly cookies set on the
// same response — the JSON deliberately carries no token, so JavaScript can
// never read it (closing the XSS-exfil path that motivated the migration to
// cookie-only auth).
type loginUserResp struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// createdUserResp is the response of Register: the new account's public summary,
// deliberately WITHOUT any token. Registration is an admin action (invite-only),
// not a login, so it must not hand the caller a session for the new account.
type createdUserResp struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !decodeJSON(w, r, &req) {
		return
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Username = strings.TrimSpace(req.Username)

	if req.FirstName == "" {
		jsonError(w, "имя обязательно", http.StatusBadRequest)
		return
	}
	if req.LastName == "" {
		jsonError(w, "фамилия обязательна", http.StatusBadRequest)
		return
	}
	if !emailRe.MatchString(req.Email) {
		jsonError(w, "некорректный email", http.StatusBadRequest)
		return
	}
	if len(req.Username) < 2 {
		jsonError(w, "никнейм минимум 2 символа", http.StatusBadRequest)
		return
	}
	if !usernameRe.MatchString(req.Username) {
		jsonError(w, "никнейм: только буквы, цифры и _", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "пароль минимум 8 символов", http.StatusBadRequest)
		return
	}
	if len(req.Password) > 72 {
		jsonError(w, "пароль максимум 72 символа", http.StatusBadRequest)
		return
	}
	// Password strength check
	if !isPasswordStrong(req.Password) {
		jsonError(w, "пароль должен содержать буквы и цифры", http.StatusBadRequest)
		return
	}
	// Common password check
	if isCommonPassword(req.Password) {
		jsonError(w, "пароль слишком распространённый", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	u := model.User{
		Username:    req.Username,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		HashedPW:    hash,
		DisplayName: req.FirstName + " " + req.LastName,
		AvatarColor: randomColor(),
		Role:        "admin",
	}

	if err := h.store.CreateUser(r.Context(), u); err != nil {
		if err.Error() == "username already exists" {
			jsonError(w, "никнейм занят", http.StatusBadRequest)
		} else if err.Error() == "email already registered" {
			jsonError(w, "email уже зарегистрирован", http.StatusBadRequest)
		} else {
			jsonError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// Return the created account's summary, never a session token: an admin
	// created this user, so logging the admin in as the new account would be
	// both wrong and a privilege/identity confusion.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUserResp{
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Username) > 100 || len(req.Password) > 72 {
		// Out-of-bounds credentials can't match any account; reject with the
		// same generic message (bcrypt ignores bytes past 72 anyway).
		jsonError(w, "неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	user, err := h.store.GetUser(r.Context(), req.Username)
	if err != nil || user == nil {
		// Constant-time: fake bcrypt call to prevent timing attack
		auth.CheckDummyPassword(req.Password)
		jsonError(w, "неверный логин или пароль", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(req.Password, user.HashedPW) {
		jsonError(w, "неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	tok, err := auth.CreateToken(req.Username, auth.TokenTypeUser, h.secret, h.expiry)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	csrf, err := cookieauth.GenCSRFToken()
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	// expiry is configured in minutes (see config.Config.TokenExpiry); the
	// browser wants seconds. Both cookies share the same MaxAge so the CSRF
	// token can't outlive the session it protects, and vice versa.
	maxAge := h.expiry * 60
	cookieauth.SetAuthCookie(w, cookieauth.AdminAuthCookie, tok, h.secure, maxAge)
	cookieauth.SetCSRFCookie(w, cookieauth.AdminCSRFCookie, csrf, h.secure, maxAge)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginUserResp{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

// Logout clears the admin auth and CSRF cookies. It is a state-changing
// endpoint (it mutates the client's session), so it MUST be mounted behind
// the CSRF middleware — otherwise a foreign origin could force-log-out an
// authenticated admin. The handler itself is intentionally idempotent: the
// cookies are cleared whether or not the caller's session was still valid.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookieauth.ClearCookie(w, cookieauth.AdminAuthCookie, h.secure)
	cookieauth.ClearCookie(w, cookieauth.AdminCSRFCookie, h.secure)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	user, err := h.store.GetUser(r.Context(), username)
	if err != nil || user == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           user.ID,
		"username":     user.Username,
		"first_name":   user.FirstName,
		"last_name":    user.LastName,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"avatar_color": user.AvatarColor,
		"role":         user.Role,
	})
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	var body map[string]string
	if !decodeJSON(w, r, &body) {
		return
	}
	// Validate only allowed fields with length limits
	updates := map[string]string{}
	for k, v := range body {
		switch k {
		case "display_name":
			if len(v) < 1 || len(v) > 100 {
				jsonError(w, "display_name: 1-100 символов", http.StatusBadRequest)
				return
			}
			updates[k] = v
		case "avatar_color":
			if len(v) > 20 {
				jsonError(w, "avatar_color: слишком длинный", http.StatusBadRequest)
				return
			}
			updates[k] = v
		case "first_name", "last_name":
			if len(v) > 100 {
				jsonError(w, k+": максимум 100 символов", http.StatusBadRequest)
				return
			}
			updates[k] = v
		default:
			jsonError(w, "unknown field: "+k, http.StatusBadRequest)
			return
		}
	}
	if len(updates) == 0 {
		jsonError(w, "no valid fields", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateUser(r.Context(), username, updates); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	var body struct {
		CurrentPW string `json:"current_password"`
		NewPW     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user, err := h.store.GetUser(r.Context(), username)
	if err != nil || user == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !auth.CheckPassword(body.CurrentPW, user.HashedPW) {
		jsonError(w, "неверный текущий пароль", http.StatusUnauthorized)
		return
	}
	if len(body.NewPW) < 8 {
		jsonError(w, "пароль минимум 8 символов", http.StatusBadRequest)
		return
	}
	if len(body.NewPW) > 72 {
		jsonError(w, "пароль максимум 72 символа", http.StatusBadRequest)
		return
	}
	if body.NewPW == body.CurrentPW {
		jsonError(w, "новый пароль должен отличаться", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(body.NewPW)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdatePassword(r.Context(), username, hash); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// ForgotPassword — request password reset via email
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	body.Email = strings.TrimSpace(strings.ToLower(body.Email))
	if !emailRe.MatchString(body.Email) {
		jsonError(w, "некорректный email", http.StatusBadRequest)
		return
	}

	// Generate cryptographically secure token
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Resolve the account from PostgreSQL.
	user, err := h.store.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		log.Printf("ForgotPassword GetUserByEmail: %v", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user != nil {
		// Persist the token (1 hour TTL) only for a real account.
		if err := h.store.CreateResetToken(r.Context(), user.Username, token); err != nil {
			log.Printf("ForgotPassword CreateResetToken: %v", err)
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Send email asynchronously (don't block the response)
		go func(emailAddr, tok string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[EMAIL] Panic sending reset email: %v", r)
				}
			}()
			h.email.SendPasswordReset(emailAddr, tok)
		}(body.Email, token)
	}

	// Always return success (don't reveal if email exists)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "если email зарегистрирован, мы отправили письмо"})
}

// ResetPassword — reset password using token
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	if len(body.Password) < 8 {
		jsonError(w, "пароль минимум 8 символов", http.StatusBadRequest)
		return
	}
	if len(body.Password) > 72 {
		jsonError(w, "пароль максимум 72 символа", http.StatusBadRequest)
		return
	}
	if !isPasswordStrong(body.Password) {
		jsonError(w, "пароль должен содержать буквы и цифры", http.StatusBadRequest)
		return
	}
	if isCommonPassword(body.Password) {
		jsonError(w, "пароль слишком распространённый", http.StatusBadRequest)
		return
	}

	// Consume token (one-time use)
	username, err := h.store.ConsumeResetToken(r.Context(), body.Token)
	if err != nil {
		jsonError(w, "недействительная или истёкшая ссылка", http.StatusBadRequest)
		return
	}

	// Update password
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdatePassword(r.Context(), username, hash); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// maxJSONBody bounds the size of a JSON request body. Auth and profile payloads
// are tiny, so 1 MiB is generous; it stops a client from exhausting server
// memory with an unbounded body. (The multipart upload path has its own,
// larger, MaxBytesReader.)
const maxJSONBody = 1 << 20 // 1 MiB

// decodeJSON caps the request body at maxJSONBody and decodes it into dst. On a
// malformed or oversized body it writes a 400 and returns false, so callers can
// simply `if !decodeJSON(w, r, &req) { return }`.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func randomColor() string {
	colors := []string{"#f85149", "#58a6ff", "#3fb950", "#f0883e", "#bc8cff", "#79c0ff", "#f778ba", "#7ee787"}
	b := make([]byte, 1)
	rand.Read(b)
	return colors[int(b[0])%len(colors)]
}

func isPasswordStrong(pw string) bool {
	hasLetter := false
	hasDigit := false
	for _, c := range pw {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func isCommonPassword(pw string) bool {
	common := map[string]bool{
		"password": true, "12345678": true, "qwerty12": true,
		"123456789": true, "1234567890": true, "abcdefgh": true,
	}
	return common[strings.ToLower(pw)]
}

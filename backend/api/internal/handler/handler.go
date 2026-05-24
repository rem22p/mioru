package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"mioru/internal/auth"
	"mioru/internal/email"
	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
)

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type AuthHandler struct {
	store      *store.SQLiteStore
	redisStore *store.Store
	email      *email.Service
	secret     string
	expiry     int
}

func NewAuthHandler(sqliteStore *store.SQLiteStore, redisStore *store.Store, emailSvc *email.Service, secret string, expiry int) *AuthHandler {
	return &AuthHandler{store: sqliteStore, redisStore: redisStore, email: emailSvc, secret: secret, expiry: expiry}
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

type tokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
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

	if err := h.store.CreateUser(u); err != nil {
		if err.Error() == "username already exists" {
			jsonError(w, "никнейм занят", http.StatusBadRequest)
		} else if err.Error() == "email already registered" {
			jsonError(w, "email уже зарегистрирован", http.StatusBadRequest)
		} else {
			jsonError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	tok, err := auth.CreateToken(req.Username, h.secret, h.expiry)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResp{AccessToken: tok, TokenType: "bearer"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.store.GetUser(req.Username)
	if err != nil || user == nil {
		// Constant-time: fake bcrypt call to prevent timing attack
		auth.CheckPassword(req.Password, "$2a$12$LJ3m4ys3Lk6L0qMqR0qMqO0qMqR0qMqR0qMqR0qMqR0qMqR0qMqR")
		jsonError(w, "неверный логин или пароль", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(req.Password, user.HashedPW) {
		jsonError(w, "неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	tok, err := auth.CreateToken(req.Username, h.secret, h.expiry)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResp{AccessToken: tok, TokenType: "bearer"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	user, err := h.store.GetUser(username)
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
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
	if err := h.store.UpdateUser(username, updates); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return
	}
	user, err := h.store.GetUser(username)
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
	if err := h.store.UpdatePassword(username, hash); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
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

	// Store token in Redis (1 hour TTL)
	if err := h.redisStore.CreateResetToken(r.Context(), body.Email, token); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
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
	username, err := h.redisStore.ConsumeResetToken(r.Context(), body.Token)
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
	if err := h.store.UpdatePassword(username, hash); err != nil {
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

func setupCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	allowed := map[string]bool{
		"http://localhost:5173":             true,
		"http://127.0.0.1:5173":            true,
		"http://localhost:5174":             true,
		"http://127.0.0.1:5174":            true,
		"http://localhost:8080":             true,
		"http://127.0.0.1:8080":            true,
		"https://admin.mioru.store":         true,
		"https://www.admin.mioru.store":     true,
	}
	if allowed[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// ── Notes ──

type NoteHandler struct {
	store *store.Store
}

func NewNoteHandler(s *store.Store) *NoteHandler {
	return &NoteHandler{store: s}
}

func (h *NoteHandler) List(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	notes, err := h.store.GetUserNotes(r.Context(), username)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if notes == nil {
		notes = []model.Note{}
	}
	setupCORS(w, r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

type createNoteReq struct {
	Content string `json:"content"`
	Color   string `json:"color"`
	PosX    int    `json:"position_x"`
	PosY    int    `json:"position_y"`
}

func (h *NoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createNoteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return
	}
	// Sanitize content to prevent XSS
	req.Content = html.EscapeString(strings.TrimSpace(req.Content))
	if req.Content == "" {
		jsonError(w, "content is required", http.StatusBadRequest)
		return
	}
	if len(req.Content) > 2000 {
		jsonError(w, "content max 2000 characters", http.StatusBadRequest)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	n := model.Note{
		ID:        generateID(),
		Content:   req.Content,
		Color:     req.Color,
		Author:    middleware.Username(r),
		PosX:      req.PosX,
		PosY:      req.PosY,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if n.Color == "" {
		n.Color = "#ffffff"
	}
	if err := h.store.CreateNote(r.Context(), n); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	setupCORS(w, r)
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(n)
}

func (h *NoteHandler) Update(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	id := r.PathValue("id")

	// Get note and verify ownership
	note, err := h.store.GetNote(r.Context(), id)
	if err != nil || note == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if note.Author != username {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "bad request", http.StatusBadRequest)
		return
	}

	// Sanitize content if being updated
	if content, ok := body["content"]; ok {
		if s, ok := content.(string); ok {
			body["content"] = html.EscapeString(strings.TrimSpace(s))
			if len(body["content"].(string)) > 2000 {
				jsonError(w, "content max 2000 characters", http.StatusBadRequest)
				return
			}
		}
	}

	body["updated_at"] = time.Now().UTC().Format(time.RFC3339)
	note, err = h.store.UpdateNote(r.Context(), id, body)
	if err != nil || note == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	setupCORS(w, r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (h *NoteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	username := middleware.Username(r)
	id := r.PathValue("id")

	// Verify ownership
	note, err := h.store.GetNote(r.Context(), id)
	if err != nil || note == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if note.Author != username {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.store.DeleteNote(r.Context(), id); err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	setupCORS(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/middleware"
	"mioru/internal/model"
)

var customerEmailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// customerStore is the subset of the store consumed by the storefront customer
// handlers. Defined here (where it is used) to keep the seam small and let tests
// supply a fake; *store.PostgresStore satisfies it.
type customerStore interface {
	CreateCustomer(ctx context.Context, c model.Customer) error
	GetCustomer(ctx context.Context, id int64) (*model.Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*model.Customer, error)
	UpdateCustomer(ctx context.Context, id int64, updates map[string]string) error
	UpdateCustomerPassword(ctx context.Context, id int64, hashedPW string) error

	// OAuth
	GetCustomerByOAuth(ctx context.Context, provider, oauthID string) (*model.Customer, *model.CustomerOAuth, error)
	CreateCustomerWithOAuth(ctx context.Context, c model.Customer, oa model.CustomerOAuth) error
	LinkOAuth(ctx context.Context, customerID int64, oa model.CustomerOAuth) error

	// Orders
	ListCustomerOrders(ctx context.Context, customerID int64, page, perPage int) ([]model.Order, int, error)
}

// CustomerHandler handles store customer auth & profile.
type CustomerHandler struct {
	store        customerStore
	secret       string
	expiry       int
	secure       bool
	botToken     string
	cookieDomain string
}

func NewCustomerHandler(s customerStore, secret string, expiry int, secure bool, botToken string, cookieDomain string) *CustomerHandler {
	return &CustomerHandler{store: s, secret: secret, expiry: expiry, secure: secure, botToken: botToken, cookieDomain: cookieDomain}
}

// ── Request / response types ──

type customerRegisterReq struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
}

type customerLoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type telegramLoginReq struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

type setPasswordReq struct {
	NewPassword string `json:"new_password"`
}

type linkOAuthReq struct {
	Provider    string `json:"provider"`
	OAuthID     string `json:"oauth_id"`     // for non-Telegram providers
	ProfileData string `json:"profile_data"`
	// Telegram-specific: full signed payload from the Login Widget.
	// When provider == "telegram", these fields are required and are
	// verified via auth.VerifyTelegramAuth before linking.
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	PhotoURL  string `json:"photo_url"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

// customerProfileResp is returned by Register/Login: a small public profile
// of the authenticated customer. The session itself lives in HttpOnly cookies
// set on the same response — no token is exposed in JSON (XSS-exfil guard).
type customerProfileResp struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Phone       string `json:"phone"`
	AvatarColor string `json:"avatar_color"`
}

// ── Handlers ──

func (h *CustomerHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req customerRegisterReq
	if !decodeJSON(w, r, &req) {
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.FirstName == "" {
		jsonError(w, "имя обязательно", http.StatusBadRequest)
		return
	}
	if !customerEmailRe.MatchString(req.Email) {
		jsonError(w, "некорректный email", http.StatusBadRequest)
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
	if !isPasswordStrong(req.Password) {
		jsonError(w, "пароль должен содержать буквы и цифры", http.StatusBadRequest)
		return
	}
	if isCommonPassword(req.Password) {
		jsonError(w, "пароль слишком распространённый", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	c := model.Customer{
		Email:       req.Email,
		HashedPW:    hash,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Phone:       req.Phone,
		AvatarColor: randomColor(),
	}

	if err := h.store.CreateCustomer(r.Context(), c); err != nil {
		if err.Error() == "email already registered" {
			jsonError(w, "email уже зарегистрирован", http.StatusConflict)
		} else {
			jsonError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the created customer to get the ID
	cust, err := h.store.GetCustomerByEmail(r.Context(), req.Email)
	if err != nil || cust == nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// JWT subject = customer ID
	if err := h.issueSession(w, cust.ID); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customerProfileResp{
		ID:          cust.ID,
		Email:       cust.Email,
		FirstName:   cust.FirstName,
		LastName:    cust.LastName,
		Phone:       cust.Phone,
		AvatarColor: cust.AvatarColor,
	})
}

// issueSession mints a customer JWT for id and writes both session cookies
// (auth + CSRF) onto w. It is the single place where the customer session is
// materialised so Register and Login can't drift apart on cookie attributes.
func (h *CustomerHandler) issueSession(w http.ResponseWriter, id int64) error {
	tok, err := auth.CreateToken(fmt.Sprintf("%d", id), auth.TokenTypeCustomer, h.secret, h.expiry)
	if err != nil {
		return err
	}
	csrf, err := cookieauth.GenCSRFToken()
	if err != nil {
		return err
	}
	maxAge := h.expiry * 60
	cookieauth.SetAuthCookie(w, cookieauth.StoreAuthCookie, tok, h.secure, maxAge, h.cookieDomain)
	cookieauth.SetCSRFCookie(w, cookieauth.StoreCSRFCookie, csrf, h.secure, maxAge, h.cookieDomain)
	return nil
}

func (h *CustomerHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req customerLoginReq
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Email) > 100 || len(req.Password) > 72 {
		jsonError(w, "неверный email или пароль", http.StatusUnauthorized)
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	cust, err := h.store.GetCustomerByEmail(r.Context(), req.Email)
	if err != nil || cust == nil {
		// Constant-time dummy check to prevent timing attacks
		auth.CheckDummyPassword(req.Password)
		jsonError(w, "неверный email или пароль", http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(req.Password, cust.HashedPW) {
		jsonError(w, "неверный email или пароль", http.StatusUnauthorized)
		return
	}

	if err := h.issueSession(w, cust.ID); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customerProfileResp{
		ID:          cust.ID,
		Email:       cust.Email,
		FirstName:   cust.FirstName,
		LastName:    cust.LastName,
		Phone:       cust.Phone,
		AvatarColor: cust.AvatarColor,
	})
}

// Logout clears the storefront session cookies. Mount behind CSRF — without
// it any third-party origin could force-log-out a signed-in customer just by
// triggering a POST. Idempotent: succeeds even if no session was active.
func (h *CustomerHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookieauth.ClearCookie(w, cookieauth.StoreAuthCookie, h.secure, h.cookieDomain)
	cookieauth.ClearCookie(w, cookieauth.StoreCSRFCookie, h.secure, h.cookieDomain)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (h *CustomerHandler) Me(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)
	cust, err := h.store.GetCustomer(r.Context(), id)
	if err != nil || cust == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           cust.ID,
		"email":        cust.Email,
		"first_name":   cust.FirstName,
		"last_name":    cust.LastName,
		"phone":        cust.Phone,
		"avatar_color": cust.AvatarColor,
	})
}

func (h *CustomerHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)

	var body map[string]string
	if !decodeJSON(w, r, &body) {
		return
	}

	updates := map[string]string{}
	for k, v := range body {
		switch k {
		case "first_name", "last_name":
			if len(v) > 100 {
				jsonError(w, k+": максимум 100 символов", http.StatusBadRequest)
				return
			}
			updates[k] = v
		case "phone":
			if len(v) > 30 {
				jsonError(w, "phone: максимум 30 символов", http.StatusBadRequest)
				return
			}
			updates[k] = v
		case "avatar_color":
			if len(v) > 20 {
				jsonError(w, "avatar_color: максимум 20 символов", http.StatusBadRequest)
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

	if err := h.store.UpdateCustomer(r.Context(), id, updates); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (h *CustomerHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)

	var body struct {
		CurrentPW string `json:"current_password"`
		NewPW     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	cust, err := h.store.GetCustomer(r.Context(), id)
	if err != nil || cust == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !auth.CheckPassword(body.CurrentPW, cust.HashedPW) {
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
	if err := h.store.UpdateCustomerPassword(r.Context(), id, hash); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// ── OAuth handlers ──

// TelegramLogin authenticates a customer via the Telegram Login Widget.
// The frontend receives signed data from the widget via data-onauth callback,
// posts it here, and the server verifies the HMAC-SHA256 signature before
// issuing a session. If the Telegram user has never logged in before, a new
// customer (OAuth-only, no email/password) is created atomically.
//
// Rate-limited like login; no CSRF needed — this bootstraps the session.
func (h *CustomerHandler) TelegramLogin(w http.ResponseWriter, r *http.Request) {
	if h.botToken == "" {
		jsonError(w, "Telegram login is not configured", http.StatusServiceUnavailable)
		return
	}

	var req telegramLoginReq
	if !decodeJSON(w, r, &req) {
		return
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Username = strings.TrimSpace(req.Username)
	req.PhotoURL = strings.TrimSpace(req.PhotoURL)

	if req.ID <= 0 {
		jsonError(w, "некорректный Telegram ID", http.StatusBadRequest)
		return
	}
	if req.FirstName == "" {
		jsonError(w, "имя обязательно", http.StatusBadRequest)
		return
	}
	if len(req.FirstName) > 100 {
		jsonError(w, "имя максимум 100 символов", http.StatusBadRequest)
		return
	}
	if len(req.LastName) > 100 {
		jsonError(w, "фамилия максимум 100 символов", http.StatusBadRequest)
		return
	}
	if len(req.Username) > 100 {
		jsonError(w, "username максимум 100 символов", http.StatusBadRequest)
		return
	}
	if len(req.PhotoURL) > 500 {
		jsonError(w, "photo_url максимум 500 символов", http.StatusBadRequest)
		return
	}
	if req.AuthDate <= 0 {
		jsonError(w, "auth_date обязателен", http.StatusBadRequest)
		return
	}
	if len(req.Hash) > 200 {
		jsonError(w, "hash слишком длинный", http.StatusBadRequest)
		return
	}

	data := auth.TelegramAuthData{
		ID:        req.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		PhotoURL:  req.PhotoURL,
		AuthDate:  req.AuthDate,
		Hash:      req.Hash,
	}

	if err := auth.VerifyTelegramAuth(data, h.botToken, 24*time.Hour); err != nil {
		jsonError(w, "неверная Telegram подпись", http.StatusUnauthorized)
		return
	}

	oauthID := fmt.Sprintf("%d", req.ID)

	// Try existing OAuth link first.
	cust, _, err := h.store.GetCustomerByOAuth(r.Context(), "telegram", oauthID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if cust != nil {
		if err := h.issueSession(w, cust.ID); err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(customerProfileResp{
			ID:          cust.ID,
			Email:       cust.Email,
			FirstName:   cust.FirstName,
			LastName:    cust.LastName,
			Phone:       cust.Phone,
			AvatarColor: cust.AvatarColor,
		})
		return
	}

	// New OAuth customer — create atomically with the oauth link.
	profileData := fmt.Sprintf(`{"first_name":"%s","last_name":"%s","username":"%s","photo_url":"%s"}`,
		escapeJSON(req.FirstName), escapeJSON(req.LastName), escapeJSON(req.Username), escapeJSON(req.PhotoURL))

	c := model.Customer{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Phone:       "",
		AvatarColor: randomColor(),
		// Email and HashedPW left empty — OAuth-only customer.
	}
	oa := model.CustomerOAuth{
		Provider:    "telegram",
		OAuthID:     oauthID,
		ProfileData: profileData,
	}

	if err := h.store.CreateCustomerWithOAuth(r.Context(), c, oa); err != nil {
		if strings.Contains(err.Error(), "already linked") {
			jsonError(w, "этот Telegram аккаунт уже привязан", http.StatusConflict)
		} else {
			jsonError(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the created customer to get the ID.
	cust, _, err = h.store.GetCustomerByOAuth(r.Context(), "telegram", oauthID)
	if err != nil || cust == nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.issueSession(w, cust.ID); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customerProfileResp{
		ID:          cust.ID,
		Email:       cust.Email,
		FirstName:   cust.FirstName,
		LastName:    cust.LastName,
		Phone:       cust.Phone,
		AvatarColor: cust.AvatarColor,
	})
}

// SetPassword sets a password for an OAuth customer that currently has none.
// Unlike ChangePassword, this does not require a current password — it is the
// first-time password setup. If the customer already has a password they must
// use ChangePassword instead.
func (h *CustomerHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)

	var body setPasswordReq
	if !decodeJSON(w, r, &body) {
		return
	}

	cust, err := h.store.GetCustomer(r.Context(), id)
	if err != nil || cust == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if cust.HashedPW != "" {
		jsonError(w, "пароль уже установлен — используйте /change-password", http.StatusConflict)
		return
	}

	if len(body.NewPassword) < 8 {
		jsonError(w, "пароль минимум 8 символов", http.StatusBadRequest)
		return
	}
	if len(body.NewPassword) > 72 {
		jsonError(w, "пароль максимум 72 символа", http.StatusBadRequest)
		return
	}
	if !isPasswordStrong(body.NewPassword) {
		jsonError(w, "пароль должен содержать буквы и цифры", http.StatusBadRequest)
		return
	}
	if isCommonPassword(body.NewPassword) {
		jsonError(w, "пароль слишком распространённый", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdateCustomerPassword(r.Context(), id, hash); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// LinkOAuth binds an OAuth provider to the currently authenticated customer.
// The operation is idempotent: linking the same provider+id twice is a no-op.
//
// For Telegram, the full signed payload (hash, auth_date, id, first_name, …)
// is required and verified via auth.VerifyTelegramAuth — a bare oauth_id is
// rejected to prevent account hijack (see issue #1).
func (h *CustomerHandler) LinkOAuth(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)

	var body linkOAuthReq
	if !decodeJSON(w, r, &body) {
		return
	}

	body.Provider = strings.TrimSpace(body.Provider)
	body.OAuthID = strings.TrimSpace(body.OAuthID)

	if body.Provider == "" {
		jsonError(w, "provider обязателен", http.StatusBadRequest)
		return
	}
	if len(body.Provider) > 50 {
		jsonError(w, "provider максимум 50 символов", http.StatusBadRequest)
		return
	}

	var oauthID string

	if body.Provider == "telegram" {
		// Telegram requires the full signed payload — never trust a bare oauth_id.
		if body.Hash == "" {
			jsonError(w, "hash обязателен для Telegram", http.StatusBadRequest)
			return
		}
		if body.AuthDate <= 0 {
			jsonError(w, "auth_date обязателен для Telegram", http.StatusBadRequest)
			return
		}
		if body.ID <= 0 {
			jsonError(w, "некорректный Telegram ID", http.StatusBadRequest)
			return
		}

		data := auth.TelegramAuthData{
			ID:        body.ID,
			FirstName: body.FirstName,
			LastName:  body.LastName,
			Username:  body.Username,
			PhotoURL:  body.PhotoURL,
			AuthDate:  body.AuthDate,
			Hash:      body.Hash,
		}

		if err := auth.VerifyTelegramAuth(data, h.botToken, 24*time.Hour); err != nil {
			jsonError(w, "неверная Telegram подпись", http.StatusUnauthorized)
			return
		}

		oauthID = fmt.Sprintf("%d", body.ID)
	} else {
		// Non-Telegram providers use the oauth_id directly.
		if body.OAuthID == "" {
			jsonError(w, "oauth_id обязателен", http.StatusBadRequest)
			return
		}
		if len(body.OAuthID) > 100 {
			jsonError(w, "oauth_id максимум 100 символов", http.StatusBadRequest)
			return
		}
		oauthID = body.OAuthID
	}

	if len(body.ProfileData) > 2000 {
		jsonError(w, "profile_data максимум 2000 символов", http.StatusBadRequest)
		return
	}

	oa := model.CustomerOAuth{
		CustomerID:  id,
		Provider:    body.Provider,
		OAuthID:     oauthID,
		ProfileData: body.ProfileData,
	}

	if err := h.store.LinkOAuth(r.Context(), id, oa); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// ListOrders returns the authenticated customer's order history, newest first.
// Query params: page (1-based, default 1), per_page (1-100, default 20).
func (h *CustomerHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)

	q := r.URL.Query()
	page := 1
	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		page = v
	}
	perPage := 20
	if v, err := strconv.Atoi(q.Get("per_page")); err == nil && v > 0 {
		perPage = v
	}

	orders, total, err := h.store.ListCustomerOrders(r.Context(), id, page, perPage)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	type orderResp struct {
		ID         int64  `json:"id"`
		TotalMinor int64  `json:"total_minor"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
	}
	out := make([]orderResp, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderResp{
			ID:         o.ID,
			TotalMinor: o.TotalMinor,
			Status:     o.Status,
			CreatedAt:  o.CreatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders":   out,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// escapeJSON escapes a string for safe inclusion in a JSON string value.
// It handles the characters that must be escaped per RFC 8259: backslash,
// double quote, and control characters.
func escapeJSON(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

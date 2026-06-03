package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mioru/internal/auth"
	"mioru/internal/cookieauth"
	"mioru/internal/middleware"
	"mioru/internal/model"
	"mioru/internal/store"
	"mioru/internal/telegram"
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
	CreateOrder(ctx context.Context, customerID int64, o *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error)

	// Cart & Favorites
	GetCustomerCart(ctx context.Context, customerID int64) ([]store.CartItem, error)
	SaveCustomerCart(ctx context.Context, customerID int64, items []store.CartItem) error
	GetCustomerFavorites(ctx context.Context, customerID int64) ([]int, error)
	SaveCustomerFavorites(ctx context.Context, customerID int64, productIDs []int) error
}

// CustomerHandler handles store customer auth & profile.
type CustomerHandler struct {
	store        customerStore
	secret       string
	expiry       int
	secure       bool
	botToken     string
	cookieDomain string
	uploadDir    string
	tgNotifier   *telegram.Notifier
}

func NewCustomerHandler(s customerStore, secret string, expiry int, secure bool, botToken string, cookieDomain string, uploadDir string, tgNotifier *telegram.Notifier) *CustomerHandler {
	return &CustomerHandler{store: s, secret: secret, expiry: expiry, secure: secure, botToken: botToken, cookieDomain: cookieDomain, uploadDir: uploadDir, tgNotifier: tgNotifier}
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
	if req.Phone == "" {
		jsonError(w, "телефон обязателен", http.StatusBadRequest)
		return
	}
	if len(req.Phone) < 5 || len(req.Phone) > 30 {
		jsonError(w, "телефон: от 5 до 30 символов", http.StatusBadRequest)
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
		jsonErrorCode(w, "неверный email или пароль", http.StatusUnauthorized, "AUTH_INVALID")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	cust, err := h.store.GetCustomerByEmail(r.Context(), req.Email)
	if err != nil || cust == nil {
		// Constant-time dummy check to prevent timing attacks
		auth.CheckDummyPassword(req.Password)
		jsonErrorCode(w, "неверный email или пароль", http.StatusUnauthorized, "AUTH_INVALID")
		return
	}
	if !auth.CheckPassword(req.Password, cust.HashedPW) {
		jsonErrorCode(w, "неверный email или пароль", http.StatusUnauthorized, "AUTH_INVALID")
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

	// Require current password for all profile changes.
	currentPW, hasPW := body["current_password"]
	delete(body, "current_password")
	if !hasPW || currentPW == "" {
		jsonErrorCode(w, "требуется текущий пароль", http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}

	cust, err := h.store.GetCustomer(r.Context(), id)
	if err != nil || cust == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !auth.CheckPassword(currentPW, cust.HashedPW) {
		jsonErrorCode(w, "неверный текущий пароль", http.StatusUnauthorized, "AUTH_INVALID")
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
		jsonErrorCode(w, "неверный текущий пароль", http.StatusUnauthorized, "AUTH_INVALID")
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

	if err := auth.VerifyTelegramAuth(data, h.botToken, 24*time.Hour, time.Now()); err != nil {
		jsonErrorCode(w, "неверная Telegram подпись", http.StatusUnauthorized, "AUTH_INVALID")
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
	profileBytes, _ := json.Marshal(map[string]string{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"username":   req.Username,
		"photo_url":  req.PhotoURL,
	})
	profileData := string(profileBytes)

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
		if errors.Is(err, store.ErrOAuthAlreadyLinked) {
			// Race: another request created the link between our GetCustomerByOAuth
			// and CreateCustomerWithOAuth. Retry — fetch the now-existing customer.
			cust, _, err2 := h.store.GetCustomerByOAuth(r.Context(), "telegram", oauthID)
			if err2 != nil || cust == nil {
				jsonError(w, "internal error", http.StatusInternalServerError)
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
			return
		}
		jsonError(w, "internal error", http.StatusInternalServerError)
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

		if err := auth.VerifyTelegramAuth(data, h.botToken, 24*time.Hour, time.Now()); err != nil {
			jsonErrorCode(w, "неверная Telegram подпись", http.StatusUnauthorized, "AUTH_INVALID")
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

// CreateOrder handles POST /api/store/orders — creates an order from checkout
// or individual order form. Requires customer auth + CSRF.
func (h *CustomerHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.CustomerID(r)

	var req struct {
		Type           string            `json:"type"`
		City           string            `json:"city"`
		DeliveryMethod string            `json:"delivery_method"`
		PaymentMethod  string            `json:"payment_method"`
		Street         string            `json:"street"`
		House          string            `json:"house"`
		Apartment      string            `json:"apartment"`
		Comment        string            `json:"comment"`
		TotalMinor     int64             `json:"total_minor"`
		Height         *float64          `json:"height"`
		Weight         *float64          `json:"weight"`
		DeliveryTime   []string          `json:"delivery_time"`
		Photos         []string          `json:"photos"`
		Items          []model.OrderItem `json:"items"`
	}

	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Type == "" {
		req.Type = "cart"
	}
	if req.City == "" {
		jsonErrorCode(w, "city is required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if req.DeliveryMethod == "" {
		jsonErrorCode(w, "delivery_method is required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if req.PaymentMethod == "" {
		jsonErrorCode(w, "payment_method is required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		jsonErrorCode(w, "Idempotency-Key header required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	order := &model.Order{
		Type:           req.Type,
		TotalMinor:     req.TotalMinor,
		City:           req.City,
		DeliveryMethod: req.DeliveryMethod,
		PaymentMethod:  req.PaymentMethod,
		Street:         req.Street,
		House:          req.House,
		Apartment:      req.Apartment,
		Comment:        req.Comment,
		Height:         req.Height,
		Weight:         req.Weight,
		DeliveryTime:   req.DeliveryTime,
		Photos:         req.Photos,
	}

	created, err := h.store.CreateOrder(r.Context(), customerID, order, req.Items, idempotencyKey, "")
	if err != nil {
		log.Printf("CreateOrder failed: %v", err)
		if strings.Contains(err.Error(), "idempotency") {
			jsonErrorCode(w, err.Error(), http.StatusConflict, "IDEMPOTENCY_REPLAY")
			return
		}
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)

	// Notify managers via Telegram (background, fire-and-forget)
	if h.tgNotifier != nil {
		go func() {
			defer func() { _ = recover() }()
			cust, err := h.store.GetCustomer(context.Background(), customerID)
			if err != nil {
				return
			}
			h.tgNotifier.OrderCreated(created, cust)
		}()
	}
}

// UploadOrderPhoto handles POST /api/store/orders/upload-photo — uploads a photo
// for an order (individual order form). Returns the URL path to the saved file.
// Requires customer auth + CSRF. Max 10 MB per file.
func (h *CustomerHandler) UploadOrderPhoto(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		jsonError(w, "file too large (max 10MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExt(ext) {
		jsonError(w, "only image files allowed (png)", http.StatusBadRequest)
		return
	}
	if err := validateImageContent(file); err != nil {
		jsonError(w, "file content is not a supported image", http.StatusBadRequest)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	safeName := uniquePrefix() + ext

	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		jsonError(w, "internal error: cannot create upload dir", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(h.uploadDir, safeName)
	dest, err := os.Create(destPath)
	if err != nil {
		jsonError(w, "internal error: cannot create file", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	if _, err := io.Copy(dest, file); err != nil {
		os.Remove(destPath)
		jsonError(w, "internal error: cannot write file", http.StatusInternalServerError)
		return
	}

	// Generate thumbnail (400×300) — best-effort.
	thumbName := "thumb_" + safeName
	thumbPath := filepath.Join(h.uploadDir, thumbName)
	if err := generateThumbnail(destPath, thumbPath, 400, 300); err != nil {
		log.Printf("UploadOrderPhoto: thumbnail failed for %s: %v", safeName, err)
	}

	url := "/uploads/" + safeName
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
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
	if perPage > 100 {
		perPage = 100
	}

	orders, total, err := h.store.ListCustomerOrders(r.Context(), id, page, perPage)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	type orderResp struct {
		ID             int64            `json:"id"`
		TotalMinor     int64            `json:"total_minor"`
		Status         string           `json:"status"`
		Type           string           `json:"type"`
		City           string           `json:"city"`
		DeliveryMethod string           `json:"delivery_method"`
		PaymentMethod  string           `json:"payment_method"`
		Street         string           `json:"street,omitempty"`
		House          string           `json:"house,omitempty"`
		Apartment      string           `json:"apartment,omitempty"`
		Comment        string           `json:"comment,omitempty"`
		Height         *float64         `json:"height,omitempty"`
		Weight         *float64         `json:"weight,omitempty"`
		DeliveryTime   []string         `json:"delivery_time,omitempty"`
		Items          []model.OrderItem `json:"items,omitempty"`
		CreatedAt      time.Time        `json:"created_at"`
	}
	out := make([]orderResp, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderResp{
			ID:             o.ID,
			TotalMinor:     o.TotalMinor,
			Status:         o.Status,
			Type:           o.Type,
			City:           o.City,
			DeliveryMethod: o.DeliveryMethod,
			PaymentMethod:  o.PaymentMethod,
			Street:         o.Street,
			House:          o.House,
			Apartment:      o.Apartment,
			Comment:        o.Comment,
			Height:         o.Height,
			Weight:         o.Weight,
			DeliveryTime:   o.DeliveryTime,
			Items:          o.Items,
			CreatedAt:      o.CreatedAt,
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

// ── Cart & Favorites handlers ──

const (
	maxCartItems      = 200
	maxFavoritesItems = 200
	maxSizeLabelLen   = 32
	maxQuantity       = 999
)

type cartItemReq struct {
	ProductID int    `json:"product_id"`
	SizeLabel string `json:"size_label"`
	Quantity  int    `json:"quantity"`
}

type cartSaveReq struct {
	Items []cartItemReq `json:"items"`
}

type favoritesSaveReq struct {
	ProductIDs []int `json:"product_ids"`
}

// GetCart returns the customer's saved cart.
func (h *CustomerHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)
	items, err := h.store.GetCustomerCart(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []store.CartItem{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}

// SaveCart replaces the customer's cart with the given items.
func (h *CustomerHandler) SaveCart(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)
	var req cartSaveReq
	if !decodeJSON(w, r, &req) {
		return
	}

	if len(req.Items) > maxCartItems {
		jsonError(w, "too many items", http.StatusBadRequest)
		return
	}

	items := make([]store.CartItem, 0, len(req.Items))
	for _, it := range req.Items {
		if it.ProductID <= 0 {
			jsonError(w, "invalid product_id", http.StatusBadRequest)
			return
		}
		if it.Quantity < 1 || it.Quantity > maxQuantity {
			jsonError(w, "quantity out of range", http.StatusBadRequest)
			return
		}
		if len(it.SizeLabel) > maxSizeLabelLen {
			jsonError(w, "size_label too long", http.StatusBadRequest)
			return
		}
		items = append(items, store.CartItem{
			ProductID: it.ProductID,
			SizeLabel: it.SizeLabel,
			Quantity:  it.Quantity,
		})
	}

	if err := h.store.SaveCustomerCart(r.Context(), id, items); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// GetFavorites returns the customer's saved favorite product IDs.
func (h *CustomerHandler) GetFavorites(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)
	ids, err := h.store.GetCustomerFavorites(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if ids == nil {
		ids = []int{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product_ids": ids})
}

// SaveFavorites replaces the customer's favorites with the given IDs.
func (h *CustomerHandler) SaveFavorites(w http.ResponseWriter, r *http.Request) {
	id := middleware.CustomerID(r)
	var req favoritesSaveReq
	if !decodeJSON(w, r, &req) {
		return
	}

	if len(req.ProductIDs) > maxFavoritesItems {
		jsonError(w, "too many product_ids", http.StatusBadRequest)
		return
	}
	for _, pid := range req.ProductIDs {
		if pid <= 0 {
			jsonError(w, "invalid product_id", http.StatusBadRequest)
			return
		}
	}

	if err := h.store.SaveCustomerFavorites(r.Context(), id, req.ProductIDs); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

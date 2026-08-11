package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
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
	UpdateCustomerPhoneIfChanged(ctx context.Context, id int64, phone string) (int64, error)
	UpdateCustomerPassword(ctx context.Context, id int64, hashedPW string) error

	// OAuth
	GetCustomerByOAuth(ctx context.Context, provider, oauthID string) (*model.Customer, *model.CustomerOAuth, error)
	CreateCustomerWithOAuth(ctx context.Context, c model.Customer, oa model.CustomerOAuth) error
	LinkOAuth(ctx context.Context, customerID int64, oa model.CustomerOAuth) error
	GetCustomerOAuth(ctx context.Context, customerID int64) ([]model.CustomerOAuth, error)

	// Orders
	ListCustomerOrders(ctx context.Context, customerID int64, page, perPage int) ([]model.Order, int, error)
	CreateOrder(ctx context.Context, customerID int64, o *model.Order, items []model.OrderItem, idempotencyKey, requestHash string) (*model.Order, error)

	// Cart & Favorites
	GetCustomerCart(ctx context.Context, customerID int64) ([]store.CartItem, error)
	SaveCustomerCart(ctx context.Context, customerID int64, items []store.CartItem) error
	GetCustomerFavorites(ctx context.Context, customerID int64) ([]int, error)
	SaveCustomerFavorites(ctx context.Context, customerID int64, productIDs []int) error

	// Idempotency: race-loser path. CreateOrder returns ErrIdempotencyRace
	// when a concurrent first submit won the INSERT on
	// order_idempotency_pkey. The handler then re-fetches the winner's
	// stored record and decides between true replay (201) and hash
	// mismatch (409 IDEMPOTENCY_REPLAY).
	GetOrderByIdempotencyKey(ctx context.Context, key string, customerID int64) (*store.IdempotencyRecord, error)
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
	OAuthID     string `json:"oauth_id"` // for non-Telegram providers
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

// customerTelegramResp is the Telegram binding info in /me responses.
type customerTelegramResp struct {
	Linked    bool   `json:"linked"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
}

// customerProfileResp is returned by Register/Login: a small public profile
// of the authenticated customer. The session itself lives in HttpOnly cookies
// set on the same response — no token is exposed in JSON (XSS-exfil guard).
type customerProfileResp struct {
	ID          int64                 `json:"id"`
	Email       string                `json:"email"`
	FirstName   string                `json:"first_name"`
	LastName    string                `json:"last_name"`
	Phone       string                `json:"phone"`
	AvatarColor string                `json:"avatar_color"`
	Telegram    *customerTelegramResp `json:"telegram"`
}

// telegramSummary extracts the telegram binding from the customer's OAuth
// links. Returns nil when not linked. Callers that already hold the OAuth
// list reuse it; otherwise they fetch it.
func telegramSummary(links []model.CustomerOAuth) *customerTelegramResp {
	for _, oa := range links {
		if oa.Provider != "telegram" {
			continue
		}
		sum := &customerTelegramResp{Linked: true}
		var profile struct {
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		}
		// profile_data is best-effort; a missing username is fine.
		_ = json.Unmarshal([]byte(oa.ProfileData), &profile)
		sum.Username = profile.Username
		sum.FirstName = profile.FirstName
		return sum
	}
	return &customerTelegramResp{Linked: false}
}

// customerProfile builds the profile response for a customer, attaching the
// Telegram binding status. Returns an error only on store failure.
func (h *CustomerHandler) customerProfile(ctx context.Context, cust *model.Customer) (customerProfileResp, error) {
	resp := customerProfileResp{
		ID:          cust.ID,
		Email:       cust.Email,
		FirstName:   cust.FirstName,
		LastName:    cust.LastName,
		Phone:       cust.Phone,
		AvatarColor: cust.AvatarColor,
	}
	links, err := h.store.GetCustomerOAuth(ctx, cust.ID)
	if err != nil {
		return resp, err
	}
	resp.Telegram = telegramSummary(links)
	return resp, nil
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
	if req.Phone != "" && !phoneRE.MatchString(req.Phone) {
		jsonError(w, "некорректный номер телефона", http.StatusBadRequest)
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
		Telegram:    &customerTelegramResp{Linked: false},
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

	// Bindings come from the store: the SPA seeds auth state from this body and
	// only refetches /me on a cold start, so a stale linked:false gates checkout.
	resp, err := h.customerProfile(r.Context(), cust)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

	resp, err := h.customerProfile(r.Context(), cust)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
			Telegram:    &customerTelegramResp{Linked: true, Username: req.Username, FirstName: req.FirstName},
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
				Telegram:    &customerTelegramResp{Linked: true, Username: req.Username, FirstName: req.FirstName},
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
		Telegram:    &customerTelegramResp{Linked: true, Username: req.Username, FirstName: req.FirstName},
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
	profileData := body.ProfileData

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
		// Checked after the shape: a malformed payload is caller error either
		// way, and unverifiable is not the same diagnosis as forged.
		if h.botToken == "" {
			jsonError(w, "Telegram login is not configured", http.StatusServiceUnavailable)
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
		// The signature covers these fields, the client blob does not: storing
		// the blob would let a customer label their own binding with someone
		// else's handle, which the admin console then shows as the contact.
		// TelegramLogin builds the record the same way.
		profileBytes, err := json.Marshal(map[string]string{
			"first_name": strings.TrimSpace(body.FirstName),
			"last_name":  strings.TrimSpace(body.LastName),
			"username":   strings.TrimSpace(body.Username),
			"photo_url":  strings.TrimSpace(body.PhotoURL),
		})
		if err != nil {
			jsonError(w, "internal error", http.StatusInternalServerError)
			return
		}
		profileData = string(profileBytes)
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

	if len(profileData) > 2000 {
		jsonError(w, "profile_data максимум 2000 символов", http.StatusBadRequest)
		return
	}
	// The column is jsonb: an empty or malformed blob is caller error, and
	// without this it reaches the driver as ''::jsonb and surfaces as a 500.
	if profileData == "" {
		profileData = "{}"
	} else if !json.Valid([]byte(profileData)) {
		jsonErrorCode(w, "profile_data должен быть корректным JSON", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	oa := model.CustomerOAuth{
		CustomerID:  id,
		Provider:    body.Provider,
		OAuthID:     oauthID,
		ProfileData: profileData,
	}

	if err := h.store.LinkOAuth(r.Context(), id, oa); err != nil {
		if errors.Is(err, store.ErrOAuthAlreadyLinked) {
			jsonErrorCode(w, "этот Telegram уже подключён к другому аккаунту", http.StatusConflict, "CONFLICT")
			return
		}
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// hasTelegram reports whether the customer's OAuth links include a Telegram
// binding. Used by the CreateOrder gate.
func hasTelegram(links []model.CustomerOAuth) bool {
	for _, oa := range links {
		if oa.Provider == "telegram" {
			return true
		}
	}
	return false
}

// CreateOrder handles POST /api/store/orders — creates an order from checkout
// or individual order form. Requires customer auth + CSRF.
//
// Prices are NEVER taken from the client. The server loads prices from the DB
// by product_id and recalculates price_minor and total_minor.
func (h *CustomerHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.CustomerID(r)

	// Telegram is required to place an order: the customer must have a
	// verified Telegram binding before they can check out. This is the
	// backend gate — the storefront disables the button and explains,
	// but the API enforces it (architectural integrity).
	links, err := h.store.GetCustomerOAuth(r.Context(), customerID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !hasTelegram(links) {
		jsonErrorCode(w, "подключите Telegram, чтобы сделать заказ", http.StatusForbidden, "TELEGRAM_REQUIRED")
		return
	}

	// Limit body size before reading (defence in depth — JSON bodies capped at 1 MiB)
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	// Read body for both validation and idempotency hash
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	var req struct {
		Type           string   `json:"type"`
		Phone          string   `json:"phone"`
		City           string   `json:"city"`
		DeliveryMethod string   `json:"delivery_method"`
		PaymentMethod  string   `json:"payment_method"`
		Street         string   `json:"street"`
		House          string   `json:"house"`
		Apartment      string   `json:"apartment"`
		Comment        string   `json:"comment"`
		Height         *float64 `json:"height"`
		Weight         *float64 `json:"weight"`
		DeliveryTime   []string `json:"delivery_time"`
		Photos         []string `json:"photos"`
		Items          []struct {
			ProductID    int64                  `json:"product_id"`
			SizeLabel    string                 `json:"size_label"`
			Quantity     int                    `json:"quantity"`
			Measurements map[string]interface{} `json:"measurements,omitempty"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rawBody, &req); err != nil {
		jsonErrorCode(w, "invalid JSON body", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	// ── Validation ──
	if req.Type == "" {
		req.Type = "cart"
	}
	if req.Type != "cart" && req.Type != "individual" {
		jsonErrorCode(w, "type must be 'cart' or 'individual'", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	// Phone is required on every checkout (cart / individual /
	// preorder). Customers are asked to type it at every order so
	// the historical record on the order is always the number they
	// actually wanted to be reached at, even if their profile phone
	// is later edited. We accept a single optional leading '+' plus 7-15
	// digits, which covers MDL (+373), RUS (+7), UA (+380), EU
	// (+NN…) — anything else gets a 400.
	req.Phone = strings.TrimSpace(req.Phone)
	if req.Phone == "" {
		jsonErrorCode(w, "phone is required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if !phoneRE.MatchString(req.Phone) {
		jsonErrorCode(w, "phone must be +<countrycode> followed by 7-15 digits", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if len(req.City) == 0 || len(req.City) > 100 {
		jsonErrorCode(w, "city is required (max 100 chars)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if len(req.DeliveryMethod) == 0 || len(req.DeliveryMethod) > 50 {
		jsonErrorCode(w, "delivery_method is required (max 50 chars)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if len(req.PaymentMethod) == 0 || len(req.PaymentMethod) > 50 {
		jsonErrorCode(w, "payment_method is required (max 50 chars)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	// Defence-in-depth: the frontend disables the cash-on-delivery
	// radio when delivery_method = "bus" via
	// apps/store/src/lib/deliveryRules.ts, but a custom curl or a
	// stale frontend bundle could still POST this combination.
	//
	// Individual orders (type = "individual") are an exception:
	// the customer rides the bus themselves and hands cash to the
	// driver or the meet-up person, so cod+bus is a real, valid
	// combination. The storefront applies the same exception in
	// the disabled-radio logic.
	if req.PaymentMethod == "cod" && req.DeliveryMethod == "bus" && req.Type != "individual" {
		jsonErrorCode(w, "cash on delivery is not available for bus delivery", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	// Defence-in-depth for the moldovaPost + PMR rule: Moldova Post
	// doesn't serve Transnistria (PMR has its own postal service).
	// The frontend hides the radio for PMR cities via
	// isDeliveryBlocked() in lib/deliveryRules.ts; this check is the
	// server-side safety net.
	if req.DeliveryMethod == "moldovaPost" {
		// PNR_CITIES mirror from apps/store/src/lib/deliveryRules.ts.
		// Drift between the two lists is caught in CI by
		// TestPNRCitiesGoAndTSAline in pnr_cities_parity_test.go —
		// adding a city to one without the other fails the build, so
		// the human "review them together" contract is no longer
		// load-bearing.
		pnrCities := map[string]bool{
			"тирасполь": true, "бендеры": true, "дубоссары": true, "рыбница": true,
			"григориополь": true, "днестровск": true, "каменка": true, "слободзея": true,
			"парканы": true, "ближний хутор": true, "красное": true, "новотираспольский": true,
			"терновка": true, "маяк": true, "суклея": true,
		}
		// Normalise for parity with apps/store/src/lib/deliveryRules.ts
		// (normaliseCity): trim whitespace, lower-case, ё→е. Without
		// trim, a city like " Тирасполь " (paste / IME padding) passes
		// Go but is blocked by the frontend, diverging from the
		// promised parity. Round-3 review D2.
		normalised := strings.ToLower(strings.TrimSpace(req.City))
		normalised = strings.ReplaceAll(normalised, "ё", "е")
		if pnrCities[normalised] {
			jsonErrorCode(w, "moldova post is not available for Transnistria cities", http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
	}
	if len(req.Street) > 200 {
		jsonErrorCode(w, "street is too long (max 200)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if len(req.House) > 20 {
		jsonErrorCode(w, "house is too long (max 20)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if len(req.Apartment) > 20 {
		jsonErrorCode(w, "apartment is too long (max 20)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if len(req.Comment) > 1000 {
		jsonErrorCode(w, "comment is too long (max 1000)", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	// Height/Weight (optional, individual orders): bound to a sane human
	// range. Go's encoding/json already rejects NaN/Inf, but absurd
	// finite values (1e30, negatives) would still reach the DB and the
	// Telegram message. Per CLAUDE.md, validate bounds on every incoming
	// numeric argument, not just strings.
	if req.Height != nil && (*req.Height < minBodyHeightCm || *req.Height > maxBodyHeightCm) {
		jsonErrorCode(w, fmt.Sprintf("height out of range (%g-%g cm)", minBodyHeightCm, maxBodyHeightCm), http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	if req.Weight != nil && (*req.Weight < minBodyWeightKg || *req.Weight > maxBodyWeightKg) {
		jsonErrorCode(w, fmt.Sprintf("weight out of range (%g-%g kg)", minBodyWeightKg, maxBodyWeightKg), http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	// Photos: defence-in-depth. Telegram notifier neutralises path
	// traversal via filepath.Base, and the body is bounded at 1 MiB,
	// but an unbounded list/format still lets a client queue hundreds
	// of failed os.Open calls in the background notifier. Cap and
	// whitelist the format so only `/uploads/<file>` references pass.
	if len(req.Photos) > 10 {
		jsonErrorCode(w, "photos: max 10 entries", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	for i, p := range req.Photos {
		if len(p) > 200 {
			jsonErrorCode(w, fmt.Sprintf("photos[%d]: path too long", i), http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
		if !strings.HasPrefix(p, "/uploads/") {
			jsonErrorCode(w, fmt.Sprintf("photos[%d]: must start with /uploads/", i), http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
	}
	// DeliveryTime: cap both the count and the per-element length.
	// Same reasoning as Photos[] — strings that flow into orders TEXT
	// columns and Telegram messages must have a max length, per
	// CLAUDE.md. The valid set in i18n is {fast, medium, slow}, but
	// we cap generously (10 entries × 32 chars) to allow future
	// options without changing the wire format.
	if len(req.DeliveryTime) > maxDeliveryTimeN {
		jsonErrorCode(w, fmt.Sprintf("delivery_time: max %d entries", maxDeliveryTimeN), http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	for i, t := range req.DeliveryTime {
		if len(t) > maxDeliveryTimeE {
			jsonErrorCode(w, fmt.Sprintf("delivery_time[%d]: too long (max %d)", i, maxDeliveryTimeE), http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
	}
	// Per-item bounds validation. Two layers, ordered by what they
	// protect:
	//
	//   1. Finance-critical bounds (apply to ANY type if items are
	//      present): `len > 50`, `ProductID > 0`, `Quantity 1-99`.
	//      These guard the common stock-decrement and total_minor
	//      paths against crafted input. Previously lifted out of
	//      `if cart` to fix the stock-drain vector (type=individual
	//      + quantity=-100 inflated stock).
	//
	//   2. Presence rule (type-specific): cart MUST have at least one
	//      item (otherwise total_minor is always 0 and nothing is
	//      decremented, defeating the order). Individual orders
	//      intentionally carry no items — the form on
	//      CustomOrderPage.tsx:117-130 submits free-text fields
	//      (height, weight, photos, comment) instead, and the store
	//      layer (order_postgres.go::CreateOrder) treats the empty
	//      list as a no-op (total_minor=0, stock untouched).
	//
	// Splitting these two layers keeps finance-critical checks
	// single-sourced (no per-branch copy-paste drift) while restoring
	// the legitimate individual-without-items path.
	if len(req.Items) > 50 {
		jsonErrorCode(w, "items: max 50 entries", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	for _, it := range req.Items {
		if it.ProductID <= 0 {
			jsonErrorCode(w, "invalid product_id in items", http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
		if it.Quantity < 1 || it.Quantity > 99 {
			jsonErrorCode(w, "quantity out of range (1-99)", http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
		// SizeLabel: cap matches the SaveCart contract (maxSizeLabelLen
		// = 32); without it, the order store and the cart store disagree
		// on the wire-format shape, and a single crafted item can dump
		// ~1 MiB into the orders TEXT column.
		if len(it.SizeLabel) > maxSizeLabelLen {
			jsonErrorCode(w, fmt.Sprintf("size_label too long (max %d)", maxSizeLabelLen), http.StatusBadRequest, "VALIDATION_FAILED")
			return
		}
	}
	if req.Type == "cart" && len(req.Items) == 0 {
		jsonErrorCode(w, "items must have at least 1 entry for cart orders", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	// ── Idempotency ──
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		jsonErrorCode(w, "Idempotency-Key header required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}
	requestHash := store.OrderRequestHash(r.Method, r.URL.Path, string(rawBody), customerID)

	// Build order model (prices calculated server-side in store)
	order := &model.Order{
		Type:           req.Type,
		Phone:          req.Phone,
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

	// Convert request items to model.OrderItem (prices filled by store)
	items := make([]model.OrderItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = model.OrderItem{
			ProductID:    it.ProductID,
			SizeLabel:    it.SizeLabel,
			Quantity:     it.Quantity,
			Measurements: it.Measurements,
		}
	}

	created, err := h.store.CreateOrder(r.Context(), customerID, order, items, idempotencyKey, requestHash)
	if err != nil {
		// Sentinel branches first — these are *expected* business
		// outcomes (concurrent double-click, hash conflict, oversell),
		// not operation failures. Per CLAUDE.md log standard, ERROR
		// means "operation failed" — logging these at ERROR would
		// noise the alerting channel with benign traffic. The
		// catch-all slog.Error at the bottom of this block stays for
		// the genuine 500 path.
		if errors.Is(err, store.ErrIdempotencyHashMismatch) {
			jsonErrorCode(w, "idempotency key reused with different request", http.StatusConflict, "IDEMPOTENCY_REPLAY")
			return
		}
		if errors.Is(err, store.ErrIdempotencyRace) {
			// Concurrent first submit with the same Idempotency-Key
			// won the race. Per CLAUDE.md, the loser should receive
			// the winner's response when the request hash matches
			// (true replay, 201), or 409 IDEMPOTENCY_REPLAY only
			// when the hash differs (a real conflict, not a benign
			// double-click). Re-fetch the stored record and branch.
			rec, ferr := h.store.GetOrderByIdempotencyKey(r.Context(), idempotencyKey, customerID)
			if ferr != nil {
				slog.Error("GetOrderByIdempotencyKey failed", "customer_id", customerID, "error", ferr)
				jsonError(w, "internal error", http.StatusInternalServerError)
				return
			}
			if rec.RequestHash != requestHash {
				// Same key, different request body — true conflict.
				jsonErrorCode(w, "idempotency key reused with different request", http.StatusConflict, "IDEMPOTENCY_REPLAY")
				return
			}
			// True replay: same key + same body. Return the winner's
			// stored response verbatim — same status, same body.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.Status)
			_, _ = w.Write([]byte(rec.ResponseBody))
			return
		}
		if errors.Is(err, store.ErrInsufficientStock) {
			jsonErrorCode(w, "товара нет в наличии", http.StatusConflict, "INSUFFICIENT_STOCK")
			return
		}
		// Sentinel: the cart contained a product_id that no longer
		// exists in `products` (admin deleted it, or the client built
		// the cart from a stale frontend cache). This is a client
		// error, not a server failure — 400 with a code the frontend
		// can branch on to force a catalog refresh.
		if errors.Is(err, store.ErrProductNotFound) {
			slog.Info("CreateOrder rejected stale product in cart",
				"customer_id", customerID, "error", err)
			// Reviewer finding #5 (P1): PRODUCT_NOT_FOUND was
			// an ad-hoc error code, not in the reserved list
			// at CLAUDE.md:172. Switched to NOT_FOUND (already
			// reserved) so the SPA keeps a single branch on
			// this code path. The message is the same, so
			// any client matching on text (which they
			// shouldn't) is unchanged.
			jsonErrorCode(w, "one or more products in your cart no longer exist; please refresh the catalog",
				http.StatusBadRequest, "NOT_FOUND")
			return
		}
		// Catch-all 500: this is the only path that warrants
		// slog.Error under the CLAUDE.md log standard. Sentinels
		// above (idempotency, oversell) are expected business
		// outcomes and intentionally silent at the log level.
		slog.Error("CreateOrder failed", "customer_id", customerID, "error", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)

	// Keep the customer's profile phone in sync with whatever they
	// typed at checkout. This is best-effort — a sync failure must
	// not fail the order (the order is already created and the
	// customer is just sitting with a stale profile row until their
	// next successful edit).
	//
	// Reviewer finding #6 (P1) and re-review finding N1: the
	// pre-fix block did `GetCustomer` + (if different)
	// `UpdateCustomer` — two DB round-trips on the busiest write
	// path. Replaced with a single conditional UPDATE
	// (`UpdateCustomerPhoneIfChanged`) that no-ops when the
	// stored phone already matches.
	//
	// The call uses `context.Background()` (not `r.Context()`)
	// because the order is already `WriteHeader(201)`-ed by this
	// point. A client disconnect must not cancel the profile
	// sync — the order is committed, the profile would be left
	// stale until the customer's next edit. The Telegram block
	// below (~line 1104) uses the same pattern.
	if h.store != nil {
		if _, uerr := h.store.UpdateCustomerPhoneIfChanged(context.Background(), customerID, req.Phone); uerr != nil {
			slog.Warn("update customer phone after order failed", "customer_id", customerID, "error", uerr)
		}
	}

	// Notify managers via Telegram (background, fire-and-forget).
	// The deferred recover() previously ate *every* panic silently —
	// a nil pointer dereference inside `formatOrderMessage` (e.g. if
	// `cust` came back as `nil` from `GetCustomer` for any reason)
	// would crash the goroutine without a single log line, which is
	// exactly how the "telegram notifications just don't arrive in
	// prod" bug stayed invisible for so long. Now we log the panic
	// with stack so the failure is recoverable from the logs.
	if h.tgNotifier != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("telegram notify goroutine panicked",
						"order_id", created.ID,
						"customer_id", customerID,
						"panic", fmt.Sprintf("%v", r),
						"stack", string(debug.Stack()),
					)
				}
			}()
			cust, err := h.store.GetCustomer(context.Background(), customerID)
			if err != nil {
				slog.Warn("telegram notify: GetCustomer failed",
					"order_id", created.ID,
					"customer_id", customerID,
					"error", err)
				return
			}
			if cust == nil {
				// GetCustomer returns (nil, nil) for missing rows
				// per customer_postgres.go::GetCustomer. Real
				// customer_id always exists in the DB at this
				// point (we just created the order under it), so
				// a nil customer here is a programmer error or a
				// race we want to see, not a normal outcome.
				slog.Warn("telegram notify: customer not found",
					"order_id", created.ID,
					"customer_id", customerID)
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
		jsonError(w, "only image files allowed (png, jpg, webp)", http.StatusBadRequest)
		return
	}
	if err := validateImageContent(file); err != nil {
		jsonError(w, "file content is not a supported image (png, jpg, webp)", http.StatusBadRequest)
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
		slog.Warn("UploadOrderPhoto: thumbnail failed", "file", safeName, "error", err)
	}

	url := "/uploads/" + safeName
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// ListOrders returns the authenticated customer's order history, newest first.
// Query params: page (1-based, default 1), per_page (1-100, default 20).
//
// The store fills model.Order with ALL order fields (type, city, delivery /
// payment method, address, photos, height, weight, etc.) plus a fully
// populated Items[] (with product slug, name and a single representative
// image) via batched queries. We hand model.Order back to the client
// directly — its json tags already match the wire schema — so adding
// new fields later only requires changing the model + store, not the
// handler.
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

	// Scrub admin-only fields before sending. CustomerID leaks the account
	// id; CustomerEmail / CustomerFirstName are populated by the admin
	// list path, not the customer self-list path, but strip them anyway
	// so they never appear in this response.
	for i := range orders {
		orders[i].CustomerID = 0
		orders[i].CustomerEmail = ""
		orders[i].CustomerFirstName = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders":   orders,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// ── Cart & Favorites handlers ──

// phoneRE matches a single optional leading '+' followed by 7-15
// digits. Covers MDL (+373...), RUS (+7...), UA (+380...), EU
// (+NN...) without being so loose it accepts nonsense like spaces,
// letters or wildly long numbers.
var phoneRE = regexp.MustCompile(`^\+?\d{7,15}$`)

const (
	maxCartItems      = 200
	maxFavoritesItems = 200
	maxSizeLabelLen   = 32
	maxQuantity       = 999
	maxDeliveryTimeN  = 10
	maxDeliveryTimeE  = 32

	// Body-measurement bounds for individual orders (defence-in-depth
	// against absurd finite values reaching the DB / Telegram message).
	minBodyHeightCm = 50.0
	maxBodyHeightCm = 300.0
	minBodyWeightKg = 20.0
	maxBodyWeightKg = 400.0
)

type cartItemReq struct {
	ProductID    int                    `json:"product_id"`
	SizeLabel    string                 `json:"size_label"`
	Quantity     int                    `json:"quantity"`
	Measurements map[string]interface{} `json:"measurements,omitempty"`
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
			ProductID:    it.ProductID,
			SizeLabel:    it.SizeLabel,
			Quantity:     it.Quantity,
			Measurements: it.Measurements,
		})
	}

	if err := h.store.SaveCustomerCart(r.Context(), id, items); err != nil {
		// Sentinel branch: a stale-cart save is a client error (the
		// products in the cart no longer exist in the catalog), not a
		// 500. Without this branch the FK violation would surface as a
		// generic SQLSTATE 23503 and the catch-all below would 500.
		if errors.Is(err, store.ErrProductNotFound) {
			slog.Info("SaveCart rejected stale product",
				"customer_id", id, "error", err)
			// See the CreateOrder branch above for the
			// NOT_FOUND vs PRODUCT_NOT_FOUND rationale.
			jsonErrorCode(w, "one or more products in your cart no longer exist; please refresh the catalog",
				http.StatusBadRequest, "NOT_FOUND")
			return
		}
		slog.Error("SaveCart failed", "customer_id", id, "error", err)
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

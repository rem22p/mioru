package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"mioru/internal/auth"
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
}

// CustomerHandler handles store customer auth & profile.
type CustomerHandler struct {
	store  customerStore
	secret string
	expiry int
}

func NewCustomerHandler(s customerStore, secret string, expiry int) *CustomerHandler {
	return &CustomerHandler{store: s, secret: secret, expiry: expiry}
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

type customerTokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
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
	tok, err := auth.CreateToken(fmt.Sprintf("%d", cust.ID), auth.TokenTypeCustomer, h.secret, h.expiry)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(customerTokenResp{AccessToken: tok, TokenType: "bearer"})
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

	tok, err := auth.CreateToken(fmt.Sprintf("%d", cust.ID), auth.TokenTypeCustomer, h.secret, h.expiry)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customerTokenResp{AccessToken: tok, TokenType: "bearer"})
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

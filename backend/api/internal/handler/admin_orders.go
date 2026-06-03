package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"mioru/internal/middleware"
	"mioru/internal/model"
)

// adminOrderStore is the subset consumed by admin order handlers.
type adminOrderStore interface {
	ListAllOrders(ctx context.Context, page, perPage int, status, orderType string) ([]model.Order, int, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status string) error
}

// AdminOrderHandler handles admin order listing and status updates.
type AdminOrderHandler struct {
	store adminOrderStore
}

func NewAdminOrderHandler(s adminOrderStore) *AdminOrderHandler {
	return &AdminOrderHandler{store: s}
}

// ListAll handles GET /api/admin/orders — returns paginated orders.
// Query params: page, per_page, status, type (optional filters).
func (h *AdminOrderHandler) ListAll(w http.ResponseWriter, r *http.Request) {
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
	status := q.Get("status")
	orderType := q.Get("type")

	orders, total, err := h.store.ListAllOrders(r.Context(), page, perPage, status, orderType)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Ensure orders is never null (nil slice → JSON null, breaks frontend)
	if orders == nil {
		orders = []model.Order{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders":   orders,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// UpdateStatus handles PATCH /api/admin/orders/{id}/status.
func (h *AdminOrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonErrorCode(w, "invalid order id", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Status == "" {
		jsonErrorCode(w, "status is required", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	if err := h.store.UpdateOrderStatus(r.Context(), orderID, req.Status); err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

var _ = middleware.CustomerID

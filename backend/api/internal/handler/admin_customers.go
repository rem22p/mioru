package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"mioru/internal/store"
)

// adminCustomerStore is the storage interface that AdminCustomerHandler
// needs. We keep it narrow (one method per endpoint) so the store
// fake in unit tests stays trivial — the integration tests get the
// real *store.PostgresStore.
type adminCustomerStore interface {
	ListCustomers(ctx context.Context, search string, page, perPage int) ([]store.AdminCustomerRow, int, error)
	GetCustomerFullDetail(ctx context.Context, id int64) (*store.AdminCustomerDetail, error)
}

// AdminCustomerHandler exposes the admin "Customers" workspace
// endpoints: a paginated list of customers and a per-customer
// detail with order history.
type AdminCustomerHandler struct {
	store adminCustomerStore
}

// NewAdminCustomerHandler builds a handler. The store parameter is
// any type that implements adminCustomerStore — production wires
// *store.PostgresStore, tests can use a fake.
func NewAdminCustomerHandler(s adminCustomerStore) *AdminCustomerHandler {
	return &AdminCustomerHandler{store: s}
}

// List handles GET /api/admin/customers.
//
// Query params:
//   - page (int, default 1)
//   - per_page (int, default 20, max 100)
//   - search (string, optional; matches email/first_name/last_name
//     case-insensitively)
//
// Response: { customers: [...], total: int, page: int, per_page: int }.
//
// Both admin and super_admin are allowed; the route is mounted
// under adminOnly at the main package level.
func (h *AdminCustomerHandler) List(w http.ResponseWriter, r *http.Request) {
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
	search := q.Get("search")

	rows, total, err := h.store.ListCustomers(r.Context(), search, page, perPage)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	// Always return a non-nil slice so the JSON encodes `[]` not
	// `null` — frontend table code can iterate without a null check.
	if rows == nil {
		rows = []store.AdminCustomerRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"customers": rows,
		"total":     total,
		"page":      page,
		"per_page":  perPage,
	})
}

// Detail handles GET /api/admin/customers/{id}.
//
// Response: full customer profile + order history + Telegram link.
// 404 if the customer doesn't exist (vs the generic 500 a missing
// row would otherwise produce via the pgx ErrNoRows path).
func (h *AdminCustomerHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonErrorCode(w, "invalid customer id", http.StatusBadRequest, "VALIDATION_FAILED")
		return
	}

	d, err := h.store.GetCustomerFullDetail(r.Context(), id)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if d == nil {
		jsonErrorCode(w, "customer not found", http.StatusNotFound, "NOT_FOUND")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(d)
}

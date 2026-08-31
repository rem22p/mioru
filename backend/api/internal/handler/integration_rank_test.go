package handler_test

import (
	"net/http"
	"testing"

	"mioru/internal/cookieauth"
)

func (e *env) createRankableProduct(t *testing.T, admin *session, slug string) int64 {
	t.Helper()
	rr := e.doMultipart(t, e.wrapAdmin(e.productH.Create), http.MethodPost, "/api/admin/products",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie}, map[string][]string{
			"slug":           {slug},
			"category_id":    {"1"},
			"name":           {"Rankable"},
			"price":          {"1000"},
			"status":         {"in_stock"},
			"stock_quantity": {"5"},
			"sizes[]":        {"M"},
		})
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %s: want 201, got %d (%s)", slug, rr.Code, rr.Body.String())
	}
	prod, err := e.st.GetProduct(t.Context(), slug)
	if err != nil {
		t.Fatalf("GetProduct(%s): %v", slug, err)
	}
	return prod.ID
}

// The rank column is chosen by the first entry — a mixed batch would silently
// write every row into one column, so the handler must reject it.
func TestIntegrationAdminUpdateRanksRejectsMixedBatch(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "adminrank", "admin")

	id1 := e.createRankableProduct(t, admin, "rank-mixed-a")
	id2 := e.createRankableProduct(t, admin, "rank-mixed-b")

	body := []any{
		map[string]any{"id": id1, "rank": 1, "key": "popularity_rank"},
		map[string]any{"id": id2, "rank": 2, "key": "popularity_rank_preorder"},
	}
	rr := e.do(t, e.wrapAdmin(e.productH.UpdateRanks), http.MethodPut, "/api/admin/products/rank",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, body: body})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("mixed rank batch: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}

	// And a homogeneous preorder batch still goes through.
	body = []any{
		map[string]any{"id": id1, "rank": 5, "key": "popularity_rank_preorder"},
		map[string]any{"id": id2, "rank": 6, "key": "popularity_rank_preorder"},
	}
	rr = e.do(t, e.wrapAdmin(e.productH.UpdateRanks), http.MethodPut, "/api/admin/products/rank",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, body: body})
	if rr.Code != http.StatusOK {
		t.Fatalf("homogeneous batch: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
}

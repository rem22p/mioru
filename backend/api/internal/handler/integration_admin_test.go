package handler_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"mioru/internal/cookieauth"
)

// doMultipart mirrors env.do, but sends a multipart/form-data body instead of
// JSON. The admin product Create/Update handlers call r.ParseMultipartForm and
// read r.FormValue/r.Form, so they require a real multipart request — the
// JSON-only env.do driver cannot exercise them. Cookie/CSRF handling matches
// env.do exactly (auth cookie + matching CSRF cookie + X-CSRF-Token header).
func (e *env) doMultipart(t *testing.T, h http.Handler, method, target string, o reqOpts, fields map[string][]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, vals := range fields {
		for _, v := range vals {
			if err := mw.WriteField(name, v); err != nil {
				t.Fatalf("write multipart field %q: %v", name, err)
			}
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if o.sess != nil {
		req.AddCookie(o.sess.authCookie)
		if o.csrfCookieName != "" {
			req.AddCookie(&http.Cookie{Name: o.csrfCookieName, Value: o.sess.csrfValue})
			if o.badCSRF {
				req.Header.Set("X-CSRF-Token", "wrong-value")
			} else {
				req.Header.Set("X-CSRF-Token", o.sess.csrfValue)
			}
		}
	}
	if o.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", o.idempotencyKey)
	}
	for k, v := range o.pathValues {
		req.SetPathValue(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestIntegrationAdminCreateAndGetProduct(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "admincrud", "admin")

	// ProductHandler.Create reads a multipart form (not JSON): slug/name/
	// category_id are required; status is canonicalized by normalizeProductStatus.
	// "in_stock" is a known-valid value under the products_status_chk constraint.
	create := map[string][]string{
		"slug":           {"admin-prod-1"},
		"category_id":    {"1"}, // seeded by migration 002; categories are not truncated between tests
		"brand":          {"B"},
		"name":           {"N"},
		"price":          {"1000"},
		"color":          {"blue"},
		"status":         {"in_stock"},
		"stock_quantity": {"5"},
		"sizes[]":        {"M"},
	}
	rr := e.doMultipart(t, e.wrapAdmin(e.productH.Create), http.MethodPost, "/api/admin/products",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie}, create)
	if rr.Code != http.StatusCreated {
		t.Fatalf("admin create product: want 201, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr2 := e.do(t, e.wrapAdmin(e.productH.Get), http.MethodGet, "/api/admin/products/admin-prod-1",
		reqOpts{sess: admin, pathValues: map[string]string{"slug": "admin-prod-1"}})
	if rr2.Code != http.StatusOK {
		t.Fatalf("admin get product: want 200, got %d (%s)", rr2.Code, rr2.Body.String())
	}
}

func TestIntegrationAdminRouteRejectsNoSession(t *testing.T) {
	e := newEnv(t)
	rr := e.do(t, e.wrapAdmin(e.productH.List), http.MethodGet, "/api/admin/products", reqOpts{})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("admin route, no session: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationAdminRouteRejectsCustomerToken(t *testing.T) {
	e := newEnv(t)
	// A customer JWT (typ "customer") must not satisfy an admin route even if
	// presented under the admin cookie name: AuthMW wants typ "user".
	cust, _ := e.customerSession(t, "sneaky@ex.com")
	adminShaped := &session{
		authCookie: &http.Cookie{Name: cookieauth.AdminAuthCookie, Value: cust.authCookie.Value},
		csrfValue:  cust.csrfValue,
	}
	rr := e.do(t, e.wrapAdmin(e.productH.List), http.MethodGet, "/api/admin/products", reqOpts{sess: adminShaped})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("admin route with customer-typ token: want 401, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationAdminCreateProductCSRFGate(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "admincsrf", "admin")
	// CSRF middleware runs BEFORE the handler, so the body need not be valid —
	// the 403 fires on the bad X-CSRF-Token before ParseMultipartForm is reached.
	rr := e.do(t, e.wrapAdmin(e.productH.Create), http.MethodPost, "/api/admin/products",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, badCSRF: true, body: map[string]any{"slug": "x"}})
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin create, bad CSRF: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
}

func TestIntegrationNonSuperAdminCannotListUsers(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "plainadmin", "admin") // admin, NOT super_admin
	rr := e.do(t, e.wrapSuperAdmin(e.authH.ListUsers), http.MethodGet, "/api/admin/users", reqOpts{sess: admin})
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin (not super) listing users: want 403, got %d (%s)", rr.Code, rr.Body.String())
	}
}

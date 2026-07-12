package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"mioru/internal/cookieauth"
)

// TestIntegrationAdminListUsersHappy: a super-admin lists users and gets back a
// bare JSON array (NOT a paginated {items:...} envelope) containing the caller.
func TestIntegrationAdminListUsersHappy(t *testing.T) {
	e := newEnv(t)
	super := e.userSession(t, "super", "super_admin")

	rr := e.do(t, e.wrapSuperAdmin(e.authH.ListUsers), http.MethodGet, "/api/admin/users", reqOpts{sess: super})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListUsers: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	// Pin the bare-array shape: decode straight into a slice. A paginated
	// {items:...} object would fail to unmarshal here.
	var users []map[string]any
	decode(t, rr, &users)
	found := false
	for _, u := range users {
		if u["username"] == "super" {
			found = true
		}
	}
	if !found {
		t.Errorf("ListUsers did not include the 'super' user; got %+v", users)
	}
}

// TestIntegrationAdminDeleteUserHappy: a super-admin deletes another user → 204,
// and the row is gone.
func TestIntegrationAdminDeleteUserHappy(t *testing.T) {
	e := newEnv(t)
	super := e.userSession(t, "super", "super_admin")
	_ = e.userSession(t, "victim", "admin") // create the target row

	rr := e.do(t, e.wrapSuperAdmin(e.authH.DeleteUser), http.MethodDelete, "/api/admin/users/victim",
		reqOpts{sess: super, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"username": "victim"}})
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DeleteUser: want 204, got %d (%s)", rr.Code, rr.Body.String())
	}
	u, err := e.st.GetUser(context.Background(), "victim")
	if err != nil {
		t.Fatalf("GetUser after delete: %v", err)
	}
	if u != nil {
		t.Errorf("victim still exists after delete: %+v", u)
	}
}

// TestIntegrationAdminDeleteUserSelf: deleting yourself → 400 VALIDATION_FAILED.
func TestIntegrationAdminDeleteUserSelf(t *testing.T) {
	e := newEnv(t)
	super := e.userSession(t, "super", "super_admin")

	rr := e.do(t, e.wrapSuperAdmin(e.authH.DeleteUser), http.MethodDelete, "/api/admin/users/super",
		reqOpts{sess: super, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"username": "super"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("DeleteUser self: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

// TestIntegrationAdminDeleteUserNotFound: deleting a non-existent user → 404 NOT_FOUND.
func TestIntegrationAdminDeleteUserNotFound(t *testing.T) {
	e := newEnv(t)
	super := e.userSession(t, "super", "super_admin")

	rr := e.do(t, e.wrapSuperAdmin(e.authH.DeleteUser), http.MethodDelete, "/api/admin/users/ghost",
		reqOpts{sess: super, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"username": "ghost"}})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("DeleteUser ghost: want 404, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", env.Code)
	}
}

// TestIntegrationAdminCategoriesTree: GET categories returns the seeded tree
// (non-empty), surviving the per-test data truncation.
func TestIntegrationAdminCategoriesTree(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "catadmin", "admin")

	rr := e.do(t, e.wrapAdmin(e.productH.Categories), http.MethodGet, "/api/admin/categories", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("Categories tree: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var cats []map[string]any
	decode(t, rr, &cats)
	if len(cats) == 0 {
		t.Errorf("Categories tree is empty; seeded categories should survive truncation")
	}
}

// TestIntegrationAdminCategoriesFlat: ?flat=1 returns a flat list whose entries
// each carry a parent_id key (the *int field always serializes, null at root).
func TestIntegrationAdminCategoriesFlat(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "catflatadmin", "admin")

	rr := e.do(t, e.wrapAdmin(e.productH.Categories), http.MethodGet, "/api/admin/categories?flat=1", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("Categories flat: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var cats []map[string]any
	decode(t, rr, &cats)
	if len(cats) == 0 {
		t.Fatalf("Categories flat is empty; seeded categories should survive truncation")
	}
	if _, ok := cats[0]["parent_id"]; !ok {
		t.Errorf("flat category entry missing parent_id key: %+v", cats[0])
	}

	// Migration 018 added Eyewear (25/26/27) — assert each is present.
	assertFlatCategoryID(t, cats, 25, "eyewear")
	assertFlatCategoryID(t, cats, 26, "sunglasses")
	assertFlatCategoryID(t, cats, 27, "optical-glasses")
}

func assertFlatCategoryID(t *testing.T, cats []map[string]any, wantID int, wantSlug string) {
	t.Helper()
	for _, c := range cats {
		if id, ok := c["id"]; ok {
			switch v := id.(type) {
			case float64:
				if int(v) == wantID {
					if s, _ := c["slug"].(string); s != wantSlug {
						t.Errorf("category %d slug = %q, want %q", wantID, s, wantSlug)
					}
					return
				}
			}
		}
	}
	t.Errorf("eyewear category id=%d (%s) not found in flat categories", wantID, wantSlug)
}

// TestIntegrationAdminListProducts: the admin product list paginates under the
// "products" key with a total count.
func TestIntegrationAdminListProducts(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "listadmin", "admin")
	seedProduct(t, e, "list-a", 100, 5)
	seedProduct(t, e, "list-b", 200, 5)

	rr := e.do(t, e.wrapAdmin(e.productH.List), http.MethodGet, "/api/admin/products?per_page=20", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("List products: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Products []json.RawMessage `json:"products"`
		Total    int               `json:"total"`
		Page     int               `json:"page"`
		PerPage  int               `json:"per_page"`
	}
	decode(t, rr, &resp)
	if resp.Total < 2 {
		t.Errorf("total = %d, want >= 2", resp.Total)
	}
	if len(resp.Products) < 2 {
		t.Errorf("products len = %d, want >= 2", len(resp.Products))
	}
}

// TestIntegrationAdminUpdateProductHappy: updating an existing product's name
// → 200 and the change is persisted.
func TestIntegrationAdminUpdateProductHappy(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "updadmin", "admin")
	seedProduct(t, e, "upd-a", 100, 5)

	rr := e.doMultipart(t, e.wrapAdmin(e.productH.Update), http.MethodPut, "/api/admin/products/upd-a",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"slug": "upd-a"}},
		map[string][]string{"slug": {"upd-a"}, "name": {"Renamed"}, "category_id": {"1"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("Update product: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	p, err := e.st.GetProduct(context.Background(), "upd-a")
	if err != nil || p == nil {
		t.Fatalf("GetProduct after update: %v / %v", p, err)
	}
	if p.Name != "Renamed" {
		t.Errorf("name after update = %q, want %q", p.Name, "Renamed")
	}
}

// TestIntegrationAdminUpdateProductNotFound: updating a non-existent slug → 404.
func TestIntegrationAdminUpdateProductNotFound(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "updnfadmin", "admin")

	rr := e.doMultipart(t, e.wrapAdmin(e.productH.Update), http.MethodPut, "/api/admin/products/ghost",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"slug": "ghost"}},
		map[string][]string{"slug": {"ghost"}, "name": {"X"}, "category_id": {"1"}})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Update missing product: want 404, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", env.Code)
	}
}

// TestIntegrationAdminUpdateProductDuplicateSlug: renaming a product onto an
// already-used slug → 409 CONFLICT.
func TestIntegrationAdminUpdateProductDuplicateSlug(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "upddupadmin", "admin")
	seedProduct(t, e, "dup-a", 100, 5)
	seedProduct(t, e, "dup-b", 100, 5)

	rr := e.doMultipart(t, e.wrapAdmin(e.productH.Update), http.MethodPut, "/api/admin/products/dup-b",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"slug": "dup-b"}},
		map[string][]string{"slug": {"dup-a"}, "name": {"X"}, "category_id": {"1"}})
	if rr.Code != http.StatusConflict {
		t.Fatalf("Update to dup slug: want 409, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "CONFLICT" {
		t.Errorf("code = %q, want CONFLICT", env.Code)
	}
}

// TestIntegrationAdminDeleteProductHappy: deleting a product → 200 {"ok":true}
// and the row is gone.
func TestIntegrationAdminDeleteProductHappy(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "deladmin", "admin")
	seedProduct(t, e, "del-1", 100, 5)

	rr := e.do(t, e.wrapAdmin(e.productH.Delete), http.MethodDelete, "/api/admin/products/del-1",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"slug": "del-1"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("Delete product: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		OK bool `json:"ok"`
	}
	decode(t, rr, &resp)
	if !resp.OK {
		t.Errorf("Delete product response ok = false, want true")
	}
	// GetProduct does NOT return (nil, nil) for a missing slug (unlike GetUser):
	// it returns an error "product not found: <slug>". That error is the
	// store-level confirmation the row is gone.
	p, err := e.st.GetProduct(context.Background(), "del-1")
	if p != nil {
		t.Errorf("del-1 still exists after delete: %+v", p)
	}
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("GetProduct after delete: want a 'not found' error, got product=%v err=%v", p, err)
	}
}

// TestIntegrationAdminDeleteProductNotFound: deleting a non-existent slug → 404.
func TestIntegrationAdminDeleteProductNotFound(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "delnfadmin", "admin")

	rr := e.do(t, e.wrapAdmin(e.productH.Delete), http.MethodDelete, "/api/admin/products/ghost",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie, pathValues: map[string]string{"slug": "ghost"}})
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Delete missing product: want 404, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", env.Code)
	}
}

// TestIntegrationAdminListAllOrdersJoinsCustomer: the admin order list joins the
// customer, exposing customer_email on each row.
func TestIntegrationAdminListAllOrdersJoinsCustomer(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "buyer@ex.com")
	pid := seedProduct(t, e, "ord-list-admin", 500, 10)

	createRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, idempotencyKey: "e-1", body: orderBody(pid, 1)})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", createRR.Code, createRR.Body.String())
	}

	admin := e.userSession(t, "ordlistadmin", "admin")
	rr := e.do(t, e.wrapAdmin(e.adminOrdH.ListAll), http.MethodGet, "/api/admin/orders", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListAll orders: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Orders []map[string]any `json:"orders"`
		Total  int              `json:"total"`
	}
	decode(t, rr, &resp)
	if resp.Total < 1 {
		t.Fatalf("total = %d, want >= 1", resp.Total)
	}
	if len(resp.Orders) == 0 {
		t.Fatalf("orders is empty")
	}
	if resp.Orders[0]["customer_email"] != "buyer@ex.com" {
		t.Errorf("customer_email = %v, want buyer@ex.com (JOIN must populate it)", resp.Orders[0]["customer_email"])
	}
}

// TestIntegrationAdminUpdateStatusInvalid: a status outside the allowed set →
// 400 VALIDATION_FAILED.
func TestIntegrationAdminUpdateStatusInvalid(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "statusinv@ex.com")
	pid := seedProduct(t, e, "ord-status-inv", 500, 10)
	createRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, idempotencyKey: "si-1", body: orderBody(pid, 1)})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", createRR.Code, createRR.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	decode(t, createRR, &created)

	admin := e.userSession(t, "statusinvadmin", "admin")
	target := "/api/admin/orders/" + itoa(created.ID) + "/status"
	rr := e.do(t, e.wrapAdmin(e.adminOrdH.UpdateStatus), http.MethodPatch, target,
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie,
			pathValues: map[string]string{"id": itoa(created.ID)}, body: map[string]any{"status": "bogus"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("UpdateStatus bogus: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

// TestIntegrationAdminUpdateStatusNonNumericId: a non-numeric order id →
// 400 VALIDATION_FAILED.
func TestIntegrationAdminUpdateStatusNonNumericId(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "statusbadid", "admin")

	rr := e.do(t, e.wrapAdmin(e.adminOrdH.UpdateStatus), http.MethodPatch, "/api/admin/orders/abc/status",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie,
			pathValues: map[string]string{"id": "abc"}, body: map[string]any{"status": "shipped"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("UpdateStatus non-numeric id: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if env.Code != "VALIDATION_FAILED" {
		t.Errorf("code = %q, want VALIDATION_FAILED", env.Code)
	}
}

// TestIntegrationAdminUploadPNG: a valid PNG uploads → 200 with a
// /uploads/<name>.png url.
func TestIntegrationAdminUploadPNG(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "uploadadmin", "admin")

	rr := e.doMultipartFile(t, e.wrapAdmin(e.productH.Upload), http.MethodPost, "/api/admin/upload",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie},
		"file", "img.png", pngBytes(t))
	if rr.Code != http.StatusOK {
		t.Fatalf("Upload PNG: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		URL string `json:"url"`
	}
	decode(t, rr, &resp)
	if !strings.HasPrefix(resp.URL, "/uploads/") {
		t.Errorf("url %q missing /uploads/ prefix", resp.URL)
	}
	if !strings.HasSuffix(resp.URL, ".png") {
		t.Errorf("url %q missing .png suffix", resp.URL)
	}
}

// TestIntegrationAdminUploadRejectsNonPNG: a non-image extension/content → 400.
func TestIntegrationAdminUploadRejectsNonPNG(t *testing.T) {
	e := newEnv(t)
	admin := e.userSession(t, "uploadadmin2", "admin")

	rr := e.doMultipartFile(t, e.wrapAdmin(e.productH.Upload), http.MethodPost, "/api/admin/upload",
		reqOpts{sess: admin, csrfCookieName: cookieauth.AdminCSRFCookie},
		"file", "doc.txt", []byte("hello"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Upload non-png: want 400, got %d (%s)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationAdminListAllOrdersExposesPhone is the admin-side
// counterpart of the customer ListOrders expansion. The /api/admin/orders
// endpoint powers the manager workspace at apps/admin/src/workspaces/Orders.tsx
// which shows each order's contact phone as a click-to-copy chip. If the
// admin response drops the field, managers see an empty chip and can't
// reach the customer.
//
// The assertion is intentionally strict (exact "+373777908542" string)
// rather than "non-empty" — the test would have caught the regression
// where admin's ListAllOrders handler accidentally selected from a
// table that didn't include `phone` (e.g. joining through `customers`
// without falling back to `orders.phone`).
func TestIntegrationAdminListAllOrdersExposesPhone(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "admin-phone@ex.com")
	pid := seedProduct(t, e, "ord-admin-phone", 500, 10)

	// orderBodyWithPhone sets a specific, recognisable number that we
	// can grep for in the admin response without false positives.
	const wantPhone = "+373777908542"
	createRR := e.do(t, e.wrapCustomer(e.customerH.CreateOrder), http.MethodPost, "/api/store/orders",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie,
			idempotencyKey: "e-admin-phone-1", body: orderBodyWithPhone(pid, 1, wantPhone)})
	if createRR.Code != http.StatusCreated {
		t.Fatalf("CreateOrder: want 201, got %d (%s)", createRR.Code, createRR.Body.String())
	}

	admin := e.userSession(t, "ordphoneadmin", "admin")
	rr := e.do(t, e.wrapAdmin(e.adminOrdH.ListAll), http.MethodGet, "/api/admin/orders", reqOpts{sess: admin})
	if rr.Code != http.StatusOK {
		t.Fatalf("ListAll orders: want 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Orders []map[string]any `json:"orders"`
		Total  int              `json:"total"`
	}
	decode(t, rr, &resp)
	if len(resp.Orders) == 0 {
		t.Fatalf("orders is empty")
	}
	// Find our specific order — the admin list is shared across the
	// whole DB, so we look up by id rather than blindly trusting the
	// first row.
	var got string
	for _, o := range resp.Orders {
		if v, ok := o["phone"].(string); ok && v == wantPhone {
			got = v
			break
		}
	}
	if got == "" {
		t.Errorf("phone field missing or wrong in admin orders response. "+
			"Managers need it to call customers. orders: %+v", resp.Orders)
	}
}

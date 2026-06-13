package handler_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/cookieauth"
)

// doMultipartFile is a sibling of doMultipart that writes ONE file part (plus
// the usual auth/CSRF handling). The upload handler reads r.FormFile("file"),
// which the field-only doMultipart cannot exercise.
func (e *env) doMultipartFile(t *testing.T, h http.Handler, method, target string, o reqOpts, fileField, fileName string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close mw: %v", err)
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
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// pngBytes encodes a real 1×1 PNG so validateImageContent's http.DetectContentType
// sniff (it sniffs the magic header, it does not fully decode) sees image/png.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// TestIntegrationUploadOrderPhotoPNG: a valid PNG uploads → 200 with a
// /uploads/<name>.png url.
func TestIntegrationUploadOrderPhotoPNG(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "uploader@ex.com")

	rr := e.doMultipartFile(t, e.wrapCustomer(e.customerH.UploadOrderPhoto), http.MethodPost,
		"/api/store/orders/upload-photo",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie},
		"file", "photo.png", pngBytes(t))
	if rr.Code != http.StatusOK {
		t.Fatalf("png upload: want 200, got %d (body %q)", rr.Code, rr.Body.String())
	}
	var resp struct {
		URL string `json:"url"`
	}
	decode(t, rr, &resp)
	if !strings.HasPrefix(resp.URL, "/uploads/") {
		t.Fatalf("png upload: url %q missing /uploads/ prefix", resp.URL)
	}
	if !strings.HasSuffix(resp.URL, ".png") {
		t.Fatalf("png upload: url %q missing .png suffix", resp.URL)
	}
}

// TestIntegrationUploadOrderPhotoRejectsNonPNG: a non-image extension/content
// is rejected by the extension allowlist → 400.
func TestIntegrationUploadOrderPhotoRejectsNonPNG(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "uploader2@ex.com")

	rr := e.doMultipartFile(t, e.wrapCustomer(e.customerH.UploadOrderPhoto), http.MethodPost,
		"/api/store/orders/upload-photo",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie},
		"file", "doc.txt", []byte("hello"))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("non-png upload: want 400, got %d (body %q)", rr.Code, rr.Body.String())
	}
}

// TestIntegrationUploadOrderPhotoRequiresFile: a multipart body with no file
// part → 400 ("file is required"), not 500 — ParseMultipartForm succeeds, then
// FormFile errors.
func TestIntegrationUploadOrderPhotoRequiresFile(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "uploader3@ex.com")

	// Field-only multipart body (no file part) via the existing doMultipart.
	rr := e.doMultipart(t, e.wrapCustomer(e.customerH.UploadOrderPhoto), http.MethodPost,
		"/api/store/orders/upload-photo",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie},
		map[string][]string{"note": {"no file here"}})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("missing file: want 400, got %d (body %q)", rr.Code, rr.Body.String())
	}
	var env errEnvelope
	decode(t, rr, &env)
	if !strings.Contains(env.Error, "file is required") {
		t.Fatalf("missing file: want error 'file is required', got %q", env.Error)
	}
}

// TestIntegrationUploadOrderPhotoCSRFGate: a valid PNG with a mismatched CSRF
// token is rejected by the CSRF gate → 403, before the handler runs.
func TestIntegrationUploadOrderPhotoCSRFGate(t *testing.T) {
	e := newEnv(t)
	sess, _ := e.customerSession(t, "uploader4@ex.com")

	rr := e.doMultipartFile(t, e.wrapCustomer(e.customerH.UploadOrderPhoto), http.MethodPost,
		"/api/store/orders/upload-photo",
		reqOpts{sess: sess, csrfCookieName: cookieauth.StoreCSRFCookie, badCSRF: true},
		"file", "photo.png", pngBytes(t))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("csrf gate: want 403, got %d (body %q)", rr.Code, rr.Body.String())
	}
}

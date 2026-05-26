package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mioru/internal/email"
	"mioru/internal/model"
)

// fakeUserStore is a minimal userStore for handler tests.
type fakeUserStore struct {
	created   *model.User
	createErr error
}

func (f *fakeUserStore) CreateUser(ctx context.Context, u model.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	cp := u
	f.created = &cp
	return nil
}
func (f *fakeUserStore) GetUser(ctx context.Context, username string) (*model.User, error) {
	return nil, nil
}
func (f *fakeUserStore) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, nil
}
func (f *fakeUserStore) UpdateUser(ctx context.Context, username string, updates map[string]string) error {
	return nil
}
func (f *fakeUserStore) UpdatePassword(ctx context.Context, username, hashedPW string) error {
	return nil
}
func (f *fakeUserStore) CreateResetToken(ctx context.Context, username, token string) error {
	return nil
}
func (f *fakeUserStore) ConsumeResetToken(ctx context.Context, token string) (string, error) {
	return "", nil
}

// TestRegisterDoesNotIssueToken guards that admin-created registration returns
// the new account's summary (201) and never a session token — issuing one would
// log the creating admin in as the new user.
func TestRegisterDoesNotIssueToken(t *testing.T) {
	fs := &fakeUserStore{}
	h := NewAuthHandler(fs, email.NewService(), "test-secret-key-at-least-32-chars-long!!", 60)

	body := `{"first_name":"Alice","last_name":"B","email":"alice@example.com","username":"alice","password":"Tr0ubadour-x9"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["access_token"]; ok {
		t.Errorf("response must not contain access_token; got %v", resp)
	}
	if resp["username"] != "alice" {
		t.Errorf("username = %v, want alice", resp["username"])
	}
	if fs.created == nil || fs.created.Username != "alice" {
		t.Errorf("expected user to be created, got %+v", fs.created)
	}
}

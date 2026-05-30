package store

import (
	"testing"

	"mioru/internal/model"
)

func TestCreateCustomerWithOAuth(t *testing.T) {
	s := testStore(t)

	oa := model.CustomerOAuth{
		Provider:    "telegram",
		OAuthID:     "12345",
		ProfileData: `{"first_name":"Pavel","last_name":"Tonkoglas","username":"tonkoglas"}`,
	}

	c := model.Customer{
		FirstName:   "Pavel",
		LastName:    "Tonkoglas",
		AvatarColor: "#44944A",
		// Email and HashedPW deliberately empty — OAuth customer.
	}

	err := s.CreateCustomerWithOAuth(t.Context(), c, oa)
	if err != nil {
		t.Fatalf("CreateCustomerWithOAuth: %v", err)
	}

	// Verify we can look up the customer by OAuth.
	cust, oaFound, err := s.GetCustomerByOAuth(t.Context(), "telegram", "12345")
	if err != nil {
		t.Fatalf("GetCustomerByOAuth: %v", err)
	}
	if cust == nil {
		t.Fatal("expected customer, got nil")
	}
	if oaFound == nil {
		t.Fatal("expected oauth record, got nil")
	}
	if cust.FirstName != "Pavel" {
		t.Errorf("expected FirstName 'Pavel', got %q", cust.FirstName)
	}
	if cust.Email != "" {
		t.Errorf("expected empty email for OAuth customer, got %q", cust.Email)
	}
	if cust.HashedPW != "" {
		t.Errorf("expected empty password for OAuth customer, got %q", cust.HashedPW)
	}
	if oaFound.OAuthID != "12345" {
		t.Errorf("expected oauth_id '12345', got %q", oaFound.OAuthID)
	}
}

func TestCreateCustomerWithOAuth_DuplicateOAuthLink(t *testing.T) {
	s := testStore(t)

	oa := model.CustomerOAuth{
		Provider:    "telegram",
		OAuthID:     "99999",
		ProfileData: `{"first_name":"Dup"}`,
	}

	// First customer with this Telegram ID.
	err := s.CreateCustomerWithOAuth(t.Context(), model.Customer{
		FirstName:   "First",
		AvatarColor: "#111111",
	}, oa)
	if err != nil {
		t.Fatalf("first CreateCustomerWithOAuth: %v", err)
	}

	// Second customer with the same Telegram ID must fail (UNIQUE constraint).
	err = s.CreateCustomerWithOAuth(t.Context(), model.Customer{
		FirstName:   "Second",
		AvatarColor: "#222222",
	}, oa)
	if err == nil {
		t.Fatal("expected error for duplicate oauth link, got nil")
	}
}

func TestGetCustomerByOAuth_NotFound(t *testing.T) {
	s := testStore(t)

	cust, oa, err := s.GetCustomerByOAuth(t.Context(), "telegram", "nonexistent")
	if err != nil {
		t.Fatalf("GetCustomerByOAuth: %v", err)
	}
	if cust != nil {
		t.Error("expected nil customer for nonexistent oauth")
	}
	if oa != nil {
		t.Error("expected nil oauth for nonexistent oauth")
	}
}

func TestLinkOAuth(t *testing.T) {
	s := testStore(t)

	// Create a password-based customer first.
	c := model.Customer{
		Email:       "link-test@example.com",
		HashedPW:    "hashed-pw",
		FirstName:   "Link",
		AvatarColor: "#44944A",
	}
	if err := s.CreateCustomer(t.Context(), c); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}

	cust, err := s.GetCustomerByEmail(t.Context(), "link-test@example.com")
	if err != nil || cust == nil {
		t.Fatalf("GetCustomerByEmail: %v, cust=%v", err, cust)
	}

	// Link Telegram to the existing customer.
	err = s.LinkOAuth(t.Context(), cust.ID, model.CustomerOAuth{
		CustomerID:  cust.ID,
		Provider:    "telegram",
		OAuthID:     "777",
		ProfileData: `{"username":"linked"}`,
	})
	if err != nil {
		t.Fatalf("LinkOAuth: %v", err)
	}

	// Verify the link works.
	found, oa, err := s.GetCustomerByOAuth(t.Context(), "telegram", "777")
	if err != nil {
		t.Fatalf("GetCustomerByOAuth: %v", err)
	}
	if found == nil {
		t.Fatal("expected customer after link, got nil")
	}
	if found.ID != cust.ID {
		t.Errorf("expected customer ID %d, got %d", cust.ID, found.ID)
	}
	if oa.OAuthID != "777" {
		t.Errorf("expected oauth_id '777', got %q", oa.OAuthID)
	}
}

func TestLinkOAuth_Idempotent(t *testing.T) {
	s := testStore(t)

	c := model.Customer{
		Email:       "idempotent@example.com",
		HashedPW:    "pw",
		FirstName:   "Idem",
		AvatarColor: "#333",
	}
	if err := s.CreateCustomer(t.Context(), c); err != nil {
		t.Fatalf("CreateCustomer: %v", err)
	}
	cust, _ := s.GetCustomerByEmail(t.Context(), "idempotent@example.com")

	oa := model.CustomerOAuth{
		CustomerID:  cust.ID,
		Provider:    "telegram",
		OAuthID:     "888",
		ProfileData: `{}`,
	}

	// Link twice — second call must not error.
	if err := s.LinkOAuth(t.Context(), cust.ID, oa); err != nil {
		t.Fatalf("first LinkOAuth: %v", err)
	}
	if err := s.LinkOAuth(t.Context(), cust.ID, oa); err != nil {
		t.Fatalf("second LinkOAuth (idempotent): %v", err)
	}
}

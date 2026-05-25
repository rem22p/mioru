package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret-at-least-32-bytes-long-xxxxx"

func TestHashAndCheckPassword(t *testing.T) {
	const pw = "correct horse battery staple"

	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if hash == pw {
		t.Error("hash equals the plaintext password")
	}
	if !CheckPassword(pw, hash) {
		t.Error("CheckPassword rejected the correct password")
	}
	if CheckPassword("wrong password", hash) {
		t.Error("CheckPassword accepted a wrong password")
	}

	// bcrypt salts each hash, so the same password hashes to different values.
	hash2, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("second HashPassword returned error: %v", err)
	}
	if hash == hash2 {
		t.Error("two hashes of the same password are identical (missing salt)")
	}

	// Cost must be 12 per the security posture.
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost returned error: %v", err)
	}
	if cost != 12 {
		t.Errorf("bcrypt cost = %d, want 12", cost)
	}
}

func TestCheckDummyPassword(t *testing.T) {
	// The timing guard has no return value; it must simply not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("CheckDummyPassword panicked: %v", r)
		}
	}()
	CheckDummyPassword("whatever the client sent")
}

func TestCreateParseTokenRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		sub  string
		typ  string
	}{
		{"admin user", "admin", TokenTypeUser},
		{"store customer", "customer@example.com", TokenTypeCustomer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := CreateToken(tt.sub, tt.typ, testSecret, 60)
			if err != nil {
				t.Fatalf("CreateToken returned error: %v", err)
			}

			sub, err := ParseToken(tok, testSecret, tt.typ)
			if err != nil {
				t.Fatalf("ParseToken returned error: %v", err)
			}
			if sub != tt.sub {
				t.Errorf("sub = %q, want %q", sub, tt.sub)
			}
		})
	}
}

func TestParseTokenWrongSecret(t *testing.T) {
	tok, err := CreateToken("admin", TokenTypeUser, testSecret, 60)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	if _, err := ParseToken(tok, "a-totally-different-secret-key-000000", TokenTypeUser); err == nil {
		t.Error("ParseToken accepted a token signed with a different secret")
	}
}

// TestParseTokenWrongType is the audience-separation guard: a token minted for
// the admin side must not validate against the customer side, and vice versa.
func TestParseTokenWrongType(t *testing.T) {
	userTok, err := CreateToken("admin", TokenTypeUser, testSecret, 60)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	_, err = ParseToken(userTok, testSecret, TokenTypeCustomer)
	if err == nil {
		t.Fatal("a user token validated as a customer token")
	}
	if !strings.Contains(err.Error(), "wrong token type") {
		t.Errorf("error = %q, want it to mention %q", err.Error(), "wrong token type")
	}
}

func TestParseTokenExpired(t *testing.T) {
	// Negative expiry → already past its exp claim.
	tok, err := CreateToken("admin", TokenTypeUser, testSecret, -1)
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	if _, err := ParseToken(tok, testSecret, TokenTypeUser); err == nil {
		t.Error("ParseToken accepted an expired token")
	}
}

func TestParseTokenMalformed(t *testing.T) {
	for _, tok := range []string{"", "not.a.jwt", "garbage"} {
		if _, err := ParseToken(tok, testSecret, TokenTypeUser); err == nil {
			t.Errorf("ParseToken accepted a malformed token %q", tok)
		}
	}
}

// TestParseTokenRejectsNoneAlg defends against the classic JWT bypass: a token
// with "alg": "none" (no signature) must be rejected because ParseToken pins
// the algorithm to HS256 via jwt.WithValidMethods.
func TestParseTokenRejectsNoneAlg(t *testing.T) {
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "admin",
		"typ": TokenTypeUser,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	unsigned, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none-alg token returned error: %v", err)
	}

	if _, err := ParseToken(unsigned, testSecret, TokenTypeUser); err == nil {
		t.Error("ParseToken accepted an alg=none token (algorithm pinning failed)")
	}
}

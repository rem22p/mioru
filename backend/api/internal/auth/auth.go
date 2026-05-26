package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Token types distinguish admin-user tokens from store-customer tokens so a
// token minted for one context cannot be replayed against the other.
const (
	TokenTypeUser     = "user"
	TokenTypeCustomer = "customer"
)

type Config interface {
	Secret() string
	Expiry() int
}

type JWTConfig struct {
	SecretKey string
	ExpiryMin int
}

func (c JWTConfig) Secret() string { return c.SecretKey }
func (c JWTConfig) Expiry() int    { return c.ExpiryMin }

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

func CheckPassword(pw, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// dummyHash is a valid bcrypt hash (cost 12) computed once at init. It is used
// for constant-time comparison when a user/customer is not found so login
// timing does not reveal whether an account exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("constant-time-timing-guard"), 12)

// CheckDummyPassword runs a throwaway bcrypt comparison to equalize login timing
// for the "account not found" path. The result is intentionally ignored.
func CheckDummyPassword(pw string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
}

func CreateToken(sub, typ, secret string, expiryMin int) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"typ": typ,
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(expiryMin) * time.Minute).Unix(),
	})
	return t.SignedString([]byte(secret))
}

// ParseToken validates the token signature, pins the algorithm to HS256, and
// requires the "typ" claim to equal wantType (audience separation between admin
// users and store customers). It returns the subject and the token's issued-at
// (iat) as a Unix timestamp; iat lets the auth middleware reject tokens minted
// before the account's last password change. A token without an iat claim yields
// iat == 0, which is treated as "older than any password change" (rejected).
func ParseToken(tokenStr, secret, wantType string) (sub string, iat int64, err error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", 0, err
	}
	if !t.Valid {
		return "", 0, errors.New("invalid token")
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", 0, errors.New("invalid claims")
	}
	if typ, _ := claims["typ"].(string); typ != wantType {
		return "", 0, errors.New("wrong token type")
	}
	sub, _ = claims["sub"].(string)
	if v, ok := claims["iat"].(float64); ok {
		iat = int64(v)
	}
	return sub, iat, nil
}

package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

func CreateToken(sub, secret string, expiryMin int) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"exp": time.Now().Add(time.Duration(expiryMin) * time.Minute).Unix(),
	})
	return t.SignedString([]byte(secret))
}

func ParseToken(tokenStr, secret string) (string, error) {
	t, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !t.Valid {
		return "", err
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return "", err
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}

// Package token выпускает и валидирует access-JWT и генерирует opaque refresh-токены.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer выпускает/валидирует токены на общем секрете (HS256).
type Issuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// New собирает Issuer.
func New(secret string, accessTTL, refreshTTL time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// RefreshTTL — время жизни refresh-токена/сессии.
func (i *Issuer) RefreshTTL() time.Duration { return i.refreshTTL }

// Access выпускает короткоживущий access-JWT с sub=userID.
func (i *Issuer) Access(userID string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.accessTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

// ParseAccess валидирует access-JWT и возвращает userID (sub).
func (i *Issuer) ParseAccess(tokenStr string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", fmt.Errorf("empty subject")
	}
	return claims.Subject, nil
}

// NewRefresh генерирует случайный opaque refresh-токен (отдаём клиенту) и его hash
// (храним в sessions.refresh_token_hash). Plaintext в БД не попадает.
func NewRefresh() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plaintext = base64.RawURLEncoding.EncodeToString(b)
	return plaintext, HashRefresh(plaintext), nil
}

// HashRefresh — SHA-256 refresh-токена (hex). Детерминированно: для lookup по hash.
func HashRefresh(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

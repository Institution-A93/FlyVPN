// Package credentials генерирует EAP-креды и NT-hash для FreeRADIUS/MSCHAPv2.
package credentials

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strings"
	"unicode/utf16"

	"golang.org/x/crypto/md4" //nolint:staticcheck // MD4 нужен для NT-hash (MSCHAPv2)
)

const (
	usernameLen = 16
	passwordLen = 32
	alphabet    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// NTHash возвращает NT-hash пароля: MD4(UTF-16LE(password)), hex в нижнем регистре.
// Это формат NT-Password для FreeRADIUS (ADR-0014).
func NTHash(password string) string {
	u := utf16.Encode([]rune(password))
	b := make([]byte, len(u)*2)
	for i, r := range u {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	h := md4.New()
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateUsername — случайный alnum-логин (не email, не имя; ADR-0014/README §4).
func GenerateUsername() (string, error) { return randString(usernameLen) }

// GeneratePassword — случайный высокоэнтропийный пароль (≥32 симв).
func GeneratePassword() (string, error) { return randString(passwordLen) }

func randString(n int) (string, error) {
	var sb strings.Builder
	sb.Grow(n)
	max := big.NewInt(int64(len(alphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		sb.WriteByte(alphabet[idx.Int64()])
	}
	return sb.String(), nil
}

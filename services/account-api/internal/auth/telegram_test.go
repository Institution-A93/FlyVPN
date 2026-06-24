package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// validHash вычисляет корректный hash виджета для данных (как это делает Telegram).
func validHash(botToken string, data map[string]string) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k + "=" + data[k])
	}
	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(sb.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyTelegram(t *testing.T) {
	const tok = "123:ABC"
	data := map[string]string{
		"id":        "777",
		"username":  "alice",
		"auth_date": strconv.FormatInt(time.Now().Unix(), 10),
	}
	h := validHash(tok, data)

	if err := VerifyTelegram(tok, data, h); err != nil {
		t.Fatalf("valid hash rejected: %v", err)
	}
	if err := VerifyTelegram(tok, data, h[:len(h)-2]+"00"); err == nil {
		t.Fatal("tampered hash accepted")
	}
	if err := VerifyTelegram("wrong-token", data, h); err == nil {
		t.Fatal("wrong bot token accepted")
	}

	// Протухший auth_date.
	stale := map[string]string{"id": "1", "auth_date": "1000000000"}
	if err := VerifyTelegram(tok, stale, validHash(tok, stale)); err == nil {
		t.Fatal("stale auth_date accepted")
	}
}

func TestOTPHashAndCode(t *testing.T) {
	if HashCode("+70000000000", "0000") == HashCode("+70000000001", "0000") {
		t.Error("hash не солится телефоном")
	}
	if c, _ := GenerateCode(false); c != PrototypeCode {
		t.Errorf("prototype code = %q, want %q", c, PrototypeCode)
	}
	if c, _ := GenerateCode(true); len(c) != 4 {
		t.Errorf("real code len = %d, want 4", len(c))
	}
}

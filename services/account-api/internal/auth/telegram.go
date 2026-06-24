// Package auth — валидация Telegram login-widget и OTP по телефону (Twilio).
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrTelegramAuth — подпись виджета не прошла проверку или данные устарели.
var ErrTelegramAuth = errors.New("telegram auth failed")

// maxAuthAge — насколько старым может быть auth_date виджета.
const maxAuthAge = 24 * time.Hour

// VerifyTelegram проверяет подпись Telegram login-widget (HMAC-SHA256, ключ = SHA256(botToken)).
// data — все поля виджета, КРОМЕ hash. Возвращает ошибку при несовпадении или протухшем auth_date.
func VerifyTelegram(botToken string, data map[string]string, hash string) error {
	// data-check-string: "key=value" по всем полям кроме hash, отсортированные, через \n.
	keys := make([]string, 0, len(data))
	for k := range data {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(data[k])
	}

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(sb.String()))
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(hash)
	if err != nil {
		return ErrTelegramAuth
	}
	if !hmac.Equal(expected, got) {
		return ErrTelegramAuth
	}

	// Свежесть auth_date.
	if ad := data["auth_date"]; ad != "" {
		ts, err := strconv.ParseInt(ad, 10, 64)
		if err != nil {
			return ErrTelegramAuth
		}
		if time.Since(time.Unix(ts, 0)) > maxAuthAge {
			return fmt.Errorf("%w: stale auth_date", ErrTelegramAuth)
		}
	}
	return nil
}

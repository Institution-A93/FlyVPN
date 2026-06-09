// Package plati — проверка подписи вебхука Plati.market.
//
// ВНИМАНИЕ: точная схема канонизации подписи Plati берётся из их docs (формат
// «уникальный товар через API»). Здесь реализован проверочный примитив HMAC-SHA256
// с constant-time сравнением; конкретная строка-для-подписи подставляется на месте
// интеграции (см. README config-api). Тест фиксирует сам примитив.
package plati

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign возвращает hex HMAC-SHA256 от payload с заданным секретом.
func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify сравнивает ожидаемую подпись с переданной (constant-time).
func Verify(secret string, payload []byte, signatureHex string) bool {
	want := Sign(secret, payload)
	return hmac.Equal([]byte(want), []byte(signatureHex))
}

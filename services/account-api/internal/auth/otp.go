package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PrototypeCode — фиксированный код, когда Twilio выключен (decision §5.2 «prototype»).
const PrototypeCode = "0000"

// GenerateCode — 4-значный OTP (или фиксированный prototype-код, если sendEnabled=false).
func GenerateCode(sendEnabled bool) (string, error) {
	if !sendEnabled {
		return PrototypeCode, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}

// HashCode — SHA-256(phone + ":" + code), hex. Соль телефоном исключает межномерный повтор.
func HashCode(phone, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code))
	return hex.EncodeToString(sum[:])
}

// Twilio шлёт SMS через REST API.
type Twilio struct {
	accountSID string
	authToken  string
	from       string
	httpc      *http.Client
}

// NewTwilio собирает клиента.
func NewTwilio(sid, token, from string) *Twilio {
	return &Twilio{accountSID: sid, authToken: token, from: from, httpc: &http.Client{Timeout: 10 * time.Second}}
}

// Send отправляет SMS с кодом.
func (t *Twilio) Send(ctx context.Context, to, code string) error {
	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + t.accountSID + "/Messages.json"
	form := url.Values{}
	form.Set("To", to)
	form.Set("From", t.from)
	form.Set("Body", "FLY VPN: код подтверждения "+code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("twilio status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

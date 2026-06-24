// Package platega — клиент платёжного агрегатора Platega.io (ADR-0020).
package platega

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Способы оплаты (paymentMethod). Без крипты (решение продукта, ADR-0020).
const (
	MethodSBP      = 2  // СБП (QR) — основной для РФ
	MethodCardRub  = 10 // карта РФ (CardsRub)
	MethodCardIntl = 12 // зарубежная карта (InternationalAcquiring)
)

// MethodID отображает наш внутренний код метода в id Platega.
func MethodID(method string) int {
	switch method {
	case "sbp":
		return MethodSBP
	case "card":
		return MethodCardRub
	case "intl_card":
		return MethodCardIntl
	default:
		return MethodSBP
	}
}

// Client — обёртка над HTTP API Platega.
type Client struct {
	baseURL    string
	merchantID string
	secret     string
	httpc      *http.Client
}

// New собирает клиента.
func New(baseURL, merchantID, secret string) *Client {
	return &Client{
		baseURL:    baseURL,
		merchantID: merchantID,
		secret:     secret,
		httpc:      &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateRequest — параметры создания транзакции.
type CreateRequest struct {
	PaymentMethod int
	AmountRub     int
	Description   string
	ReturnURL     string
	FailedURL     string
	Payload       string // = payments.id (привязка к user_id/заказу)
}

type createBody struct {
	PaymentMethod  int `json:"paymentMethod"`
	PaymentDetails struct {
		Amount   int    `json:"amount"`
		Currency string `json:"currency"`
	} `json:"paymentDetails"`
	Description string `json:"description"`
	ReturnURL   string `json:"returnUrl"`
	FailedURL   string `json:"failedUrl"`
	Payload     string `json:"payload"`
}

// CreateResponse — ответ Platega (имена полей — сверить на боевом аккаунте, ADR-0020 «Открытые места»).
type CreateResponse struct {
	TransactionID string `json:"transactionId"`
	PaymentURL    string `json:"paymentUrl"`
	// допускаем альтернативные имена с боевого аккаунта
	ID          string `json:"id"`
	RedirectURL string `json:"redirectUrl"`
}

// Transaction возвращает id и платёжный URL из ответа, учитывая возможные варианты имён.
func (r CreateResponse) Transaction() (id, url string) {
	id = r.TransactionID
	if id == "" {
		id = r.ID
	}
	url = r.PaymentURL
	if url == "" {
		url = r.RedirectURL
	}
	return id, url
}

// CreateTransaction создаёт платёж и возвращает transactionId + URL платёжной страницы.
func (c *Client) CreateTransaction(ctx context.Context, in CreateRequest) (CreateResponse, error) {
	var body createBody
	body.PaymentMethod = in.PaymentMethod
	body.PaymentDetails.Amount = in.AmountRub
	body.PaymentDetails.Currency = "RUB"
	body.Description = in.Description
	body.ReturnURL = in.ReturnURL
	body.FailedURL = in.FailedURL
	body.Payload = in.Payload

	buf, err := json.Marshal(body)
	if err != nil {
		return CreateResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/transaction/process", bytes.NewReader(buf))
	if err != nil {
		return CreateResponse{}, err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return CreateResponse{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 300 {
		return CreateResponse{}, fmt.Errorf("platega create status %d: %s", resp.StatusCode, string(raw))
	}
	var out CreateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return CreateResponse{}, fmt.Errorf("platega create decode: %w", err)
	}
	return out, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("X-MerchantId", c.merchantID)
	req.Header.Set("X-Secret", c.secret)
}

// VerifyWebhookHeaders сверяет X-MerchantId/X-Secret константным сравнением (ADR-0020).
// Отдельная HMAC-подпись тела — открытый вопрос; добавить, когда подтвердится на боевом.
func (c *Client) VerifyWebhookHeaders(merchantID, secret string) bool {
	okM := subtle.ConstantTimeCompare([]byte(merchantID), []byte(c.merchantID)) == 1
	okS := subtle.ConstantTimeCompare([]byte(secret), []byte(c.secret)) == 1
	return okM && okS
}

// Webhook — тело callback'а Platega.
type Webhook struct {
	ID      string `json:"id"`     // transactionId провайдера
	Status  string `json:"status"` // PENDING | CONFIRMED | CANCELED
	Amount  int    `json:"amount"`
	Payload string `json:"payload"` // = payments.id
}

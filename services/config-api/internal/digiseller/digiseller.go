// Package digiseller — клиент Digiseller API (платформа Plati.market): получение токена
// и проверка уникального кода покупки. Спецификация:
//   POST /api/apilogin            {seller_id, timestamp, sign=SHA256(api_key+timestamp)}
//   GET  /api/purchases/unique-code/{code}?token=...
package digiseller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const defaultBase = "https://api.digiseller.com"

// Client потокобезопасен; кэширует токен (валиден 120 мин, обновляем заранее).
type Client struct {
	sellerID int
	apiKey   string
	base     string
	http     *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func New(sellerID int, apiKey string) *Client {
	return &Client{
		sellerID: sellerID,
		apiKey:   apiKey,
		base:     defaultBase,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

// SetBaseURL — для тестов.
func (c *Client) SetBaseURL(u string) { c.base = u }

// Sign — подпись запроса apilogin: SHA256(api_key + timestamp), hex.
func Sign(apiKey string, ts int64) string {
	sum := sha256.Sum256([]byte(apiKey + strconv.FormatInt(ts, 10)))
	return hex.EncodeToString(sum[:])
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	ts := time.Now().Unix()
	body, _ := json.Marshal(map[string]any{
		"seller_id": c.sellerID,
		"timestamp": ts,
		"sign":      Sign(c.apiKey, ts),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/apilogin", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("apilogin: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		Retval    int    `json:"retval"`
		Desc      string `json:"desc"`
		Token     string `json:"token"`
		ValidThru string `json:"valid_thru"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("apilogin: bad json: %w", err)
	}
	if r.Token == "" {
		return "", fmt.Errorf("apilogin: retval=%d %s", r.Retval, r.Desc)
	}
	c.token = r.Token
	c.tokenExp = time.Now().Add(110 * time.Minute) // запас до 120
	return c.token, nil
}

// Purchase — нужные поля ответа проверки уникального кода.
type Purchase struct {
	IDGoods int     `json:"id_goods"`
	Amount  float64 `json:"amount"`
	Email   string  `json:"email"`
	DatePay string  `json:"date_pay"`
	State   int     // unique_code_state.state
}

// CheckUniqueCode валидирует уникальный код и возвращает данные покупки.
func (c *Client) CheckUniqueCode(ctx context.Context, code string) (Purchase, error) {
	tok, err := c.getToken(ctx)
	if err != nil {
		return Purchase{}, err
	}
	u := fmt.Sprintf("%s/api/purchases/unique-code/%s?token=%s",
		c.base, url.PathEscape(code), url.QueryEscape(tok))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Purchase{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Purchase{}, fmt.Errorf("unique-code: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		Retval          int     `json:"retval"`
		RetDesc         string  `json:"retdesc"`
		IDGoods         int     `json:"id_goods"`
		Amount          float64 `json:"amount"`
		Email           string  `json:"email"`
		DatePay         string  `json:"date_pay"`
		UniqueCodeState struct {
			State int `json:"state"`
		} `json:"unique_code_state"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return Purchase{}, fmt.Errorf("unique-code: bad json: %w", err)
	}
	if r.IDGoods == 0 {
		return Purchase{}, fmt.Errorf("unique-code: невалидный код (retval=%d %s)", r.Retval, r.RetDesc)
	}
	return Purchase{
		IDGoods: r.IDGoods,
		Amount:  r.Amount,
		Email:   r.Email,
		DatePay: r.DatePay,
		State:   r.UniqueCodeState.State,
	}, nil
}

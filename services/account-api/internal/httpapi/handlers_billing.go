package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
	"github.com/institution-a93/flyvpn/services/account-api/internal/platega"
	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
)

// Package — позиция каталога MVP (decision #4: один пакет, month-only).
type pkg struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	PriceRub     int    `json:"priceRub"`
	TrafficBytes int64  `json:"trafficBytes"`
	DurationDays int    `json:"durationDays"`
}

// mvpPackages — единственный пакет 300 ₽ / 10 ГБ / 1 мес.
var mvpPackages = []pkg{{
	ID:           "vpn-10gb-month",
	Title:        "FLY VPN — 10 ГБ / 1 мес",
	PriceRub:     300,
	TrafficBytes: config.PaidLimitBytes,
	DurationDays: 30,
}}

func packageByID(id string) (pkg, bool) {
	for _, p := range mvpPackages {
		if p.ID == id {
			return p, true
		}
	}
	return pkg{}, false
}

// GET /packages → каталог.
func (s *Server) packages(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"packages": mvpPackages})
}

// POST /packages/{id}/purchase → создаёт платёж в Platega, возвращает {paymentUrl}.
func (s *Server) purchase(w http.ResponseWriter, r *http.Request) {
	if s.platega == nil {
		s.writeErr(w, http.StatusServiceUnavailable, "unavailable", "payments not configured")
		return
	}
	uid := userID(r)
	p, ok := packageByID(r.PathValue("id"))
	if !ok {
		s.writeErr(w, http.StatusNotFound, "not_found", "unknown package")
		return
	}

	var body struct {
		Method string `json:"method"` // sbp | card | intl_card
		Email  string `json:"email"`  // billing-email для чеков (decision #10)
	}
	_ = decodeJSON(r, &body)
	method := body.Method
	if method == "" {
		method = "sbp"
	}
	if body.Email != "" {
		if err := s.store.SetEmail(r.Context(), uid, strings.TrimSpace(body.Email)); err != nil {
			s.fail(w, "set email", err)
			return
		}
	}

	// Идемпотентность: ключ от клиента или сгенерированный.
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		idem = uuid.NewString()
	}

	pay, isNew, err := s.store.CreatePayment(r.Context(), uid, p.ID, p.PriceRub, method, idem)
	if err != nil {
		s.fail(w, "create payment", err)
		return
	}
	// Повтор по тому же idempotency_key для уже-оплаченного — не создаём новую транзакцию.
	if !isNew && pay.Status == "succeeded" {
		s.writeErr(w, http.StatusConflict, "already_paid", "payment already completed")
		return
	}

	resp, err := s.platega.CreateTransaction(r.Context(), platega.CreateRequest{
		PaymentMethod: platega.MethodID(method),
		AmountRub:     p.PriceRub,
		Description:   p.Title,
		ReturnURL:     s.cfg.PublicBaseURL + "/cabinet?pay=ok",
		FailedURL:     s.cfg.PublicBaseURL + "/cabinet?pay=fail",
		Payload:       pay.ID, // привязка к user_id/заказу
	})
	if err != nil {
		s.fail(w, "platega create", err)
		return
	}
	txID, payURL := resp.Transaction()
	if txID != "" {
		if err := s.store.AttachProviderID(r.Context(), pay.ID, txID); err != nil {
			s.fail(w, "attach provider id", err)
			return
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"paymentUrl": payURL})
}

// POST /webhooks/payments/platega — источник истины по правам (ADR-0020).
func (s *Server) plategaWebhook(w http.ResponseWriter, r *http.Request) {
	if s.platega == nil {
		http.Error(w, "payments not configured", http.StatusServiceUnavailable)
		return
	}
	// Аутентификация callback'а — по заголовкам Platega (константное сравнение).
	if !s.platega.VerifyWebhookHeaders(r.Header.Get("X-MerchantId"), r.Header.Get("X-Secret")) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	raw := readBody(r)
	var wh platega.Webhook
	if err := json.Unmarshal(raw, &wh); err != nil || wh.Payload == "" {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}

	switch strings.ToUpper(wh.Status) {
	case "CONFIRMED":
		if err := s.store.ConfirmPayment(r.Context(), wh.Payload, wh.ID, raw); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, "unknown payment", http.StatusNotFound)
				return
			}
			s.log.Error("webhook confirm", "err", err, "payload", wh.Payload)
			http.Error(w, "internal", http.StatusInternalServerError)
			return
		}
	case "CANCELED":
		if err := s.store.MarkPayment(r.Context(), wh.Payload, "failed", raw); err != nil && !errors.Is(err, store.ErrNotFound) {
			s.log.Error("webhook cancel", "err", err, "payload", wh.Payload)
			http.Error(w, "internal", http.StatusInternalServerError)
			return
		}
	case "PENDING":
		// промежуточный статус — подтверждаем приём, прав не меняем.
	default:
		s.log.Warn("webhook unknown status", "status", wh.Status, "payload", wh.Payload)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

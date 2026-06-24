// Package httpapi — HTTP-слой account-api (/api/v1 + вебхук Platega + healthz).
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
	"github.com/institution-a93/flyvpn/services/account-api/internal/platega"
	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
	"github.com/institution-a93/flyvpn/services/account-api/internal/token"
)

// Twilio — то, что нужно хендлерам для отправки SMS (для тестируемости/выключения).
type Twilio interface {
	Send(ctx context.Context, to, code string) error
}

// Server держит зависимости HTTP-слоя.
type Server struct {
	cfg      config.Config
	store    *store.Store
	tokens   *token.Issuer
	platega  *platega.Client // nil, если Platega не сконфигурирован
	twilio   Twilio          // nil, если Twilio выключен (prototype-код)
	template string
	log      *slog.Logger
}

// New собирает сервер. platega/twilio могут быть nil.
func New(cfg config.Config, st *store.Store, tk *token.Issuer, pg *platega.Client, tw Twilio, tmpl string, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: st, tokens: tk, platega: pg, twilio: tw, template: tmpl, log: log}
}

// Routes возвращает маршрутизатор со всеми эндпоинтами.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)

	// Публичные (без access-токена).
	mux.HandleFunc("POST /api/v1/auth/otp/request", s.otpRequest)
	mux.HandleFunc("POST /api/v1/auth/otp/verify", s.otpVerify)
	mux.HandleFunc("POST /api/v1/auth/telegram", s.telegramLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", s.refresh)

	// Провайдерский вебхук (своя аутентификация по заголовкам Platega).
	mux.HandleFunc("POST /webhooks/payments/platega", s.plategaWebhook)

	// Защищённые (Bearer access-токен).
	mux.HandleFunc("POST /api/v1/auth/logout", s.auth(s.logout))
	mux.HandleFunc("GET /api/v1/me", s.auth(s.me))
	mux.HandleFunc("DELETE /api/v1/me", s.auth(s.deleteMe))
	mux.HandleFunc("GET /api/v1/subscription", s.auth(s.subscription))
	mux.HandleFunc("GET /api/v1/traffic", s.auth(s.traffic))
	mux.HandleFunc("GET /api/v1/packages", s.auth(s.packages))
	mux.HandleFunc("POST /api/v1/packages/{id}/purchase", s.auth(s.purchase))
	mux.HandleFunc("GET /api/v1/referral", s.auth(s.referral))
	mux.HandleFunc("POST /api/v1/devices", s.auth(s.issueDevice))
	mux.HandleFunc("GET /api/v1/devices", s.auth(s.listDevices))
	mux.HandleFunc("DELETE /api/v1/devices/{id}", s.auth(s.revokeDevice))

	return mux
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
		return
	}
	_, _ = w.Write([]byte("ok\n"))
}

// --- auth middleware ---

type ctxKey string

const userIDKey ctxKey = "userID"

// auth оборачивает хендлер проверкой Bearer access-токена; кладёт userID в контекст.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			s.writeErr(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		userID, err := s.tokens.ParseAccess(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			s.writeErr(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

func userID(r *http.Request) string {
	v, _ := r.Context().Value(userIDKey).(string)
	return v
}

// --- helpers ---

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// errorEnvelope — формат ошибок фронта: {"error":{"code","message"}}.
type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Server) writeErr(w http.ResponseWriter, status int, code, msg string) {
	var e errorEnvelope
	e.Error.Code = code
	e.Error.Message = msg
	s.writeJSON(w, status, e)
}

// fail логирует внутреннюю ошибку и отдаёт 500 без деталей.
func (s *Server) fail(w http.ResponseWriter, step string, err error) {
	s.log.Error("account-api", "step", step, "err", err)
	s.writeErr(w, http.StatusInternalServerError, "internal", "internal error")
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	return dec.Decode(dst)
}

// clientIP — best-effort IP клиента (для rate-limit/сессий).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i > 0 {
		return host[:i]
	}
	return host
}

// Package httpapi — HTTP-слой config-api (healthz + выдача по уникальному коду Plati/Digiseller).
package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/institution-a93/flyvpn/services/config-api/internal/config"
	"github.com/institution-a93/flyvpn/services/config-api/internal/credentials"
	"github.com/institution-a93/flyvpn/services/config-api/internal/digiseller"
	"github.com/institution-a93/flyvpn/services/config-api/internal/mobileconfig"
	"github.com/institution-a93/flyvpn/services/config-api/internal/store"
)

// Issuer — то, что нужно хендлеру от стораджа (для тестируемости).
type Issuer interface {
	Issue(ctx context.Context, p store.IssueParams) (store.IssueResult, error)
}

// CodeChecker — проверка уникального кода в Digiseller (для тестируемости).
type CodeChecker interface {
	CheckUniqueCode(ctx context.Context, code string) (digiseller.Purchase, error)
}

// Server держит зависимости HTTP-слоя.
type Server struct {
	cfg      config.Config
	store    Issuer
	checker  CodeChecker // nil, если Digiseller не сконфигурирован
	template string
	log      *slog.Logger
}

// New собирает сервер. checker может быть nil (тогда /plati/issue отдаёт 503).
func New(cfg config.Config, st Issuer, checker CodeChecker, tmpl string, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: st, checker: checker, template: tmpl, log: log}
}

// Routes возвращает маршрутизатор.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	// Digiseller редиректит покупателя сюда с ?uniquecode=... (GET).
	mux.HandleFunc("GET /plati/issue", s.platiIssue)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

var planDuration = map[string]time.Duration{
	"30d":  30 * 24 * time.Hour,
	"90d":  90 * 24 * time.Hour,
	"365d": 365 * 24 * time.Hour,
}

// platiIssue: валидирует уникальный код Digiseller, создаёт/продлевает подписку и
// возвращает .mobileconfig покупателю.
func (s *Server) platiIssue(w http.ResponseWriter, r *http.Request) {
	if s.checker == nil {
		http.Error(w, "billing not configured", http.StatusServiceUnavailable)
		return
	}
	code := r.URL.Query().Get("uniquecode")
	if code == "" {
		http.Error(w, "uniquecode required", http.StatusBadRequest)
		return
	}

	// Проверка кода в Digiseller ДО любых изменений в БД.
	purchase, err := s.checker.CheckUniqueCode(r.Context(), code)
	if err != nil {
		s.log.Warn("digiseller check", "err", err)
		http.Error(w, "invalid or unpaid code", http.StatusForbidden)
		return
	}

	plan := s.cfg.PlanFor(purchase.IDGoods)
	dur, ok := planDuration[plan]
	if !ok {
		s.fail(w, "plan mapping", nil)
		return
	}

	username, err := credentials.GenerateUsername()
	if err != nil {
		s.fail(w, "gen username", err)
		return
	}
	password, err := credentials.GeneratePassword()
	if err != nil {
		s.fail(w, "gen password", err)
		return
	}

	res, err := s.store.Issue(r.Context(), store.IssueParams{
		PlatiBuyerID:  purchase.Email, // стабильный идентификатор покупателя (Digiseller)
		Email:         purchase.Email,
		PlatiOrderID:  code, // уникальный код = ключ идемпотентности
		Plan:          plan,
		AmountRub:     int(purchase.Amount),
		Duration:      dur,
		CandidateUser: username,
		NTHash:        credentials.NTHash(password),
	})
	if err != nil {
		s.fail(w, "issue", err)
		return
	}

	profile, err := mobileconfig.Render(s.template, mobileconfig.Fields{
		ProfileIdentifier: "com.smartinternet.vpn." + uuid.NewString(),
		ProfileUUID:       uuid.NewString(),
		PayloadUUID:       uuid.NewString(),
		DisplayName:       s.cfg.DisplayName,
		OrgName:           s.cfg.Organization,
		VPNRemoteAddress:  s.cfg.VPNRemote,
		VPNRemoteID:       s.cfg.VPNRemoteID,
		ServerCACN:        s.cfg.ServerCACN,
		EAPUsername:       res.Username,
		EAPPassword:       password,
	})
	if err != nil {
		s.fail(w, "render profile", err)
		return
	}

	w.Header().Set("Content-Type", "application/x-apple-aspen-config")
	w.Header().Set("Content-Disposition", `attachment; filename="smartinternet.mobileconfig"`)
	_, _ = w.Write(profile)
}

func (s *Server) fail(w http.ResponseWriter, msg string, err error) {
	s.log.Error("plati issue", "step", msg, "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

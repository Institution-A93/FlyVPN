// Package httpapi — HTTP-слой config-api (healthz + вебхук Plati).
package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/institution-a93/flyvpn/services/config-api/internal/config"
	"github.com/institution-a93/flyvpn/services/config-api/internal/credentials"
	"github.com/institution-a93/flyvpn/services/config-api/internal/mobileconfig"
	"github.com/institution-a93/flyvpn/services/config-api/internal/plati"
	"github.com/institution-a93/flyvpn/services/config-api/internal/store"
)

// Issuer — то, что нужно хендлеру от стораджа (для тестируемости).
type Issuer interface {
	Issue(ctx context.Context, p store.IssueParams) (store.IssueResult, error)
}

// Server держит зависимости HTTP-слоя.
type Server struct {
	cfg      config.Config
	store    Issuer
	template string
	log      *slog.Logger
}

// New собирает сервер.
func New(cfg config.Config, st Issuer, tmpl string, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: st, template: tmpl, log: log}
}

// Routes возвращает маршрутизатор.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("POST /plati/issue", s.platiIssue)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

// platiRequest — упрощённое тело вебхука. Точная схема Plati — из их docs.
type platiRequest struct {
	BuyerID   string `json:"buyer_id"`
	Email     string `json:"email"`
	OrderID   string `json:"order_id"`
	Plan      string `json:"plan"`       // '30d' | '90d' | '365d'
	AmountRub int    `json:"amount_rub"`
}

var planDuration = map[string]time.Duration{
	"30d":  30 * 24 * time.Hour,
	"90d":  90 * 24 * time.Hour,
	"365d": 365 * 24 * time.Hour,
}

func (s *Server) platiIssue(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	// Проверка HMAC до любых изменений в БД. Подпись — в заголовке X-Plati-Signature.
	// ВНИМАНИЕ: канонизация payload под конкретный формат Plati — на месте интеграции.
	sig := r.Header.Get("X-Plati-Signature")
	if !plati.Verify(s.cfg.PlatiSecret, body, sig) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var req platiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	dur, ok := planDuration[req.Plan]
	if !ok || req.BuyerID == "" || req.OrderID == "" {
		http.Error(w, "bad params", http.StatusBadRequest)
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
		PlatiBuyerID:  req.BuyerID,
		Email:         req.Email,
		PlatiOrderID:  req.OrderID,
		Plan:          req.Plan,
		AmountRub:     req.AmountRub,
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

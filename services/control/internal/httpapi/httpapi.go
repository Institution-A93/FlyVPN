// Package httpapi — внутренний HTTP-слой control: операторский API (CRUD кредов),
// реестр узлов и платёжные webhook'и. Всё за приватной сетью (ADR-0021); операторские
// и webhook-эндпоинты — за bearer-токенами. Панель (HTML) добавляется поверх этих ручек.
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/institution-a93/flyvpn/services/control/internal/config"
	"github.com/institution-a93/flyvpn/services/control/internal/contract"
	"github.com/institution-a93/flyvpn/services/control/internal/fleet"
	"github.com/institution-a93/flyvpn/services/control/internal/profiles"
)

// Server держит зависимости.
type Server struct {
	cfg  config.Config
	ct   *contract.Contract
	fl   *fleet.Fleet
	dis  fleet.Disconnector
	log  *slog.Logger
}

func New(cfg config.Config, ct *contract.Contract, fl *fleet.Fleet, dis fleet.Disconnector, log *slog.Logger) *Server {
	return &Server{cfg: cfg, ct: ct, fl: fl, dis: dis, log: log}
}

// Register вешает ручки control на общий mux (рядом с панелью).
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", s.healthz)

	// Узлы: self-register/heartbeat + список (оператор).
	mux.HandleFunc("POST /fleet/nodes", s.op(s.registerNode))
	mux.HandleFunc("GET /fleet/nodes", s.op(s.listNodes))

	// Операторский контракт (единственный писатель — contract).
	mux.HandleFunc("GET /api/v1/users/{id}", s.op(s.getUser))
	mux.HandleFunc("POST /api/v1/users/{id}/provision", s.op(s.provision))
	mux.HandleFunc("POST /api/v1/users/{id}/renew", s.op(s.renew))
	mux.HandleFunc("POST /api/v1/users/{id}/revoke", s.op(s.revoke))

	// Платёжные коннекторы (вне MVP-учёта): подтверждённый платёж → renew.
	mux.HandleFunc("POST /webhooks/pay", s.webhookRenew)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok\n"))
}

// op — bearer-мидлвар оператора (constant-time сравнение).
func (s *Server) op(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !bearerOK(r, s.cfg.OperatorToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func bearerOK(r *http.Request, want string) bool {
	const p = "Bearer "
	h := r.Header.Get("Authorization")
	if want == "" || len(h) <= len(p) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(h[len(p):]), []byte(want)) == 1
}

// --- handlers ---

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	cr, err := s.ct.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, contract.ErrNotFound) {
		http.Error(w, "not found", 404)
		return
	}
	if err != nil {
		s.fail(w, "get", err)
		return
	}
	writeJSON(w, 200, cr)
}

type planReq struct {
	Plan string `json:"plan"`
}

func (s *Server) planOf(code string) (contract.Plan, bool) {
	p, ok := s.cfg.Plans[code]
	return p, ok
}

func (s *Server) provision(w http.ResponseWriter, r *http.Request) {
	var b planReq
	_ = json.NewDecoder(r.Body).Decode(&b)
	plan, ok := s.planOf(b.Plan)
	if !ok {
		http.Error(w, "unknown plan", 400)
		return
	}
	pr, err := s.ct.Provision(r.Context(), r.PathValue("id"), plan)
	if err != nil {
		s.fail(w, "provision", err)
		return
	}
	// Профили (с паролем — отдаются ОДИН раз; в БД только nt_hash).
	pp := profiles.Params{
		DisplayName: s.cfg.DisplayName, OrgName: s.cfg.OrgName,
		ServerAddr: s.cfg.ServerAddr, ServerID: s.cfg.ServerID,
		Username: pr.Username, Password: pr.Password,
		ProfileIdentifier: "pro.flynet.vpn." + uuid.NewString(),
		ProfileUUID:       uuid.NewString(),
		PayloadUUID:       uuid.NewString(),
	}
	mc, err := profiles.Mobileconfig(pp)
	if err != nil {
		s.fail(w, "mobileconfig", err)
		return
	}
	ss, err := profiles.Sswan(pp)
	if err != nil {
		s.fail(w, "sswan", err)
		return
	}
	writeJSON(w, 201, map[string]any{
		"username": pr.Username, "password": pr.Password, "framed_ip": pr.FramedIP,
		"mobileconfig": string(mc), "sswan": string(ss),
	})
}

func (s *Server) renew(w http.ResponseWriter, r *http.Request) {
	var b planReq
	_ = json.NewDecoder(r.Body).Decode(&b)
	plan, ok := s.planOf(b.Plan)
	if !ok {
		http.Error(w, "unknown plan", 400)
		return
	}
	if err := s.ct.Renew(r.Context(), r.PathValue("id"), plan); err != nil {
		if errors.Is(err, contract.ErrNotFound) {
			http.Error(w, "not found", 404)
			return
		}
		s.fail(w, "renew", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) revoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.ct.Revoke(r.Context(), id); err != nil && !errors.Is(err, contract.ErrNotFound) {
		s.fail(w, "revoke", err)
		return
	}
	// Анти-локаут наоборот: revoke = жёсткий отказ → рвём живые сессии немедленно.
	if s.dis != nil {
		if err := s.fl.DisconnectUser(r.Context(), id, s.dis); err != nil {
			s.log.Warn("revoke disconnect", "user", id, "err", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type webhookReq struct {
	Secret string `json:"secret"`
	UserID string `json:"user_id"`
	Plan   string `json:"plan"`
}

// webhookRenew — точка для платёжных коннекторов: подтверждённый платёж → renew.
// Аутентификация по общему секрету (коннектор проверяет подпись провайдера у себя).
func (s *Server) webhookRenew(w http.ResponseWriter, r *http.Request) {
	var b webhookReq
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	if s.cfg.WebhookSecret == "" || subtle.ConstantTimeCompare([]byte(b.Secret), []byte(s.cfg.WebhookSecret)) != 1 {
		http.Error(w, "unauthorized", 401)
		return
	}
	plan, ok := s.planOf(b.Plan)
	if !ok {
		http.Error(w, "unknown plan", 400)
		return
	}
	if err := s.ct.Renew(r.Context(), b.UserID, plan); err != nil {
		if errors.Is(err, contract.ErrNotFound) {
			http.Error(w, "not found", 404)
			return
		}
		s.fail(w, "webhook renew", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.fl.List(r.Context())
	if err != nil {
		s.fail(w, "nodes", err)
		return
	}
	writeJSON(w, 200, nodes)
}

type registerReq struct {
	Role, Region, PublicIP string
}

func (s *Server) registerNode(w http.ResponseWriter, r *http.Request) {
	var b registerReq
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.PublicIP == "" {
		http.Error(w, "bad request", 400)
		return
	}
	id, err := s.fl.Register(r.Context(), b.Role, b.Region, b.PublicIP)
	if err != nil {
		s.fail(w, "register", err)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) fail(w http.ResponseWriter, step string, err error) {
	s.log.Error("control", "step", step, "err", err)
	http.Error(w, "internal error", 500)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Package httpapi — admin/HTTP API оркестратора (реестр узлов, healthz).
package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/institution-a93/flyvpn/services/orchestrator/internal/store"
)

// Registry — что нужно API от стораджа.
type Registry interface {
	Register(ctx context.Context, role, region, publicIP string) (string, error)
	List(ctx context.Context) ([]store.Node, error)
}

type Server struct {
	reg Registry
	log *slog.Logger
}

func New(reg Registry, log *slog.Logger) *Server { return &Server{reg: reg, log: log} }

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("GET /nodes", s.listNodes)
	mux.HandleFunc("POST /nodes", s.registerNode)
	return mux
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.reg.List(r.Context())
	if err != nil {
		s.fail(w, "list", err)
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

type registerReq struct {
	Role     string `json:"role"`
	Region   string `json:"region"`
	PublicIP string `json:"public_ip"`
}

func (s *Server) registerNode(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.Role == "" || req.PublicIP == "" {
		http.Error(w, "role и public_ip обязательны", http.StatusBadRequest)
		return
	}
	id, err := s.reg.Register(r.Context(), req.Role, req.Region, req.PublicIP)
	if err != nil {
		s.fail(w, "register", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *Server) fail(w http.ResponseWriter, step string, err error) {
	s.log.Error("orchestrator api", "step", step, "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

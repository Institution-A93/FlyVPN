// Command orchestrator — реестр узлов, health-checking, (phase 2: ротация, GeoDNS).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/institution-a93/flyvpn/services/orchestrator/internal/config"
	"github.com/institution-a93/flyvpn/services/orchestrator/internal/health"
	"github.com/institution-a93/flyvpn/services/orchestrator/internal/httpapi"
	"github.com/institution-a93/flyvpn/services/orchestrator/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.FromEnv()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// Активные пробы: egress и control — TLS на 443. ingress (UDP/IKEv2) — без активной
	// пробы на MMVP (статус ведётся heartbeat'ом; глубокая IKE-проба — TODO).
	probes := map[string]health.Probe{
		"egress":  health.TLSProbe(443),
		"control": health.TLSProbe(443),
	}
	checker := health.NewChecker(st, probes, cfg.HealthThreshold)
	go checker.Run(ctx, cfg.HealthInterval)

	srv := httpapi.New(st, log)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(sctx)
	}()

	log.Info("orchestrator listening", "addr", cfg.ListenAddr, "health_interval", cfg.HealthInterval.String())
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}

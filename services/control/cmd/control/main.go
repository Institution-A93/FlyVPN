// Command control — единый бинарь control plane (ADR-0021): операторская панель/API
// (contract — единственный писатель auth_credentials), профили (.mobileconfig/.sswan),
// реестр узлов + health + CoA, платёжные webhook'и. Один pgx-пул, один HTTP-сервер,
// одна фоновая горутина health.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/institution-a93/flyvpn/services/control/internal/config"
	"github.com/institution-a93/flyvpn/services/control/internal/contract"
	"github.com/institution-a93/flyvpn/services/control/internal/fleet"
	"github.com/institution-a93/flyvpn/services/control/internal/httpapi"
	"github.com/institution-a93/flyvpn/services/control/internal/panel"
	"github.com/institution-a93/flyvpn/services/control/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg, err := config.FromEnv()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := store.New(ctx, cfg.DSN)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	ct := contract.New(pool)
	fl := fleet.New(pool)
	dis := fleet.RadclientDisconnector{Port: cfg.DAEPort, Secret: cfg.DAESecret}

	mux := http.NewServeMux()
	httpapi.New(cfg, ct, fl, dis, log).Register(mux)
	panel.New(cfg.OperatorToken, cfg.Plans, ct, fl, log).Register(mux)

	// Фоновая health-проверка узлов (egress: TLS Reality-порт; ingress: IKE-порт).
	go func() {
		probes := map[string]fleet.Probe{
			"egress":  fleet.TLSProbe(443),
			"ingress": fleet.TCPProbe(500),
		}
		fails := map[string]int{}
		t := time.NewTicker(cfg.HealthEvery)
		defer t.Stop()
		for {
			if err := fl.CheckOnce(ctx, probes, 3, fails); err != nil {
				log.Warn("health", "err", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()

	httpServer := &http.Server{Addr: cfg.Listen, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(sh)
	}()

	log.Info("control up", "listen", cfg.Listen)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("http", "err", err)
		os.Exit(1)
	}
	log.Info("control stopped")
}

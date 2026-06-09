// Command config-api — Plati-вебхук, генерация .mobileconfig, выдача EAP-кредов.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	configapi "github.com/institution-a93/flyvpn/services/config-api"
	"github.com/institution-a93/flyvpn/services/config-api/internal/config"
	"github.com/institution-a93/flyvpn/services/config-api/internal/httpapi"
	"github.com/institution-a93/flyvpn/services/config-api/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.FromEnv()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	srv := httpapi.New(cfg, st, configapi.ProfileTemplate, log)
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info("config-api listening", "addr", cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}

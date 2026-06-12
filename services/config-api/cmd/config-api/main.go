// Command config-api — выдача по коду Plati/Digiseller, генерация .mobileconfig, выдача EAP-кредов.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	configapi "github.com/institution-a93/flyvpn/services/config-api"
	"github.com/institution-a93/flyvpn/services/config-api/internal/config"
	"github.com/institution-a93/flyvpn/services/config-api/internal/digiseller"
	"github.com/institution-a93/flyvpn/services/config-api/internal/httpapi"
	"github.com/institution-a93/flyvpn/services/config-api/internal/store"
	"golang.org/x/crypto/acme/autocert"
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

	// Digiseller (Plati) — опционально: если креды не заданы, /plati/issue отдаёт 503.
	var checker httpapi.CodeChecker
	if cfg.DigisellerEnabled() {
		checker = digiseller.New(cfg.DigisellerSellerID, cfg.DigisellerAPIKey)
		log.Info("digiseller enabled", "seller_id", cfg.DigisellerSellerID)
	} else {
		log.Warn("digiseller not configured — /plati/issue вернёт 503")
	}

	handler := httpapi.New(cfg, st, checker, configapi.ProfileTemplate, log).Routes()

	if cfg.TLSEnabled() {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(cfg.ACMEDomain),
			Cache:      autocert.DirCache(cfg.ACMECacheDir),
		}
		// :80 — HTTP-01 challenge ACME (+ редирект на HTTPS).
		go func() {
			if err := http.ListenAndServe(":80", m.HTTPHandler(nil)); err != nil {
				log.Error("acme http", "err", err)
			}
		}()
		httpsServer := &http.Server{
			Addr:              ":443",
			Handler:           handler,
			TLSConfig:         m.TLSConfig(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		log.Info("config-api listening (TLS/ACME)", "domain", cfg.ACMEDomain)
		if err := httpsServer.ListenAndServeTLS("", ""); err != nil {
			log.Error("serve tls", "err", err)
			os.Exit(1)
		}
		return
	}

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info("config-api listening (plain HTTP)", "addr", cfg.ListenAddr)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}

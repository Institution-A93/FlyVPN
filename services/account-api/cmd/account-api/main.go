// Command account-api — аккаунты, сессии, подписки/квота, устройства и платежи Platega
// для веб-кабинета FLY VPN (docs/backend-requirements.md, ADR-0020).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	accountapi "github.com/institution-a93/flyvpn/services/account-api"
	"github.com/institution-a93/flyvpn/services/account-api/internal/auth"
	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
	"github.com/institution-a93/flyvpn/services/account-api/internal/httpapi"
	"github.com/institution-a93/flyvpn/services/account-api/internal/platega"
	"github.com/institution-a93/flyvpn/services/account-api/internal/store"
	"github.com/institution-a93/flyvpn/services/account-api/internal/token"
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

	tk := token.New(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)

	// Platega — опционально: без кредов purchase/webhook отдают 503.
	var pg *platega.Client
	if cfg.PlategaEnabled() {
		pg = platega.New(cfg.PlategaBaseURL, cfg.PlategaMerchantID, cfg.PlategaSecret)
		log.Info("platega enabled", "base", cfg.PlategaBaseURL)
	} else {
		log.Warn("platega not configured — purchase/webhook вернут 503")
	}

	// Twilio — опционально: без него OTP-код фиксирован 0000 (prototype).
	var tw httpapi.Twilio
	if cfg.TwilioEnabled {
		tw = auth.NewTwilio(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber)
		log.Info("twilio enabled")
	} else {
		log.Warn("twilio not configured — OTP в prototype-режиме (код 0000)")
	}

	handler := httpapi.New(cfg, st, tk, pg, tw, accountapi.ProfileTemplate, log).Routes()

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info("account-api listening", "addr", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("serve", "err", err)
		os.Exit(1)
	}
}

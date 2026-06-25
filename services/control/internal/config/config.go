// Package config — конфигурация бинаря control из окружения.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/institution-a93/flyvpn/services/control/internal/contract"
)

// Config — всё, что нужно control.
type Config struct {
	DSN           string // CONTROL_DSN
	Listen        string // CONTROL_LISTEN (внутренний адрес)
	OperatorToken string // CONTROL_OPERATOR_TOKEN — bearer для панели/API оператора
	WebhookSecret string // CONTROL_WEBHOOK_SECRET — для платёжных коннекторов
	DAEPort       int    // CONTROL_DAE_PORT (по умолчанию 3799)
	DAESecret     string // CONTROL_DAE_SECRET — общий секрет с узлами
	HealthEvery   time.Duration

	// Дефолты профиля (ADR-0022): адрес/идентификатор узла, имена.
	DisplayName string // CONTROL_DISPLAY_NAME
	OrgName     string // CONTROL_ORG_NAME
	ServerAddr  string // CONTROL_SERVER_ADDR (домен ingress)
	ServerID    string // CONTROL_SERVER_ID (CN серта)

	// Тарифы: код → план.
	Plans map[string]contract.Plan
}

// FromEnv читает конфиг. DSN обязателен.
func FromEnv() (Config, error) {
	c := Config{
		DSN:           os.Getenv("CONTROL_DSN"),
		Listen:        envOr("CONTROL_LISTEN", "127.0.0.1:8080"),
		OperatorToken: os.Getenv("CONTROL_OPERATOR_TOKEN"),
		WebhookSecret: os.Getenv("CONTROL_WEBHOOK_SECRET"),
		DAESecret:     os.Getenv("CONTROL_DAE_SECRET"),
		DAEPort:       envInt("CONTROL_DAE_PORT", 3799),
		HealthEvery:   time.Duration(envInt("CONTROL_HEALTH_SECS", 30)) * time.Second,
		DisplayName:   envOr("CONTROL_DISPLAY_NAME", "FLY VPN"),
		OrgName:       envOr("CONTROL_ORG_NAME", "FLY VPN"),
		ServerAddr:    os.Getenv("CONTROL_SERVER_ADDR"),
		ServerID:      os.Getenv("CONTROL_SERVER_ID"),
		// MVP-тарифы: срок + кап. Расширяется без кода продукта (потом — из БД/конфига).
		Plans: map[string]contract.Plan{
			"30d-10g": {Duration: 30 * 24 * time.Hour, CapBytes: 10 << 30},
			"30d-30g": {Duration: 30 * 24 * time.Hour, CapBytes: 30 << 30},
		},
	}
	if c.DSN == "" {
		return Config{}, fmt.Errorf("CONTROL_DSN обязателен")
	}
	return c, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

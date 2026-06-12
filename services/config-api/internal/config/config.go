// Package config загружает конфигурацию сервиса из окружения.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config — параметры запуска config-api. Секреты приходят из окружения, не из файлов.
type Config struct {
	ListenAddr   string // адрес plain-HTTP (если ACME выключен), напр. ":8443"
	DatabaseURL  string // DSN PostgreSQL (pgx)
	VPNRemote    string // vpn.X.com — RemoteAddress в профиле
	VPNRemoteID  string // идентификатор сервера (CN/домен серверного серта)
	ServerCACN   string // CN издателя серверного сертификата
	Organization string // PayloadOrganization
	DisplayName  string // отображаемое имя профиля

	// ACME (Let's Encrypt). Если ACMEDomain задан — сервис слушает :443 (TLS,
	// автосерт) + :80 (HTTP-01 challenge). Иначе — plain HTTP на ListenAddr.
	ACMEDomain   string
	ACMECacheDir string

	// Digiseller (Plati.market). Опциональны: без них сервис стартует, но /plati/issue
	// отвечает 503. Активируются, когда заведён аккаунт продавца.
	DigisellerSellerID int
	DigisellerAPIKey   string

	// Маппинг id товара Digiseller -> план (30d|90d|365d). Если товар не в маппинге —
	// берётся DefaultPlan.
	PlanByGoods map[int]string
	DefaultPlan string
}

// TLSEnabled — включён ли ACME/TLS.
func (c Config) TLSEnabled() bool { return c.ACMEDomain != "" }

// DigisellerEnabled — заданы ли креды Digiseller.
func (c Config) DigisellerEnabled() bool { return c.DigisellerSellerID != 0 && c.DigisellerAPIKey != "" }

// PlanFor возвращает план для id товара (или DefaultPlan).
func (c Config) PlanFor(idGoods int) string {
	if p, ok := c.PlanByGoods[idGoods]; ok {
		return p
	}
	return c.DefaultPlan
}

// FromEnv читает конфиг из переменных окружения, проверяя обязательные.
func FromEnv() (Config, error) {
	c := Config{
		ListenAddr:         getenv("CONFIGAPI_LISTEN", ":8443"),
		DatabaseURL:        os.Getenv("CONFIGAPI_DATABASE_URL"),
		VPNRemote:          os.Getenv("CONFIGAPI_VPN_REMOTE"),
		VPNRemoteID:        os.Getenv("CONFIGAPI_VPN_REMOTE_ID"),
		ServerCACN:         os.Getenv("CONFIGAPI_SERVER_CA_CN"),
		Organization:       getenv("CONFIGAPI_ORG", "Smart Internet"),
		DisplayName:        getenv("CONFIGAPI_DISPLAY_NAME", "Smart Internet"),
		ACMEDomain:         os.Getenv("CONFIGAPI_ACME_DOMAIN"),
		ACMECacheDir:       getenv("CONFIGAPI_ACME_CACHE", "/var/lib/config-api/acme"),
		DigisellerAPIKey:   os.Getenv("CONFIGAPI_DIGISELLER_API_KEY"),
		DefaultPlan:        getenv("CONFIGAPI_DEFAULT_PLAN", "30d"),
		PlanByGoods:        parsePlanMap(os.Getenv("CONFIGAPI_PLAN_BY_GOODS")),
	}
	if v := os.Getenv("CONFIGAPI_DIGISELLER_SELLER_ID"); v != "" {
		c.DigisellerSellerID, _ = strconv.Atoi(v)
	}
	for k, v := range map[string]string{
		"CONFIGAPI_DATABASE_URL": c.DatabaseURL,
		"CONFIGAPI_VPN_REMOTE":   c.VPNRemote,
	} {
		if v == "" {
			return Config{}, fmt.Errorf("обязательная переменная %s не задана", k)
		}
	}
	if c.VPNRemoteID == "" {
		c.VPNRemoteID = c.VPNRemote
	}
	return c, nil
}

// parsePlanMap парсит "12345:30d,67890:365d" -> {12345:"30d", 67890:"365d"}.
func parsePlanMap(s string) map[int]string {
	m := map[int]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) != 2 {
			continue
		}
		if id, err := strconv.Atoi(strings.TrimSpace(kv[0])); err == nil {
			m[id] = strings.TrimSpace(kv[1])
		}
	}
	return m
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

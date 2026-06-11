// Package config загружает конфигурацию сервиса из окружения.
package config

import (
	"fmt"
	"os"
)

// Config — параметры запуска config-api. Секреты приходят из окружения, не из файлов.
type Config struct {
	ListenAddr   string // адрес plain-HTTP (если ACME выключен), напр. ":8443"
	DatabaseURL  string // DSN PostgreSQL (pgx)
	PlatiSecret  string // секрет для проверки HMAC-подписи Plati
	VPNRemote    string // vpn.X.com — RemoteAddress в профиле
	VPNRemoteID  string // идентификатор сервера (CN/домен серверного серта)
	ServerCACN   string // CN издателя серверного сертификата
	Organization string // PayloadOrganization
	DisplayName  string // отображаемое имя профиля

	// ACME (Let's Encrypt). Если ACMEDomain задан — сервис слушает :443 (TLS,
	// автосерт) + :80 (HTTP-01 challenge). Иначе — plain HTTP на ListenAddr.
	ACMEDomain   string
	ACMECacheDir string
}

// TLSEnabled — включён ли ACME/TLS.
func (c Config) TLSEnabled() bool { return c.ACMEDomain != "" }

// FromEnv читает конфиг из переменных окружения, проверяя обязательные.
func FromEnv() (Config, error) {
	c := Config{
		ListenAddr:   getenv("CONFIGAPI_LISTEN", ":8443"),
		DatabaseURL:  os.Getenv("CONFIGAPI_DATABASE_URL"),
		PlatiSecret:  os.Getenv("CONFIGAPI_PLATI_SECRET"),
		VPNRemote:    os.Getenv("CONFIGAPI_VPN_REMOTE"),
		VPNRemoteID:  os.Getenv("CONFIGAPI_VPN_REMOTE_ID"),
		ServerCACN:   os.Getenv("CONFIGAPI_SERVER_CA_CN"),
		Organization: getenv("CONFIGAPI_ORG", "Smart Internet"),
		DisplayName:  getenv("CONFIGAPI_DISPLAY_NAME", "Smart Internet"),
		ACMEDomain:   os.Getenv("CONFIGAPI_ACME_DOMAIN"),
		ACMECacheDir: getenv("CONFIGAPI_ACME_CACHE", "/var/lib/config-api/acme"),
	}
	for k, v := range map[string]string{
		"CONFIGAPI_DATABASE_URL": c.DatabaseURL,
		"CONFIGAPI_PLATI_SECRET": c.PlatiSecret,
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

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

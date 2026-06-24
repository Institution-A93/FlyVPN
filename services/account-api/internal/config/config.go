// Package config загружает конфигурацию account-api из окружения (секреты — не из файлов).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Трафик-константы MVP (docs/backend-requirements.md §3, §6).
const (
	GiB = 1 << 30
	MiB = 1 << 20

	TrialLimitBytes = 300 * MiB // триал: 300 МБ
	PaidLimitBytes  = 10 * GiB  // платный пакет: 10 ГБ/мес
	ReferralBonus   = 1 * GiB   // +1 ГБ рефереру за первый платёж приглашённого

	TrialDuration = 30 * 24 * time.Hour
	PaidDuration  = 30 * 24 * time.Hour // month-only (decision #4)
)

// Config — параметры запуска account-api.
type Config struct {
	ListenAddr  string // адрес HTTP, напр. ":8080" (TLS терминируется reverse-proxy)
	DatabaseURL string // DSN PostgreSQL (pgx)

	// Профиль .mobileconfig (как у config-api).
	VPNRemote    string // RemoteAddress в профиле (ingress endpoint)
	VPNRemoteID  string // RemoteIdentifier (CN/домен серверного серта)
	Organization string
	DisplayName  string

	// JWT/сессии.
	JWTSecret     string        // подпись access-токенов (HS256)
	AccessTTL     time.Duration // время жизни access-токена
	RefreshTTL    time.Duration // время жизни refresh-токена/сессии
	PublicBaseURL string        // https://flynet.pro — для реф-ссылок и returnUrl

	// Telegram login-widget (валидация hash). Токен бота — общий с ботом, но здесь
	// нужен только для проверки подписи виджета.
	TelegramBotToken string

	// Twilio (OTP по SMS). Prototype: если выключено — код фиксирован 0000 (см. auth).
	TwilioEnabled    bool
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	// Platega.io (ADR-0020). Без кредов /packages/{id}/purchase отдаёт 503.
	PlategaBaseURL    string
	PlategaMerchantID string
	PlategaSecret     string
}

// TelegramEnabled — задан ли токен для валидации Telegram login-widget.
func (c Config) TelegramEnabled() bool { return c.TelegramBotToken != "" }

// PlategaEnabled — заданы ли креды Platega.
func (c Config) PlategaEnabled() bool { return c.PlategaMerchantID != "" && c.PlategaSecret != "" }

// FromEnv читает конфиг из переменных окружения, проверяя обязательные.
func FromEnv() (Config, error) {
	c := Config{
		ListenAddr:        getenv("ACCOUNTAPI_LISTEN", ":8080"),
		DatabaseURL:       os.Getenv("ACCOUNTAPI_DATABASE_URL"),
		VPNRemote:         os.Getenv("ACCOUNTAPI_VPN_REMOTE"),
		VPNRemoteID:       os.Getenv("ACCOUNTAPI_VPN_REMOTE_ID"),
		Organization:      getenv("ACCOUNTAPI_ORG", "FLY VPN"),
		DisplayName:       getenv("ACCOUNTAPI_DISPLAY_NAME", "FLY VPN"),
		JWTSecret:         os.Getenv("ACCOUNTAPI_JWT_SECRET"),
		AccessTTL:         getdur("ACCOUNTAPI_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:        getdur("ACCOUNTAPI_REFRESH_TTL", 30*24*time.Hour),
		PublicBaseURL:     getenv("ACCOUNTAPI_PUBLIC_BASE_URL", "https://flynet.pro"),
		TelegramBotToken:  os.Getenv("ACCOUNTAPI_TELEGRAM_BOT_TOKEN"),
		TwilioAccountSID:  os.Getenv("ACCOUNTAPI_TWILIO_SID"),
		TwilioAuthToken:   os.Getenv("ACCOUNTAPI_TWILIO_TOKEN"),
		TwilioFromNumber:  os.Getenv("ACCOUNTAPI_TWILIO_FROM"),
		PlategaBaseURL:    getenv("ACCOUNTAPI_PLATEGA_BASE_URL", "https://app.platega.io"),
		PlategaMerchantID: os.Getenv("ACCOUNTAPI_PLATEGA_MERCHANT_ID"),
		PlategaSecret:     os.Getenv("ACCOUNTAPI_PLATEGA_SECRET"),
	}
	c.TwilioEnabled = c.TwilioAccountSID != "" && c.TwilioAuthToken != "" && c.TwilioFromNumber != ""

	for k, v := range map[string]string{
		"ACCOUNTAPI_DATABASE_URL": c.DatabaseURL,
		"ACCOUNTAPI_VPN_REMOTE":   c.VPNRemote,
		"ACCOUNTAPI_JWT_SECRET":   c.JWTSecret,
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

func getdur(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return def
}

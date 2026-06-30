// Package profiles рендерит клиентские профили: iOS/macOS .mobileconfig (IKEv2 +
// EAP-MSCHAPv2) и Android .sswan (strongSwan-app). Оба несут одни EAP-креды
// (username/password) и адрес узла; доставляет Telegram-бот (ADR-0022). Перенесено из
// config-api + добавлен .sswan.
package profiles

import (
	"encoding/json"
	"fmt"
	"strings"

	_ "embed"
)

//go:embed profile.mobileconfig.tmpl
var mobileconfigTemplate string

// Params — общие параметры профиля для обеих платформ.
type Params struct {
	DisplayName string // "FLY VPN"
	OrgName     string
	ServerAddr  string // адрес ingress-узла (домен)
	ServerID    string // remote identifier (CN серта = домен)
	Username    string // EAP-логин (auth_credentials.username)
	Password    string // EAP-пароль (отдаётся единожды; в БД только nt_hash)
	// идентификаторы (UUID) генерит вызывающий
	ProfileIdentifier string
	ProfileUUID       string
	PayloadUUID       string
}

// Mobileconfig рендерит Apple Configuration Profile (iOS/macOS).
func Mobileconfig(p Params) ([]byte, error) {
	repl := map[string]string{
		"{{PROFILE_IDENTIFIER}}":    p.ProfileIdentifier,
		"{{PROFILE_UUID}}":          p.ProfileUUID,
		"{{PAYLOAD_UUID}}":          p.PayloadUUID,
		"{{DISPLAY_NAME}}":          p.DisplayName,
		"{{ORG_NAME}}":              p.OrgName,
		"{{VPN_REMOTE_ADDRESS}}":    p.ServerAddr,
		"{{VPN_REMOTE_IDENTIFIER}}": p.ServerID,
		"{{EAP_USERNAME}}":          p.Username,
		"{{EAP_PASSWORD}}":          p.Password,
	}
	out := mobileconfigTemplate
	for token, val := range repl {
		if val == "" {
			return nil, fmt.Errorf("mobileconfig: пустое значение для %s", token)
		}
		out = strings.ReplaceAll(out, token, xmlEscape(val))
	}
	if strings.Contains(out, "{{") {
		return nil, fmt.Errorf("mobileconfig: остались неподставленные токены")
	}
	return []byte(out), nil
}

// sswan — структура профиля strongSwan VPN Client для Android (.sswan, JSON).
// Импортируется приложением (с 1.8.0). Пароль в файле не хранится (приложение
// спрашивает/сохраняет) — доставляется ботом рядом с профилем.
type sswan struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Version int    `json:"version"`
	Type    string `json:"type"` // ikev2-eap (username/password)
	Remote  struct {
		Addr string `json:"addr"`
		ID   string `json:"id"`
	} `json:"remote"`
	Local struct {
		EapID string `json:"eap_id"`
	} `json:"local"`
}

// Sswan рендерит Android strongSwan-профиль (.sswan).
func Sswan(p Params) ([]byte, error) {
	if p.ServerAddr == "" || p.Username == "" || p.PayloadUUID == "" {
		return nil, fmt.Errorf("sswan: пустые обязательные поля")
	}
	var s sswan
	s.UUID = p.PayloadUUID
	s.Name = p.DisplayName
	s.Version = 1
	s.Type = "ikev2-eap"
	s.Remote.Addr = p.ServerAddr
	s.Remote.ID = p.ServerID
	s.Local.EapID = p.Username
	return json.MarshalIndent(&s, "", "  ")
}

func xmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;",
	).Replace(s)
}

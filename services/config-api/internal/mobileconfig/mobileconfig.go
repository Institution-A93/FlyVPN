// Package mobileconfig рендерит Apple Configuration Profile из шаблона с токенами.
package mobileconfig

import (
	"fmt"
	"strings"
)

// Fields — значения для подстановки в шаблон profile.mobileconfig.tmpl.
type Fields struct {
	ProfileIdentifier string
	ProfileUUID       string
	PayloadUUID       string
	DisplayName       string
	OrgName           string
	VPNRemoteAddress  string
	VPNRemoteID       string
	ServerCACN        string
	EAPUsername       string
	EAPPassword       string
}

// Render подставляет токены {{NAME}} в шаблон. Возвращает готовый .mobileconfig.
// Все обязательные поля должны быть непусты.
func Render(tmpl string, f Fields) ([]byte, error) {
	repl := map[string]string{
		"{{PROFILE_IDENTIFIER}}":   f.ProfileIdentifier,
		"{{PROFILE_UUID}}":         f.ProfileUUID,
		"{{PAYLOAD_UUID}}":         f.PayloadUUID,
		"{{DISPLAY_NAME}}":         f.DisplayName,
		"{{ORG_NAME}}":             f.OrgName,
		"{{VPN_REMOTE_ADDRESS}}":   f.VPNRemoteAddress,
		"{{VPN_REMOTE_IDENTIFIER}}": f.VPNRemoteID,
		"{{SERVER_CA_CN}}":         f.ServerCACN,
		"{{EAP_USERNAME}}":         f.EAPUsername,
		"{{EAP_PASSWORD}}":         f.EAPPassword,
	}
	out := tmpl
	for token, val := range repl {
		if val == "" {
			return nil, fmt.Errorf("mobileconfig: пустое значение для %s", token)
		}
		out = strings.ReplaceAll(out, token, xmlEscape(val))
	}
	if strings.Contains(out, "{{") {
		return nil, fmt.Errorf("mobileconfig: в шаблоне остались неподставленные токены")
	}
	return []byte(out), nil
}

// xmlEscape экранирует значения, попадающие в XML-текст профиля.
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

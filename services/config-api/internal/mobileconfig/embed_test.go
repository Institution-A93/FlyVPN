package mobileconfig_test

import (
	"encoding/xml"
	"strings"
	"testing"

	configapi "github.com/institution-a93/flyvpn/services/config-api"
	"github.com/institution-a93/flyvpn/services/config-api/internal/mobileconfig"
)

// TestRenderRealEmbeddedTemplate рендерит НАСТОЯЩИЙ встроенный шаблон (а не мини-заглушку):
// ловит токены/`{{`, которые есть в боевом profile.mobileconfig.tmpl, но не маппятся в Render
// (регрессия: `{{NAME}}` в комментарии шаблона валил выдачу 500 на проде).
func TestRenderRealEmbeddedTemplate(t *testing.T) {
	f := mobileconfig.Fields{
		ProfileIdentifier: "com.smartinternet.vpn.00000000",
		ProfileUUID:       "11111111-1111-1111-1111-111111111111",
		PayloadUUID:       "22222222-2222-2222-2222-222222222222",
		DisplayName:       "Smart Internet",
		OrgName:           "Smart Internet",
		VPNRemoteAddress:  "vpn.fly-vpn.net",
		VPNRemoteID:       "vpn.fly-vpn.net",
		ServerCACN:        "R3",
		EAPUsername:       "testuser",
		EAPPassword:       "testpass",
	}
	out, err := mobileconfig.Render(configapi.ProfileTemplate, f)
	if err != nil {
		t.Fatalf("Render(боевой шаблон) ошибка: %v", err)
	}
	if strings.Contains(string(out), "{{") {
		t.Error("в боевом шаблоне остались неподставленные токены")
	}
	if err := xml.Unmarshal(out, new(struct{})); err != nil {
		t.Errorf("результат не well-formed XML: %v", err)
	}
}

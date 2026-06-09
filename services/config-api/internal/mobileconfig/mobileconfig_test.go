package mobileconfig

import (
	"encoding/xml"
	"strings"
	"testing"
)

const miniTmpl = `<plist><dict>
<key>id</key><string>{{PROFILE_IDENTIFIER}}</string>
<key>puuid</key><string>{{PROFILE_UUID}}</string>
<key>payload</key><string>{{PAYLOAD_UUID}}</string>
<key>name</key><string>{{DISPLAY_NAME}}</string>
<key>org</key><string>{{ORG_NAME}}</string>
<key>remote</key><string>{{VPN_REMOTE_ADDRESS}}</string>
<key>remoteid</key><string>{{VPN_REMOTE_IDENTIFIER}}</string>
<key>ca</key><string>{{SERVER_CA_CN}}</string>
<key>user</key><string>{{EAP_USERNAME}}</string>
<key>pass</key><string>{{EAP_PASSWORD}}</string>
</dict></plist>`

func validFields() Fields {
	return Fields{
		ProfileIdentifier: "com.x.vpn.1", ProfileUUID: "U1", PayloadUUID: "U2",
		DisplayName: "Smart Internet", OrgName: "X", VPNRemoteAddress: "vpn.x.com",
		VPNRemoteID: "vpn.x.com", ServerCACN: "R3", EAPUsername: "abc", EAPPassword: "p&w<d>",
	}
}

func TestRenderReplacesAllTokensAndIsWellFormed(t *testing.T) {
	out, err := Render(miniTmpl, validFields())
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if strings.Contains(string(out), "{{") {
		t.Error("остались неподставленные токены")
	}
	// XML well-formed (экранирование спецсимволов пароля сработало)
	if err := xml.Unmarshal(out, new(struct{})); err != nil {
		t.Errorf("результат не well-formed XML: %v", err)
	}
	if !strings.Contains(string(out), "p&amp;w&lt;d&gt;") {
		t.Error("пароль со спецсимволами не экранирован")
	}
}

func TestRenderRejectsEmptyField(t *testing.T) {
	f := validFields()
	f.EAPPassword = ""
	if _, err := Render(miniTmpl, f); err == nil {
		t.Error("ожидалась ошибка на пустом поле")
	}
}

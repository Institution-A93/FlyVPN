package profiles

import (
	"encoding/json"
	"strings"
	"testing"
)

func sample() Params {
	return Params{
		DisplayName: "FLY VPN", OrgName: "FLY", ServerAddr: "vpn.example.net",
		ServerID: "vpn.example.net", Username: "alice123", Password: "S3cr3t-Pass-Word",
		ProfileIdentifier: "pro.flynet.vpn.abc", ProfileUUID: "11111111-1111-1111-1111-111111111111",
		PayloadUUID: "22222222-2222-2222-2222-222222222222",
	}
}

func TestMobileconfig(t *testing.T) {
	b, err := Mobileconfig(sample())
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "{{") {
		t.Error("остались неподставленные токены")
	}
	for _, want := range []string{"alice123", "vpn.example.net", "IKEv2", "S3cr3t-Pass-Word"} {
		if !strings.Contains(s, want) {
			t.Errorf("в .mobileconfig нет %q", want)
		}
	}
}

func TestMobileconfigEmptyField(t *testing.T) {
	p := sample()
	p.Username = ""
	if _, err := Mobileconfig(p); err == nil {
		t.Error("ожидалась ошибка на пустом username")
	}
}

func TestSswan(t *testing.T) {
	b, err := Sswan(sample())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("невалидный JSON: %v", err)
	}
	if m["type"] != "ikev2-eap" {
		t.Errorf("type = %v, ждали ikev2-eap", m["type"])
	}
	local := m["local"].(map[string]any)
	if local["eap_id"] != "alice123" {
		t.Errorf("eap_id = %v", local["eap_id"])
	}
}

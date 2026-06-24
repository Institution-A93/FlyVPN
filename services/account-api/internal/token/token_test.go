package token

import (
	"testing"
	"time"
)

func TestAccessRoundTrip(t *testing.T) {
	iss := New("supersecret", 15*time.Minute, 24*time.Hour)
	tok, err := iss.Access("user-123")
	if err != nil {
		t.Fatalf("access: %v", err)
	}
	sub, err := iss.ParseAccess(tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sub != "user-123" {
		t.Fatalf("sub = %q, want user-123", sub)
	}
	// Чужой секрет не валидируется.
	if _, err := New("other", time.Minute, time.Hour).ParseAccess(tok); err == nil {
		t.Fatal("token валиден под чужим секретом")
	}
}

func TestExpiredAccess(t *testing.T) {
	iss := New("s", -time.Minute, time.Hour) // уже истёкший
	tok, _ := iss.Access("u")
	if _, err := iss.ParseAccess(tok); err == nil {
		t.Fatal("истёкший токен принят")
	}
}

func TestRefreshHash(t *testing.T) {
	pt, h, err := NewRefresh()
	if err != nil {
		t.Fatal(err)
	}
	if HashRefresh(pt) != h {
		t.Fatal("HashRefresh не детерминирован")
	}
	if pt == h {
		t.Fatal("plaintext == hash")
	}
}

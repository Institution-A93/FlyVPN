package credentials

import "testing"

// NT-hash должен совпадать с config-api (тот же вектор: "password").
func TestNTHashKnownVector(t *testing.T) {
	// MD4(UTF-16LE("password")) = 8846f7eaee8fb117ad06bdd830b7586c
	if got := NTHash("password"); got != "8846f7eaee8fb117ad06bdd830b7586c" {
		t.Fatalf("NTHash(password) = %q, want 8846f7eaee8fb117ad06bdd830b7586c", got)
	}
}

func TestGenerators(t *testing.T) {
	u, err := GenerateUsername()
	if err != nil || len(u) != usernameLen {
		t.Fatalf("username %q err=%v", u, err)
	}
	p, err := GeneratePassword()
	if err != nil || len(p) != passwordLen {
		t.Fatalf("password len=%d err=%v", len(p), err)
	}
	if u2, _ := GenerateUsername(); u2 == u {
		t.Error("usernames не случайны")
	}
}

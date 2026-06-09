package credentials

import "testing"

// Известные векторы NT-hash (проверены внешне: smbencrypt /公known vectors).
func TestNTHash(t *testing.T) {
	cases := map[string]string{
		"password":                    "8846f7eaee8fb117ad06bdd830b7586c",
		"S3cret-MMVP-pass-0123456789": "db9a6949e84a9a0c08ef4ea3a1506b32",
	}
	for pw, want := range cases {
		if got := NTHash(pw); got != want {
			t.Errorf("NTHash(%q) = %s, want %s", pw, got, want)
		}
	}
}

func TestGenerators(t *testing.T) {
	u, err := GenerateUsername()
	if err != nil || len(u) != usernameLen {
		t.Fatalf("username: %q err=%v", u, err)
	}
	p, err := GeneratePassword()
	if err != nil || len(p) != passwordLen {
		t.Fatalf("password: %q err=%v", p, err)
	}
	// NT-hash сгенерированного пароля — валидный 32-hex.
	if h := NTHash(p); len(h) != 32 {
		t.Errorf("nt_hash len = %d, want 32", len(h))
	}
	u2, _ := GenerateUsername()
	if u == u2 {
		t.Errorf("username не случаен: %q == %q", u, u2)
	}
}

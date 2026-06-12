package digiseller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestSign(t *testing.T) {
	// Эталон: SHA256("secret123") при ts=100 => SHA256("secret123100").
	want := func() string { s := sha256.Sum256([]byte("secret123100")); return hex.EncodeToString(s[:]) }()
	if got := Sign("secret123", 100); got != want {
		t.Errorf("Sign = %s, want %s", got, want)
	}
}

func TestCheckUniqueCode(t *testing.T) {
	var loginCalls, codeCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/apilogin":
			loginCalls++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			// проверим, что подпись соответствует api_key+timestamp
			ts := int64(body["timestamp"].(float64))
			if body["sign"] != Sign("KEY", ts) {
				t.Errorf("bad sign in apilogin")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"retval": 0, "token": "TOK", "valid_thru": "2030-01-01T00:00:00"})
		case r.URL.Path == "/api/purchases/unique-code/ABC123":
			codeCalls++
			if r.URL.Query().Get("token") != "TOK" {
				t.Errorf("token not passed")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"retval": 0, "id_goods": 4242, "amount": 299.0, "email": "buyer@example.com",
				"date_pay": "2026-06-12 10:00:00", "unique_code_state": map[string]any{"state": 1},
			})
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer srv.Close()

	c := New(777, "KEY")
	c.SetBaseURL(srv.URL)

	p, err := c.CheckUniqueCode(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("CheckUniqueCode: %v", err)
	}
	if p.IDGoods != 4242 || p.Email != "buyer@example.com" || p.State != 1 {
		t.Errorf("unexpected purchase: %+v", p)
	}

	// Повторный вызов — токен из кэша (apilogin не дёргается снова).
	if _, err := c.CheckUniqueCode(context.Background(), "ABC123"); err != nil {
		t.Fatal(err)
	}
	if loginCalls != 1 {
		t.Errorf("token не закэширован: loginCalls=%d", loginCalls)
	}
	if codeCalls != 2 {
		t.Errorf("codeCalls=%d, want 2", codeCalls)
	}
}

// невалидный код (id_goods=0) -> ошибка
func TestCheckUniqueCodeInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/apilogin" {
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "T"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"retval": 2, "retdesc": "not found", "id_goods": 0})
	}))
	defer srv.Close()
	c := New(1, "k")
	c.SetBaseURL(srv.URL)
	if _, err := c.CheckUniqueCode(context.Background(), "X"+strconv.Itoa(1)); err == nil {
		t.Error("ожидалась ошибка на невалидный код")
	}
}

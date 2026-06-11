package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// Интеграционный тест: запускается только если задан CONFIGAPI_TEST_DSN.
// go test без DSN — пропускает (default-прогон зелёный без БД).
func TestIssueIntegration(t *testing.T) {
	dsn := os.Getenv("CONFIGAPI_TEST_DSN")
	if dsn == "" {
		t.Skip("CONFIGAPI_TEST_DSN не задан — пропуск интеграционного теста")
	}
	ctx := context.Background()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer st.Close()

	buyer := "buyer-" + time.Now().Format("150405.000")
	p := IssueParams{
		PlatiBuyerID: buyer, Email: "x@y.z", PlatiOrderID: buyer + "-o1",
		Plan: "30d", AmountRub: 299, Duration: 30 * 24 * time.Hour,
		CandidateUser: "user_" + buyer[len(buyer)-6:], NTHash: "8846f7eaee8fb117ad06bdd830b7586c",
	}

	r1, err := st.Issue(ctx, p)
	if err != nil {
		t.Fatalf("issue1: %v", err)
	}
	if r1.Username != p.CandidateUser {
		t.Errorf("username = %q, want %q", r1.Username, p.CandidateUser)
	}
	if r1.FramedIP == "" {
		t.Error("framed_ip пуст")
	}

	// Идемпотентность по order_id: повтор того же заказа.
	r2, err := st.Issue(ctx, p)
	if err != nil {
		t.Fatalf("issue2: %v", err)
	}
	if r2.Username != r1.Username || r2.FramedIP != r1.FramedIP {
		t.Errorf("повтор заказа изменил креды: %+v vs %+v", r1, r2)
	}

	// Продление: новый order, тот же buyer — username/ip стабильны, nt_hash ротируется.
	p2 := p
	p2.PlatiOrderID = buyer + "-o2"
	p2.NTHash = "db9a6949e84a9a0c08ef4ea3a1506b32"
	r3, err := st.Issue(ctx, p2)
	if err != nil {
		t.Fatalf("issue3 (renewal): %v", err)
	}
	if r3.Username != r1.Username || r3.FramedIP != r1.FramedIP {
		t.Errorf("продление изменило username/ip: %+v vs %+v", r1, r3)
	}

	// Проверки в БД: 1 кред, 2 подписки, nt_hash ротирован.
	var creds, subs int
	var ntHash string
	_ = st.pool.QueryRow(ctx, `SELECT count(*) FROM auth_credentials ac JOIN users u ON u.id=ac.user_id WHERE u.plati_buyer_id=$1`, buyer).Scan(&creds)
	_ = st.pool.QueryRow(ctx, `SELECT count(*) FROM subscriptions s JOIN users u ON u.id=s.user_id WHERE u.plati_buyer_id=$1`, buyer).Scan(&subs)
	_ = st.pool.QueryRow(ctx, `SELECT ac.nt_hash FROM auth_credentials ac JOIN users u ON u.id=ac.user_id WHERE u.plati_buyer_id=$1`, buyer).Scan(&ntHash)
	if creds != 1 {
		t.Errorf("creds = %d, want 1", creds)
	}
	if subs != 2 {
		t.Errorf("subs = %d, want 2", subs)
	}
	if ntHash != p2.NTHash {
		t.Errorf("nt_hash = %s, want rotated %s", ntHash, p2.NTHash)
	}
}

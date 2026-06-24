package store

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Интеграционный тест cron-обязанностей. Требует ORCH_TEST_DSN (схема 0001..0005).
func TestLifecycle(t *testing.T) {
	dsn := os.Getenv("ORCH_TEST_DSN")
	if dsn == "" {
		t.Skip("ORCH_TEST_DSN не задан — пропуск")
	}
	ctx := context.Background()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	seq := func() string { return strconv.FormatInt(time.Now().UnixNano()+lcSeq.Add(1), 10) }

	// Юзер + платная подписка (лимит 1000 байт) + активный кред.
	var uid string
	if err := pool.QueryRow(ctx, `INSERT INTO users (status, referral_code) VALUES ('active',$1) RETURNING id`,
		"rc"+seq()[:10]).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO subscriptions
		(user_id, plan, kind, started_at, expires_at, current_period_start, traffic_bytes_limit, amount_rub, status)
		VALUES ($1,'30d','paid', now(), now()+interval '30 days', now(), 1000, 300, 'active')`, uid); err != nil {
		t.Fatal(err)
	}
	ip := "10." + seq()[:1] + ".0.5" // грубо-уникальный, нам важна сама строка
	_ = ip
	framed := "10.9." + strconv.Itoa(int(lcSeq.Load()%250)) + "." + strconv.Itoa(int(lcSeq.Load()%200+1))
	var credID string
	if err := pool.QueryRow(ctx, `INSERT INTO auth_credentials (user_id, username, nt_hash, framed_ip)
		VALUES ($1,$2,'8846f7eaee8fb117ad06bdd830b7586c',$3) RETURNING id`,
		uid, "u"+seq(), framed).Scan(&credID); err != nil {
		t.Fatal(err)
	}

	// Потребление сверх лимита (2000 > 1000).
	if _, err := pool.Exec(ctx, `INSERT INTO usage_log (user_id, date, bytes_in, bytes_out)
		VALUES ($1, current_date, 1500, 500)`, uid); err != nil {
		t.Fatal(err)
	}

	// EnforceQuota → кред отозван с reason='quota'.
	if _, err := st.EnforceQuota(ctx); err != nil {
		t.Fatalf("enforce: %v", err)
	}
	if reason := credReason(t, pool, ctx, credID); reason != "quota" {
		t.Fatalf("после EnforceQuota reason=%q, want quota", reason)
	}

	// Порог: 100mb/1gb не сработают (лимит крошечный), но проверим, что вызов не падает.
	if _, err := st.EnqueueThresholdEvents(ctx); err != nil {
		t.Fatalf("threshold: %v", err)
	}

	// Снизим потребление под лимит → RestoreUnderQuota вернёт кред.
	if _, err := pool.Exec(ctx, `DELETE FROM usage_log WHERE user_id=$1`, uid); err != nil {
		t.Fatal(err)
	}
	if _, err := st.RestoreUnderQuota(ctx); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if revoked := credRevoked(t, pool, ctx, credID); revoked {
		t.Fatal("кред должен быть восстановлен после RestoreUnderQuota")
	}

	// Истекаем подписку → ExpireSubscriptions + RevokeExpiredCreds (reason='expiry').
	if _, err := pool.Exec(ctx, `UPDATE subscriptions SET expires_at = now() - interval '1 hour' WHERE user_id=$1`, uid); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ExpireSubscriptions(ctx); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if _, err := st.RevokeExpiredCreds(ctx); err != nil {
		t.Fatalf("revoke expired: %v", err)
	}
	if reason := credReason(t, pool, ctx, credID); reason != "expiry" {
		t.Fatalf("после истечения reason=%q, want expiry", reason)
	}

	// sub_expired-событие поставлено.
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events
		WHERE user_id=$1 AND type='sub_expired'`, uid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if _, err := st.EnqueueThresholdEvents(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events
		WHERE user_id=$1 AND type='sub_expired'`, uid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("sub_expired событий = %d, want 1 (dedupe)", n)
	}

	// OTP cleanup не падает.
	if _, err := st.CleanupOTP(ctx); err != nil {
		t.Fatalf("otp cleanup: %v", err)
	}
}

var lcSeq atomic.Int64

func credReason(t *testing.T, pool *pgxpool.Pool, ctx context.Context, id string) string {
	t.Helper()
	var reason *string
	if err := pool.QueryRow(ctx, `SELECT revoked_reason FROM auth_credentials WHERE id=$1`, id).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason == nil {
		return ""
	}
	return *reason
}

func credRevoked(t *testing.T, pool *pgxpool.Pool, ctx context.Context, id string) bool {
	t.Helper()
	var revoked *time.Time
	if err := pool.QueryRow(ctx, `SELECT revoked_at FROM auth_credentials WHERE id=$1`, id).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	return revoked != nil
}

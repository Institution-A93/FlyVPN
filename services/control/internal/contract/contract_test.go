package contract

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Интеграционный тест: нужен PostgreSQL со схемой 0001+0006. DSN — в CONTROL_TEST_DSN.
// Скип, если не задан (как в account-api).
func testPool(t *testing.T) *pgxpool.Pool {
	dsn := os.Getenv("CONTROL_TEST_DSN")
	if dsn == "" {
		t.Skip("CONTROL_TEST_DSN не задан — пропуск интеграционного теста")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return pool
}

func TestContractLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	defer pool.Close()

	// чистый юзер (FK auth_credentials.user_id → users)
	uid := "22222222-2222-2222-2222-222222222222"
	_, _ = pool.Exec(ctx, `DELETE FROM auth_credentials WHERE user_id=$1`, uid)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, uid)
	if _, err := pool.Exec(ctx, `INSERT INTO users (id, status) VALUES ($1,'active')`, uid); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM auth_credentials WHERE user_id=$1`, uid)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, uid)
	})

	c := New(pool)
	plan := Plan{Duration: 30 * 24 * time.Hour, CapBytes: 10 << 30} // 30д / 10 ГБ

	// Provision
	pr, err := c.Provision(ctx, uid, plan)
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	if pr.Username == "" || pr.Password == "" {
		t.Fatal("provision: пустые креды")
	}
	if got := pr.FramedIP[:5]; got != "10.8." && got != "10.9." && got != "10.10" && got != "10.11" {
		t.Errorf("framed_ip вне active-пула 10.8.0.0/14: %s", pr.FramedIP)
	}

	// Get + проверка лимитов
	cr, err := c.Get(ctx, uid)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if cr.ExpiresAt == nil || cr.ExpiresAt.Before(time.Now().Add(29*24*time.Hour)) {
		t.Errorf("expires_at не выставлен на ~30д: %v", cr.ExpiresAt)
	}
	if cr.TrafficCap == nil || *cr.TrafficCap != 10<<30 {
		t.Errorf("кап неверный: %v", cr.TrafficCap)
	}
	if cr.RevokedAt != nil {
		t.Error("свежий кред не должен быть revoked")
	}

	// nt-hash до продления
	var ntBefore string
	_ = pool.QueryRow(ctx, `SELECT nt_hash FROM auth_credentials WHERE user_id=$1`, uid).Scan(&ntBefore)
	expBefore := *cr.ExpiresAt
	periodBefore := cr.PeriodStart

	time.Sleep(1100 * time.Millisecond) // чтобы now() заметно сдвинулся

	// Renew: срок продлён, period_start сдвинут, nt-hash НЕ изменился
	if err := c.Renew(ctx, uid, plan); err != nil {
		t.Fatalf("renew: %v", err)
	}
	cr2, _ := c.Get(ctx, uid)
	if !cr2.ExpiresAt.After(expBefore) {
		t.Errorf("renew не продлил срок: было %v стало %v", expBefore, cr2.ExpiresAt)
	}
	if !cr2.PeriodStart.After(periodBefore) {
		t.Errorf("renew не сдвинул period_start (счётчик не обнулился): было %v стало %v", periodBefore, cr2.PeriodStart)
	}
	var ntAfter string
	_ = pool.QueryRow(ctx, `SELECT nt_hash FROM auth_credentials WHERE user_id=$1`, uid).Scan(&ntAfter)
	if ntBefore != ntAfter {
		t.Error("nt-hash ДОЛЖЕН быть стабильным на продлении, но изменился")
	}

	// Revoke
	if err := c.Revoke(ctx, uid); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	cr3, _ := c.Get(ctx, uid)
	if cr3.RevokedAt == nil {
		t.Error("revoke не выставил revoked_at")
	}

	// Renew снова снимает revoke
	if err := c.Renew(ctx, uid, plan); err != nil {
		t.Fatalf("renew after revoke: %v", err)
	}
	cr4, _ := c.Get(ctx, uid)
	if cr4.RevokedAt != nil {
		t.Error("renew должен был снять revoked_at")
	}
}

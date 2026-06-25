package fleet

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type mockDis struct{ calls []string }

func (m *mockDis) Disconnect(_ context.Context, nasIP, username string) error {
	m.calls = append(m.calls, nasIP+"|"+username)
	return nil
}

func TestDisconnectUser(t *testing.T) {
	dsn := os.Getenv("CONTROL_TEST_DSN")
	if dsn == "" {
		t.Skip("CONTROL_TEST_DSN не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	uid := "33333333-3333-3333-3333-333333333333"
	exec := func(q string, args ...any) { _, _ = pool.Exec(ctx, q, args...) }
	exec(`DELETE FROM radacct WHERE username='covuser'`)
	exec(`DELETE FROM auth_credentials WHERE user_id=$1`, uid)
	exec(`DELETE FROM users WHERE id=$1`, uid)
	if _, err := pool.Exec(ctx, `INSERT INTO users (id,status) VALUES ($1,'active')`, uid); err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO auth_credentials (user_id,username,nt_hash,framed_ip,period_start) VALUES ($1,'covuser','e05afee4e22b6fe7e11549e2193c8202','10.8.0.7',now())`, uid)
	// одна ОТКРЫТАЯ сессия (acctstoptime NULL) на узле 203.0.113.5
	exec(`INSERT INTO radacct (acctsessionid,acctuniqueid,username,nasipaddress,acctstarttime) VALUES ('s','cov-1','covuser','203.0.113.5',now())`)
	// одна закрытая — не должна триггерить disconnect
	exec(`INSERT INTO radacct (acctsessionid,acctuniqueid,username,nasipaddress,acctstarttime,acctstoptime) VALUES ('s2','cov-2','covuser','203.0.113.5',now(),now())`)
	t.Cleanup(func() {
		exec(`DELETE FROM radacct WHERE username='covuser'`)
		exec(`DELETE FROM auth_credentials WHERE user_id=$1`, uid)
		exec(`DELETE FROM users WHERE id=$1`, uid)
	})

	f := New(pool)
	m := &mockDis{}
	if err := f.DisconnectUser(ctx, uid, m); err != nil {
		t.Fatalf("DisconnectUser: %v", err)
	}
	if len(m.calls) != 1 || m.calls[0] != "203.0.113.5|covuser" {
		t.Errorf("ждали 1 disconnect (203.0.113.5|covuser), получили %v", m.calls)
	}
}

package store

import (
	"context"
	"os"
	"testing"
)

func TestNodesIntegration(t *testing.T) {
	dsn := os.Getenv("ORCH_TEST_DSN")
	if dsn == "" {
		t.Skip("ORCH_TEST_DSN не задан — пропуск интеграционного теста")
	}
	ctx := context.Background()
	st, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer st.Close()

	id1, err := st.Register(ctx, "egress", "nl-ams", "203.0.113.50")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	// Идемпотентность по public_ip.
	id2, err := st.Register(ctx, "egress", "nl-ams", "203.0.113.50")
	if err != nil || id1 != id2 {
		t.Fatalf("повторная регистрация не идемпотентна: %s vs %s err=%v", id1, id2, err)
	}

	if err := st.SetStatus(ctx, id1, "down"); err != nil {
		t.Fatalf("setstatus: %v", err)
	}
	if err := st.Heartbeat(ctx, id1); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	nodes, err := st.List(ctx)
	if err != nil || len(nodes) == 0 {
		t.Fatalf("list: %v (n=%d)", err, len(nodes))
	}
	var found *Node
	for i := range nodes {
		if nodes[i].ID == id1 {
			found = &nodes[i]
		}
	}
	if found == nil {
		t.Fatal("узел не найден в списке")
	}
	if found.Status != "up" || found.LastHeartbeat == nil {
		t.Errorf("после heartbeat ожидался up + last_heartbeat, got status=%s hb=%v", found.Status, found.LastHeartbeat)
	}
}

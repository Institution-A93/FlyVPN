package health

import (
	"context"
	"net"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/institution-a93/flyvpn/services/orchestrator/internal/store"
)

func hostPort(t *testing.T, url string) (string, int) {
	t.Helper()
	// url вида https://127.0.0.1:PORT
	h, p, err := net.SplitHostPort(url[len("https://"):])
	if err != nil {
		t.Fatal(err)
	}
	n, _ := strconv.Atoi(p)
	return h, n
}

func TestTLSProbe(t *testing.T) {
	srv := httptest.NewTLSServer(nil)
	host, port := hostPort(t, srv.URL)
	node := store.Node{PublicIP: host, Role: "egress"}

	if err := TLSProbe(port)(context.Background(), node); err != nil {
		t.Errorf("живой TLS-сервер должен проходить пробу: %v", err)
	}
	srv.Close()
	if err := TLSProbe(port)(context.Background(), node); err == nil {
		t.Error("закрытый сервер должен валить пробу")
	}
}

// fakeReg реализует Registry в памяти.
type fakeReg struct {
	nodes  []store.Node
	status map[string]string
}

func (f *fakeReg) List(context.Context) ([]store.Node, error) { return f.nodes, nil }
func (f *fakeReg) SetStatus(_ context.Context, id, s string) error {
	f.status[id] = s
	return nil
}

func TestCheckerThreshold(t *testing.T) {
	reg := &fakeReg{nodes: []store.Node{{ID: "n1", Role: "egress", PublicIP: "203.0.113.255"}}, status: map[string]string{}}
	// Проба, которая всегда падает.
	failing := func(context.Context, store.Node) error { return context.DeadlineExceeded }
	c := NewChecker(reg, map[string]Probe{"egress": failing}, 3)

	c.RunOnce(context.Background())
	c.RunOnce(context.Background())
	if reg.status["n1"] == "down" {
		t.Error("узел помечен down раньше threshold")
	}
	c.RunOnce(context.Background())
	if reg.status["n1"] != "down" {
		t.Errorf("узел должен быть down после 3 неудач, got %q", reg.status["n1"])
	}

	// Успешная проба сбрасывает счётчик и поднимает up.
	c.probes["egress"] = func(context.Context, store.Node) error { return nil }
	c.RunOnce(context.Background())
	if reg.status["n1"] != "up" {
		t.Errorf("узел должен вернуться в up, got %q", reg.status["n1"])
	}
}

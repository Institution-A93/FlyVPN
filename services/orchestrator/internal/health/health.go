// Package health — активные проверки доступности узлов.
package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/institution-a93/flyvpn/services/orchestrator/internal/store"
)

// Probe проверяет один узел; nil = жив.
type Probe func(ctx context.Context, n store.Node) error

const dialTimeout = 5 * time.Second

// TLSProbe: успешный TLS-handshake к PublicIP:port (для egress Reality и config-api).
// Сертификат не верифицируем — Reality презентует серт сайта-донора; нам важен сам
// факт корректного TLS-ответа.
func TLSProbe(port int) Probe {
	return func(ctx context.Context, n store.Node) error {
		addr := net.JoinHostPort(n.PublicIP, strconv.Itoa(port))
		d := &tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
		ctx, cancel := context.WithTimeout(ctx, dialTimeout)
		defer cancel()
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("tls probe %s: %w", addr, err)
		}
		return conn.Close()
	}
}

// TCPProbe: успешный TCP-connect к PublicIP:port.
func TCPProbe(port int) Probe {
	return func(ctx context.Context, n store.Node) error {
		addr := net.JoinHostPort(n.PublicIP, strconv.Itoa(port))
		ctx, cancel := context.WithTimeout(ctx, dialTimeout)
		defer cancel()
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("tcp probe %s: %w", addr, err)
		}
		return conn.Close()
	}
}

// Registry — реестр, нужный Checker'у (для тестируемости).
type Registry interface {
	List(ctx context.Context) ([]store.Node, error)
	SetStatus(ctx context.Context, id, status string) error
}

// Checker гоняет пробы и обновляет статусы. Узел помечается down после threshold
// подряд-неудач (по роли; роли без пробы пропускаются — статус ведётся heartbeat'ом).
type Checker struct {
	reg       Registry
	probes    map[string]Probe
	threshold int
	fails     map[string]int
}

func NewChecker(reg Registry, probes map[string]Probe, threshold int) *Checker {
	return &Checker{reg: reg, probes: probes, threshold: threshold, fails: map[string]int{}}
}

// RunOnce — один проход по всем узлам.
func (c *Checker) RunOnce(ctx context.Context) error {
	nodes, err := c.reg.List(ctx)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		probe, ok := c.probes[n.Role]
		if !ok {
			continue // нет активной пробы для роли (напр. ingress на MMVP)
		}
		if err := probe(ctx, n); err != nil {
			c.fails[n.ID]++
			if c.fails[n.ID] >= c.threshold {
				_ = c.reg.SetStatus(ctx, n.ID, "down")
			}
		} else {
			c.fails[n.ID] = 0
			_ = c.reg.SetStatus(ctx, n.ID, "up")
		}
	}
	return nil
}

// Run гоняет RunOnce по тикеру до отмены контекста.
func (c *Checker) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	_ = c.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = c.RunOnce(ctx)
		}
	}
}

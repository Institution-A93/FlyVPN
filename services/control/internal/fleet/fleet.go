// Package fleet — реестр узлов, health-пробы и автопровизия (перенесено из
// orchestrator) + CoA-origination (новое, см. coa.go). Часть бинаря control (ADR-0021).
package fleet

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Fleet держит пул и отвечает за узлы.
type Fleet struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Fleet { return &Fleet{pool: pool} }

// Node — строка реестра.
type Node struct {
	ID            string     `json:"id"`
	Role          string     `json:"role"`
	Region        string     `json:"region"`
	PublicIP      string     `json:"public_ip"`
	Status        string     `json:"status"`
	LastHeartbeat *time.Time `json:"last_heartbeat,omitempty"`
	ConfigVersion int        `json:"config_version"`
}

// Register идемпотентно регистрирует узел по public_ip.
func (f *Fleet) Register(ctx context.Context, role, region, publicIP string) (string, error) {
	var id string
	err := f.pool.QueryRow(ctx,
		`INSERT INTO nodes (role, region, public_ip, status, deployed_at, config_version)
		 VALUES ($1, $2, $3, 'up', now(), 0)
		 ON CONFLICT (public_ip) DO UPDATE SET role = EXCLUDED.role, region = EXCLUDED.region
		 RETURNING id`, role, region, publicIP).Scan(&id)
	return id, err
}

// List возвращает все узлы (для панели и выбора exit).
func (f *Fleet) List(ctx context.Context) ([]Node, error) {
	rows, err := f.pool.Query(ctx,
		`SELECT id, role, region, host(public_ip), status, last_heartbeat, config_version
		 FROM nodes ORDER BY region, role`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.Role, &n.Region, &n.PublicIP, &n.Status, &n.LastHeartbeat, &n.ConfigVersion); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (f *Fleet) SetStatus(ctx context.Context, id, status string) error {
	_, err := f.pool.Exec(ctx, `UPDATE nodes SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (f *Fleet) Heartbeat(ctx context.Context, id string) error {
	_, err := f.pool.Exec(ctx, `UPDATE nodes SET last_heartbeat = now(), status = 'up' WHERE id = $1`, id)
	return err
}

// --- health-пробы (из orchestrator) ---

const dialTimeout = 5 * time.Second

// Probe проверяет один узел; nil = жив.
type Probe func(ctx context.Context, n Node) error

// TLSProbe: успешный TLS-handshake к PublicIP:port (egress Reality presentует серт
// сайта-донора — серт не верифицируем, важен сам факт ответа).
func TLSProbe(port int) Probe {
	return func(ctx context.Context, n Node) error {
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
	return func(ctx context.Context, n Node) error {
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

// CheckOnce — один проход по узлам: метит down после threshold подряд-неудач.
func (f *Fleet) CheckOnce(ctx context.Context, probes map[string]Probe, threshold int, fails map[string]int) error {
	nodes, err := f.List(ctx)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		probe, ok := probes[n.Role]
		if !ok {
			continue // нет пробы для роли — статус ведётся heartbeat'ом
		}
		if err := probe(ctx, n); err != nil {
			fails[n.ID]++
			if fails[n.ID] >= threshold {
				_ = f.SetStatus(ctx, n.ID, "down")
			}
		} else {
			fails[n.ID] = 0
			_ = f.SetStatus(ctx, n.ID, "up")
		}
	}
	return nil
}

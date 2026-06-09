// Package store — реестр узлов (таблица nodes) поверх PostgreSQL.
package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }

func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

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

// Register идемпотентно регистрирует узел по public_ip (upsert).
func (s *Store) Register(ctx context.Context, role, region, publicIP string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO nodes (role, region, public_ip, status, deployed_at, config_version)
		 VALUES ($1, $2, $3, 'up', now(), 0)
		 ON CONFLICT (public_ip) DO UPDATE SET role = EXCLUDED.role, region = EXCLUDED.region
		 RETURNING id`, role, region, publicIP).Scan(&id)
	return id, err
}

// List возвращает все узлы.
func (s *Store) List(ctx context.Context) ([]Node, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, role, region, host(public_ip), status, last_heartbeat, config_version
		 FROM nodes ORDER BY role, region`)
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

// SetStatus меняет статус узла.
func (s *Store) SetStatus(ctx context.Context, id, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE nodes SET status = $1 WHERE id = $2`, status, id)
	return err
}

// Heartbeat обновляет last_heartbeat и поднимает статус в up.
func (s *Store) Heartbeat(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE nodes SET last_heartbeat = now(), status = 'up' WHERE id = $1`, id)
	return err
}

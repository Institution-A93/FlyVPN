// Package store — доступ к PostgreSQL для account-api (аккаунты, сессии, подписки,
// квоты, устройства, платежи, рефералы). Общая БД с config-api/orchestrator.
package store

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/netip"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound — запрашиваемая запись отсутствует (мапится в 404/401 на HTTP-слое).
var ErrNotFound = errors.New("not found")

// Store держит пул соединений.
type Store struct{ pool *pgxpool.Pool }

// New открывает пул по DSN (pgx) и проверяет соединение.
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

// Close закрывает пул.
func (s *Store) Close() { s.pool.Close() }

// Ping проверяет доступность БД (для /healthz).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Пул sticky-IP: 10.8.0.0/14 => 10.8.0.0 .. 10.11.255.255 (как в config-api).
const framedBase = 0x0A080000 // 10.8.0.0
const framedSize = 1 << 18    // /14

func randFramedIP() string {
	off := rand.IntN(framedSize-1) + 1 // исключаем .0 (сеть)
	v := uint32(framedBase + off)
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}).String()
}

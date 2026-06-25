// Package store — общий пул PostgreSQL для control (pgx).
package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New открывает пул по DSN и проверяет соединение.
func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

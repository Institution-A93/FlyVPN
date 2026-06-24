package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateSession сохраняет hash refresh-токена. Возвращает session id.
func (s *Store) CreateSession(ctx context.Context, userID, refreshHash, userAgent, ip string, ttl time.Duration) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `INSERT INTO sessions
		(user_id, refresh_token_hash, user_agent, ip, expires_at)
		VALUES ($1,$2,$3, NULLIF($4,'')::inet, $5) RETURNING id`,
		userID, refreshHash, userAgent, ip, time.Now().UTC().Add(ttl)).Scan(&id)
	return id, err
}

// RotateSession атомарно проверяет старый refresh-hash (не отозван, не истёк), отзывает
// его и создаёт новый. Возвращает user_id и id новой сессии. Защита от повторного
// использования refresh-токена: старый hash инвалидируется.
func (s *Store) RotateSession(ctx context.Context, oldHash, newHash, userAgent, ip string, ttl time.Duration) (userID, newSessionID string, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	err = tx.QueryRow(ctx, `UPDATE sessions SET revoked_at = now()
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id`, oldHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", err
	}
	err = tx.QueryRow(ctx, `INSERT INTO sessions
		(user_id, refresh_token_hash, user_agent, ip, expires_at)
		VALUES ($1,$2,$3, NULLIF($4,'')::inet, $5) RETURNING id`,
		userID, newHash, userAgent, ip, time.Now().UTC().Add(ttl)).Scan(&newSessionID)
	if err != nil {
		return "", "", err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", "", err
	}
	return userID, newSessionID, nil
}

// RevokeSessionByHash отзывает сессию по refresh-hash (logout). Идемпотентно.
func (s *Store) RevokeSessionByHash(ctx context.Context, refreshHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at = now()
		WHERE refresh_token_hash = $1 AND revoked_at IS NULL`, refreshHash)
	return err
}

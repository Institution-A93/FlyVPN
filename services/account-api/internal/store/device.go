package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Device — выданный кред (= «устройство») для списка GET /devices.
type Device struct {
	ID         string
	Username   string
	FramedIP   string
	IssuedAt   time.Time
	LastUsedAt *time.Time
}

// IssueDevice создаёт новый стабильный кред для пользователя (decision #7: устройств
// без лимита; #11: креды не ротируются). Подбирает свободный framed_ip из пула.
// Возвращает id и framed_ip нового кредала.
func (s *Store) IssueDevice(ctx context.Context, userID, username, ntHash string) (deviceID, framedIP string, err error) {
	for attempt := 0; attempt < 8; attempt++ {
		ip := randFramedIP()
		err = s.pool.QueryRow(ctx, `INSERT INTO auth_credentials (user_id, username, nt_hash, framed_ip)
			VALUES ($1,$2,$3,$4) RETURNING id, host(framed_ip)`, userID, username, ntHash, ip).Scan(&deviceID, &framedIP)
		if err == nil {
			_, _ = s.pool.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action, metadata)
				VALUES ('user', $1, 'device_issued', jsonb_build_object('device_id', $2::text))`, userID, deviceID)
			return deviceID, framedIP, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_auth_credentials_framed_ip" {
			continue // коллизия IP — другой адрес
		}
		return "", "", fmt.Errorf("insert credential: %w", err)
	}
	return "", "", errors.New("не удалось подобрать свободный framed_ip за 8 попыток")
}

// ListDevices возвращает активные (не отозванные) креды пользователя.
func (s *Store) ListDevices(ctx context.Context, userID string) ([]Device, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, host(framed_ip), issued_at, last_used_at
		FROM auth_credentials WHERE user_id = $1 AND revoked_at IS NULL ORDER BY issued_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.Username, &d.FramedIP, &d.IssuedAt, &d.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RevokeDevice отзывает кред пользователя (revoked_at) — FreeRADIUS перестаёт его принимать.
// ErrNotFound, если кред не принадлежит юзеру или уже отозван.
func (s *Store) RevokeDevice(ctx context.Context, userID, deviceID string) error {
	ct, err := s.pool.Exec(ctx, `UPDATE auth_credentials SET revoked_at = now(), revoked_reason = 'user'
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`, deviceID, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" { // invalid uuid
			return ErrNotFound
		}
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, _ = s.pool.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action, metadata)
		VALUES ('user', $1, 'device_revoked', jsonb_build_object('device_id', $2::text))`, userID, deviceID)
	return nil
}

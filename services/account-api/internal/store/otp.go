package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// OTP-лимиты (docs/backend-requirements.md §10).
const (
	OTPMaxAttempts   = 5
	OTPPerPhonePerHr = 5
	OTPPerIPPerHr    = 20
)

// StoreOTP сохраняет hash кода. Перед этим проверяет rate-limit по телефону и IP.
func (s *Store) StoreOTP(ctx context.Context, phone, codeHash, ip string, ttl time.Duration) error {
	since := time.Now().UTC().Add(-time.Hour)

	var perPhone int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM otp_codes WHERE phone = $1 AND created_at > $2`, phone, since).Scan(&perPhone); err != nil {
		return err
	}
	if perPhone >= OTPPerPhonePerHr {
		return ErrRateLimited
	}
	if ip != "" {
		var perIP int
		if err := s.pool.QueryRow(ctx,
			`SELECT count(*) FROM otp_codes WHERE request_ip = $1::inet AND created_at > $2`, ip, since).Scan(&perIP); err != nil {
			return err
		}
		if perIP >= OTPPerIPPerHr {
			return ErrRateLimited
		}
	}

	_, err := s.pool.Exec(ctx, `INSERT INTO otp_codes (phone, code_hash, expires_at, request_ip)
		VALUES ($1, $2, $3, NULLIF($4,'')::inet)`, phone, codeHash, time.Now().UTC().Add(ttl), ip)
	return err
}

// ErrRateLimited — превышен лимит запросов OTP.
var ErrRateLimited = errors.New("rate limited")

// ErrOTPInvalid — код не совпал/истёк/исчерпаны попытки.
var ErrOTPInvalid = errors.New("otp invalid")

// VerifyOTP проверяет код для телефона: берёт самый свежий неиспользованный неистёкший,
// сверяет hash, инкрементит attempts, при совпадении гасит (consumed_at). Константного
// времени сравнение не требуется — code_hash уже хэш.
func (s *Store) VerifyOTP(ctx context.Context, phone, codeHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var id string
	var stored string
	var attempts int
	err = tx.QueryRow(ctx, `SELECT id, code_hash, attempts FROM otp_codes
		WHERE phone = $1 AND consumed_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, phone).Scan(&id, &stored, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOTPInvalid
	}
	if err != nil {
		return err
	}
	if attempts >= OTPMaxAttempts {
		return ErrOTPInvalid
	}
	if stored != codeHash {
		if _, err := tx.Exec(ctx, `UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1`, id); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return ErrOTPInvalid
	}
	if _, err := tx.Exec(ctx, `UPDATE otp_codes SET consumed_at = now() WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

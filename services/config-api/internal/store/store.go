// Package store — доступ к PostgreSQL (юзеры, подписки, EAP-креды).
package store

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store держит пул соединений.
type Store struct{ pool *pgxpool.Pool }

// New открывает пул по DSN (pgx).
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

// IssueParams — вход для Issue (один Plati-заказ).
type IssueParams struct {
	PlatiBuyerID    string
	Email           string
	PlatiOrderID    string
	Plan            string // '30d' | '90d' | '365d'
	AmountRub       int
	Duration        time.Duration
	CandidateUser   string // username для НОВОГО кредала (для существующего игнорируется)
	NTHash          string // NT-hash свежесгенерированного пароля
}

// IssueResult — что вернуть для сборки профиля.
type IssueResult struct {
	Username string
	FramedIP string
}

// pool 10.8.0.0/14 => 10.8.0.0 .. 10.11.255.255
const framedBase = 0x0A080000 // 10.8.0.0
const framedSize = 1 << 18    // /14

// Issue идемпотентно по plati_order_id: апсертит юзера, добавляет подписку,
// гарантирует EAP-кред. На повторную покупку (продление) username стабилен,
// nt_hash ротируется под свежий пароль (MMVP: профиль перевыпускается).
func (s *Store) Issue(ctx context.Context, p IssueParams) (IssueResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IssueResult{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID string
	err = tx.QueryRow(ctx,
		`INSERT INTO users (plati_buyer_id, email, status)
		 VALUES ($1, $2, 'active')
		 ON CONFLICT (plati_buyer_id) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`, p.PlatiBuyerID, p.Email).Scan(&userID)
	if err != nil {
		return IssueResult{}, fmt.Errorf("upsert user: %w", err)
	}

	now := time.Now().UTC()
	_, err = tx.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plati_order_id, plan, started_at, expires_at, amount_rub, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active')
		 ON CONFLICT (plati_order_id) DO NOTHING`,
		userID, p.PlatiOrderID, p.Plan, now, now.Add(p.Duration), p.AmountRub)
	if err != nil {
		return IssueResult{}, fmt.Errorf("insert subscription: %w", err)
	}

	// Существующий кред юзера?
	var username, framedIP string
	err = tx.QueryRow(ctx,
		`SELECT username, host(framed_ip) FROM auth_credentials WHERE user_id = $1`,
		userID).Scan(&username, &framedIP)
	switch {
	case err == nil:
		// Ротируем nt_hash под свежий пароль, снимаем возможный revoke.
		if _, err = tx.Exec(ctx,
			`UPDATE auth_credentials SET nt_hash = $1, revoked_at = NULL WHERE user_id = $2`,
			p.NTHash, userID); err != nil {
			return IssueResult{}, fmt.Errorf("rotate credential: %w", err)
		}
	case errors.Is(err, pgx.ErrNoRows):
		username = p.CandidateUser
		framedIP, err = insertCredentialWithIP(ctx, tx, userID, username, p.NTHash)
		if err != nil {
			return IssueResult{}, err
		}
	default:
		return IssueResult{}, fmt.Errorf("select credential: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return IssueResult{}, err
	}
	return IssueResult{Username: username, FramedIP: framedIP}, nil
}

// insertCredentialWithIP подбирает свободный sticky-IP из пула (повтор при коллизии).
func insertCredentialWithIP(ctx context.Context, tx pgx.Tx, userID, username, ntHash string) (string, error) {
	for attempt := 0; attempt < 8; attempt++ {
		ip := randFramedIP()
		_, err := tx.Exec(ctx,
			`INSERT INTO auth_credentials (user_id, username, nt_hash, framed_ip)
			 VALUES ($1, $2, $3, $4)`, userID, username, ntHash, ip)
		if err == nil {
			return ip, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// конфликт по framed_ip — пробуем другой IP; по username — это фатально
			if pgErr.ConstraintName == "idx_auth_credentials_framed_ip" {
				continue
			}
		}
		return "", fmt.Errorf("insert credential: %w", err)
	}
	return "", errors.New("не удалось подобрать свободный framed_ip за 8 попыток")
}

func randFramedIP() string {
	// исключаем .0 (сеть)
	off := rand.IntN(framedSize-1) + 1
	v := uint32(framedBase + off)
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}).String()
}

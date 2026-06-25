// Package contract — ЕДИНСТВЕННЫЙ писатель в живой entitlement (auth_credentials).
// И панель оператора, и платёжные коннекторы ходят только через него (ADR-0021).
// Идемпотентный CRUD: Provision/Renew/Revoke/Get. nt-hash СТАБИЛЬНЫЙ (на продлении
// не ротируется). Срок/кап пишутся на кред; класс маршрутизации (active/restricted
// пул) выбирает FreeRADIUS в authorize_reply по живому состоянию — здесь только данные.
package contract

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

	"github.com/institution-a93/flyvpn/services/control/internal/credentials"
)

// active-пул Framed-IP: 10.8.0.0/14. restricted-пул (10.12.0.0/14) — не здесь, его
// вычисляет FreeRADIUS reply для lapsed-юзеров (анти-локаут).
const (
	framedBase = 0x0A080000 // 10.8.0.0
	framedSize = 1 << 18     // /14
	ipAttempts = 8
)

// ErrNotFound — креда для юзера нет (мапится в 404 на HTTP-слое).
var ErrNotFound = errors.New("not found")

// Plan — длительность подписки и кап трафика за период.
type Plan struct {
	Duration time.Duration
	CapBytes int64 // 0 = безлимит (NULL в БД)
}

// Contract держит пул.
type Contract struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Contract { return &Contract{pool: pool} }

// Credential — состояние креда (для панели и выдачи профиля).
type Credential struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Username    string     `json:"username"`
	FramedIP    string     `json:"framed_ip"`
	ExpiresAt   *time.Time `json:"expires_at"`
	TrafficCap  *int64     `json:"traffic_cap_bytes"`
	PeriodStart time.Time  `json:"period_start"`
	RevokedAt   *time.Time `json:"revoked_at"`
}

// Provisioned — что вернуть для сборки профиля после Provision.
type Provisioned struct {
	Username string
	Password string // в БД хранится только nt_hash; пароль отдаётся ОДИН раз
	FramedIP string
}

// Provision создаёт новый кред для юзера: username/пароль + NT-hash + sticky-IP из
// active-пула + лимиты плана (expires_at, cap, period_start=now). nt-hash рождается
// здесь и больше не меняется. Пароль возвращается единожды (для профиля).
func (c *Contract) Provision(ctx context.Context, userID string, p Plan) (Provisioned, error) {
	username, err := credentials.GenerateUsername()
	if err != nil {
		return Provisioned{}, err
	}
	password, err := credentials.GeneratePassword()
	if err != nil {
		return Provisioned{}, err
	}
	ntHash := credentials.NTHash(password)
	expires := time.Now().UTC().Add(p.Duration)
	var cap *int64
	if p.CapBytes > 0 {
		cap = &p.CapBytes
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return Provisioned{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	framedIP, err := insertWithIP(ctx, tx, userID, username, ntHash, expires, cap)
	if err != nil {
		return Provisioned{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Provisioned{}, err
	}
	return Provisioned{Username: username, Password: password, FramedIP: framedIP}, nil
}

// Renew продлевает кред юзера: expires_at = max(now, expires_at) + длительность;
// period_start = now (счётчик трафика обнуляется — БЕЗ переноса); кап = план; revoke
// снимается. nt-hash НЕ трогаем. Если креда нет — ErrNotFound.
func (c *Contract) Renew(ctx context.Context, userID string, p Plan) error {
	var cap *int64
	if p.CapBytes > 0 {
		cap = &p.CapBytes
	}
	ct, err := c.pool.Exec(ctx, `
		UPDATE auth_credentials
		   SET expires_at = GREATEST(now(), COALESCE(expires_at, now())) + make_interval(secs => $2),
		       period_start = now(),
		       traffic_cap_bytes = $3,
		       revoked_at = NULL
		 WHERE user_id = $1`, userID, p.Duration.Seconds(), cap)
	if err != nil {
		return fmt.Errorf("renew: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Revoke жёстко отзывает кред (удаление аккаунта/фрод): revoked_at = now → FreeRADIUS
// перестаёт пускать вовсе (туннеля нет; это НЕ walled garden — то по сроку/капу).
func (c *Contract) Revoke(ctx context.Context, userID string) error {
	ct, err := c.pool.Exec(ctx,
		`UPDATE auth_credentials SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("revoke: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Get возвращает состояние креда юзера (для панели).
func (c *Contract) Get(ctx context.Context, userID string) (Credential, error) {
	var cr Credential
	err := c.pool.QueryRow(ctx, `
		SELECT id, user_id, username, host(framed_ip), expires_at, traffic_cap_bytes, period_start, revoked_at
		  FROM auth_credentials WHERE user_id = $1`, userID).
		Scan(&cr.ID, &cr.UserID, &cr.Username, &cr.FramedIP, &cr.ExpiresAt, &cr.TrafficCap, &cr.PeriodStart, &cr.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credential{}, ErrNotFound
	}
	if err != nil {
		return Credential{}, err
	}
	return cr, nil
}

// insertWithIP подбирает свободный sticky-IP из active-пула (повтор при коллизии).
func insertWithIP(ctx context.Context, tx pgx.Tx, userID, username, ntHash string, expires time.Time, cap *int64) (string, error) {
	for attempt := 0; attempt < ipAttempts; attempt++ {
		ip := randFramedIP()
		_, err := tx.Exec(ctx, `
			INSERT INTO auth_credentials (user_id, username, nt_hash, framed_ip, expires_at, traffic_cap_bytes, period_start)
			VALUES ($1, $2, $3, $4, $5, $6, now())`, userID, username, ntHash, ip, expires, cap)
		if err == nil {
			return ip, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_auth_credentials_framed_ip" {
			continue // коллизия по IP — другой; по username — фатально
		}
		return "", fmt.Errorf("insert credential: %w", err)
	}
	return "", errors.New("не удалось подобрать свободный framed_ip за 8 попыток")
}

func randFramedIP() string {
	off := rand.IntN(framedSize-1) + 1 // исключаем .0 (сеть)
	v := uint32(framedBase + off)
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}).String()
}

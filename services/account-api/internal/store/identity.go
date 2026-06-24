package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// User — представление аккаунта для /me.
type User struct {
	ID               string
	Email            string
	ReferralCode     string
	TelegramUsername string
	Phone            string
}

// TelegramProfile — данные из Telegram login-widget.
type TelegramProfile struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	PhotoURL   string
}

const referralCodeAlphabet = "abcdefghijkmnpqrstuvwxyz23456789" // без похожих символов

func newReferralCode() (string, error) {
	const n = 8
	b := make([]byte, n)
	maxN := big.NewInt(int64(len(referralCodeAlphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, maxN)
		if err != nil {
			return "", err
		}
		b[i] = referralCodeAlphabet[idx.Int64()]
	}
	return string(b), nil
}

// UpsertByTelegram находит пользователя по telegram_id или заводит нового (с триалом,
// реф-кодом, атрибуцией referred_by). Идемпотентно по telegram_id.
func (s *Store) UpsertByTelegram(ctx context.Context, p TelegramProfile, refCode string) (userID string, isNew bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	err = tx.QueryRow(ctx, `SELECT user_id FROM telegram_identities WHERE telegram_id = $1`, p.TelegramID).Scan(&userID)
	switch {
	case err == nil:
		// обновим отображаемые поля
		_, err = tx.Exec(ctx, `UPDATE telegram_identities
			SET username=$2, first_name=$3, last_name=$4, photo_url=$5 WHERE telegram_id=$1`,
			p.TelegramID, p.Username, p.FirstName, p.LastName, p.PhotoURL)
		if err != nil {
			return "", false, err
		}
	case errors.Is(err, pgx.ErrNoRows):
		userID, err = createUser(ctx, tx, refCode)
		if err != nil {
			return "", false, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO telegram_identities
			(user_id, telegram_id, username, first_name, last_name, photo_url)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			userID, p.TelegramID, p.Username, p.FirstName, p.LastName, p.PhotoURL); err != nil {
			return "", false, fmt.Errorf("insert telegram identity: %w", err)
		}
		isNew = true
	default:
		return "", false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return userID, isNew, nil
}

// UpsertByPhone находит пользователя по телефону или заводит нового. Идемпотентно по phone.
func (s *Store) UpsertByPhone(ctx context.Context, phone, refCode string) (userID string, isNew bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	err = tx.QueryRow(ctx, `SELECT user_id FROM phone_identities WHERE phone = $1`, phone).Scan(&userID)
	switch {
	case err == nil:
		_, err = tx.Exec(ctx, `UPDATE phone_identities SET verified_at = now() WHERE phone = $1`, phone)
		if err != nil {
			return "", false, err
		}
	case errors.Is(err, pgx.ErrNoRows):
		userID, err = createUser(ctx, tx, refCode)
		if err != nil {
			return "", false, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO phone_identities (user_id, phone, verified_at)
			VALUES ($1,$2, now())`, userID, phone); err != nil {
			return "", false, fmt.Errorf("insert phone identity: %w", err)
		}
		isNew = true
	default:
		return "", false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return userID, isNew, nil
}

// createUser создаёт users-строку (уникальный реф-код), проставляет referred_by по refCode,
// заводит реферал-связь (pending), создаёт триал-подписку и welcome-нотификацию.
func createUser(ctx context.Context, tx pgx.Tx, refCode string) (string, error) {
	// referred_by из ?ref= (если код существует и это не сам себя — проверим после insert).
	var referredBy *string
	if refCode != "" {
		var inviter string
		switch err := tx.QueryRow(ctx, `SELECT id FROM users WHERE referral_code = $1`, refCode).Scan(&inviter); {
		case err == nil:
			referredBy = &inviter
		case errors.Is(err, pgx.ErrNoRows):
			// неизвестный код — игнорируем атрибуцию
		default:
			return "", err
		}
	}

	var userID string
	for attempt := 0; attempt < 8; attempt++ {
		code, err := newReferralCode()
		if err != nil {
			return "", err
		}
		err = tx.QueryRow(ctx, `INSERT INTO users (status, referral_code, referred_by)
			VALUES ('active', $1, $2) RETURNING id`, code, referredBy).Scan(&userID)
		if err == nil {
			break
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_users_referral_code" {
			continue // коллизия реф-кода — пробуем другой
		}
		return "", fmt.Errorf("insert user: %w", err)
	}
	if userID == "" {
		return "", errors.New("не удалось подобрать уникальный referral_code")
	}

	if referredBy != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO referrals (inviter_user_id, invitee_user_id, status)
			VALUES ($1, $2, 'pending') ON CONFLICT (invitee_user_id) DO NOTHING`, *referredBy, userID); err != nil {
			return "", fmt.Errorf("insert referral: %w", err)
		}
	}

	// Триал: 300 МБ + 1 месяц (decision #3). plati_order_id = NULL (не Digiseller).
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `INSERT INTO subscriptions
		(user_id, plan, kind, started_at, expires_at, current_period_start, traffic_bytes_limit, amount_rub, status)
		VALUES ($1, '30d', 'trial', $2, $3, $2, $4, 0, 'active')`,
		userID, now, now.Add(config.TrialDuration), int64(config.TrialLimitBytes)); err != nil {
		return "", fmt.Errorf("insert trial subscription: %w", err)
	}

	// Welcome-нотификация (доставка ботом отложена — decision #14).
	if _, err := tx.Exec(ctx, `INSERT INTO notification_events (user_id, type, dedupe_key)
		VALUES ($1, 'welcome', $2) ON CONFLICT (dedupe_key) DO NOTHING`,
		userID, "welcome:"+userID); err != nil {
		return "", fmt.Errorf("insert welcome event: %w", err)
	}

	if _, err := tx.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action) VALUES ('system', $1, 'account_created')`, userID); err != nil {
		return "", err
	}
	return userID, nil
}

// GetMe возвращает данные аккаунта для /me.
func (s *Store) GetMe(ctx context.Context, userID string) (User, error) {
	u := User{ID: userID}
	var email, tgUsername, phone *string
	err := s.pool.QueryRow(ctx, `
		SELECT u.email, u.referral_code, t.username, p.phone
		FROM users u
		LEFT JOIN telegram_identities t ON t.user_id = u.id
		LEFT JOIN phone_identities   p ON p.user_id = u.id
		WHERE u.id = $1`, userID).Scan(&email, &u.ReferralCode, &tgUsername, &phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	u.Email = derefStr(email)
	u.TelegramUsername = derefStr(tgUsername)
	u.Phone = derefStr(phone)
	return u, nil
}

// SetEmail сохраняет billing-email (для чеков Platega, decision #10).
func (s *Store) SetEmail(ctx context.Context, userID, email string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET email = $2 WHERE id = $1`, userID, email)
	return err
}

// DeleteAccount отзывает все креды и сессии, удаляет идентичности и PII, но сохраняет
// subscriptions/amount_rub для учёта (decision: §5.3).
func (s *Store) DeleteAccount(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, q := range []string{
		`UPDATE auth_credentials SET revoked_at = now(), revoked_reason = 'account_deleted' WHERE user_id = $1 AND revoked_at IS NULL`,
		`UPDATE sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
		`DELETE FROM telegram_identities WHERE user_id = $1`,
		`DELETE FROM phone_identities WHERE user_id = $1`,
		`UPDATE users SET email = NULL, status = 'churned' WHERE id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action) VALUES ('user', $1, 'account_deleted')`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

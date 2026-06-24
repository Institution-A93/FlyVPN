package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Subscription — активная подписка пользователя.
type Subscription struct {
	ID                 string
	Plan               string
	Kind               string
	Active             bool
	ExpiresAt          time.Time
	CurrentPeriodStart time.Time
	TrafficBytesLimit  int64
}

// ActiveSubscription возвращает текущую действующую подписку (status='active' и не истёкшую),
// самую «дальнобойную» по expires_at. ErrNotFound, если такой нет.
func (s *Store) ActiveSubscription(ctx context.Context, userID string) (Subscription, error) {
	var sub Subscription
	err := s.pool.QueryRow(ctx, `
		SELECT id, plan, kind, expires_at, current_period_start, traffic_bytes_limit
		FROM subscriptions
		WHERE user_id = $1 AND status = 'active' AND expires_at > now()
		ORDER BY expires_at DESC LIMIT 1`, userID).
		Scan(&sub.ID, &sub.Plan, &sub.Kind, &sub.ExpiresAt, &sub.CurrentPeriodStart, &sub.TrafficBytesLimit)
	if errors.Is(err, pgx.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, err
	}
	sub.Active = true
	return sub, nil
}

// Traffic возвращает использованные за текущий период байты и лимит (квота + реф-бонус).
// usedBytes = Σ usage_log с current_period_start; limitBytes = traffic_bytes_limit + bonus_bytes.
func (s *Store) Traffic(ctx context.Context, userID string) (usedBytes, limitBytes int64, err error) {
	sub, err := s.ActiveSubscription(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	var bonus int64
	if err = s.pool.QueryRow(ctx, `SELECT bonus_bytes FROM users WHERE id = $1`, userID).Scan(&bonus); err != nil {
		return 0, 0, err
	}
	if err = s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(bytes_in + bytes_out), 0) FROM usage_log
		WHERE user_id = $1 AND date >= $2::date`, userID, sub.CurrentPeriodStart).Scan(&usedBytes); err != nil {
		return 0, 0, err
	}
	return usedBytes, sub.TrafficBytesLimit + bonus, nil
}

// ReferralStats — данные для GET /referral.
type ReferralStats struct {
	InvitedCount  int
	RewardedCount int
}

// Referral возвращает реф-код и статистику приглашений.
func (s *Store) Referral(ctx context.Context, userID string) (code string, stats ReferralStats, err error) {
	if err = s.pool.QueryRow(ctx, `SELECT referral_code FROM users WHERE id = $1`, userID).Scan(&code); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ReferralStats{}, ErrNotFound
		}
		return "", ReferralStats{}, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE true),
		       count(*) FILTER (WHERE status = 'rewarded')
		FROM referrals WHERE inviter_user_id = $1`, userID).
		Scan(&stats.InvitedCount, &stats.RewardedCount)
	if err != nil {
		return "", ReferralStats{}, err
	}
	return code, stats, nil
}

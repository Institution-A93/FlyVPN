package store

import (
	"context"
	"errors"
	"time"

	"github.com/institution-a93/flyvpn/services/account-api/internal/config"
	"github.com/jackc/pgx/v5"
)

// Payment — строка платежа.
type Payment struct {
	ID                string
	UserID            string
	PackageID         string
	AmountRub         int
	Method            string
	Status            string
	ProviderPaymentID string
}

// CreatePayment создаёт платёж (status=created). Идемпотентно по idempotency_key:
// при повторе возвращает существующий платёж (isNew=false).
func (s *Store) CreatePayment(ctx context.Context, userID, packageID string, amountRub int, method, idempotencyKey string) (p Payment, isNew bool, err error) {
	err = s.pool.QueryRow(ctx, `
		INSERT INTO payments (user_id, package_id, amount_rub, method, idempotency_key, status)
		VALUES ($1,$2,$3,$4,$5,'created')
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id, status`, userID, packageID, amountRub, method, idempotencyKey).Scan(&p.ID, &p.Status)
	if err == nil {
		p.UserID, p.PackageID, p.AmountRub, p.Method = userID, packageID, amountRub, method
		return p, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, false, err
	}
	// Конфликт по idempotency_key — отдаём существующий.
	err = s.pool.QueryRow(ctx, `SELECT id, user_id, package_id, amount_rub, COALESCE(method,''), status,
		COALESCE(provider_payment_id,'') FROM payments WHERE idempotency_key = $1`, idempotencyKey).
		Scan(&p.ID, &p.UserID, &p.PackageID, &p.AmountRub, &p.Method, &p.Status, &p.ProviderPaymentID)
	if err != nil {
		return Payment{}, false, err
	}
	return p, false, nil
}

// AttachProviderID сохраняет transactionId Platega и переводит в pending.
func (s *Store) AttachProviderID(ctx context.Context, paymentID, providerID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE payments
		SET provider_payment_id = $2, status = 'pending', updated_at = now()
		WHERE id = $1 AND status = 'created'`, paymentID, providerID)
	return err
}

// GetPayment читает платёж по id (= payload вебхука).
func (s *Store) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	var p Payment
	err := s.pool.QueryRow(ctx, `SELECT id, user_id, package_id, amount_rub, COALESCE(method,''), status,
		COALESCE(provider_payment_id,'') FROM payments WHERE id = $1`, paymentID).
		Scan(&p.ID, &p.UserID, &p.PackageID, &p.AmountRub, &p.Method, &p.Status, &p.ProviderPaymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payment{}, ErrNotFound
	}
	if err != nil {
		return Payment{}, err
	}
	return p, nil
}

// ConfirmPayment идемпотентно подтверждает платёж: переводит в succeeded, создаёт платную
// подписку (10 ГБ / +30 дней, свежий период) и улаживает реферал при первом успешном
// платеже приглашённого. Повторный вызов на уже-succeeded — no-op. Источник истины —
// только этот путь (вебхук Platega, ADR-0020).
func (s *Store) ConfirmPayment(ctx context.Context, paymentID, providerID string, rawPayload []byte) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID, status string
	var amountRub int
	err = tx.QueryRow(ctx, `SELECT user_id, status, amount_rub FROM payments WHERE id = $1 FOR UPDATE`, paymentID).
		Scan(&userID, &status, &amountRub)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == "succeeded" {
		return nil // идемпотентность: уже обработан
	}

	if _, err = tx.Exec(ctx, `UPDATE payments
		SET status = 'succeeded', provider_payment_id = COALESCE(NULLIF($2,''), provider_payment_id),
		    raw_payload = $3, updated_at = now() WHERE id = $1`, paymentID, providerID, rawPayload); err != nil {
		return err
	}

	// Платная подписка: новая строка (ActiveSubscription выберет самую дальнобойную;
	// current_period_start=now сбрасывает месячный счётчик квоты).
	now := time.Now().UTC()
	if _, err = tx.Exec(ctx, `INSERT INTO subscriptions
		(user_id, plan, kind, started_at, expires_at, current_period_start, traffic_bytes_limit, amount_rub, status)
		VALUES ($1,'30d','paid',$2,$3,$2,$4,$5,'active')`,
		userID, now, now.Add(config.PaidDuration), int64(config.PaidLimitBytes), amountRub); err != nil {
		return err
	}

	// Снимаем ТОЛЬКО quota/expiry-блокировку прошлых кредов (top-up — стабильные креды).
	// Удалённые аккаунтом / вручную отозванные устройства не воскрешаем (revoked_reason).
	if _, err = tx.Exec(ctx, `UPDATE auth_credentials SET revoked_at = NULL, revoked_reason = NULL
		WHERE user_id = $1 AND revoked_reason IN ('quota','expiry')`, userID); err != nil {
		return err
	}

	if err = settleReferral(ctx, tx, userID, paymentID); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action, metadata)
		VALUES ('system', $1, 'payment_succeeded', jsonb_build_object('payment_id', $2::text))`, userID, paymentID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// settleReferral начисляет рефереру бонус, если это ПЕРВЫЙ успешный платёж приглашённого.
func settleReferral(ctx context.Context, tx pgx.Tx, inviteeID, paymentID string) error {
	// Первый ли это успешный платёж? (текущий уже помечен succeeded — считаем > 1 => не первый)
	var succeededCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM payments WHERE user_id = $1 AND status = 'succeeded'`, inviteeID).
		Scan(&succeededCount); err != nil {
		return err
	}
	if succeededCount != 1 {
		return nil // не первый успешный платёж
	}

	var refID, inviterID string
	err := tx.QueryRow(ctx, `SELECT id, inviter_user_id FROM referrals
		WHERE invitee_user_id = $1 AND status <> 'rewarded' FOR UPDATE`, inviteeID).Scan(&refID, &inviterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // приглашённого никто не звал, либо уже награждён
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET bonus_bytes = bonus_bytes + $2 WHERE id = $1`,
		inviterID, int64(config.ReferralBonus)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE referrals
		SET status = 'rewarded', qualifying_payment_id = $2, qualified_at = now(), rewarded_at = now()
		WHERE id = $1`, refID, paymentID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action, metadata)
		VALUES ('system', $1, 'referral_rewarded', jsonb_build_object('invitee', $2::text))`, inviterID, inviteeID)
	return err
}

// MarkPayment переводит платёж в failed/refunded. На refunded — откатывает подписку и
// реферальный бонус, начисленный этим платежом. Идемпотентно.
func (s *Store) MarkPayment(ctx context.Context, paymentID, status string, rawPayload []byte) error {
	if status != "failed" && status != "refunded" {
		return errors.New("invalid terminal status")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var userID, cur string
	err = tx.QueryRow(ctx, `SELECT user_id, status FROM payments WHERE id = $1 FOR UPDATE`, paymentID).Scan(&userID, &cur)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if cur == status {
		return nil // идемпотентность
	}

	if _, err = tx.Exec(ctx, `UPDATE payments SET status = $2, raw_payload = $3, updated_at = now() WHERE id = $1`,
		paymentID, status, rawPayload); err != nil {
		return err
	}

	if status == "refunded" && cur == "succeeded" {
		// Откатываем реф-бонус, начисленный этим платежом.
		var inviterID *string
		err = tx.QueryRow(ctx, `UPDATE referrals SET status = 'qualified', rewarded_at = NULL
			WHERE qualifying_payment_id = $1 AND status = 'rewarded'
			RETURNING inviter_user_id`, paymentID).Scan(&inviterID)
		switch {
		case err == nil && inviterID != nil:
			if _, err = tx.Exec(ctx, `UPDATE users
				SET bonus_bytes = GREATEST(0, bonus_bytes - $2) WHERE id = $1`,
				*inviterID, int64(config.ReferralBonus)); err != nil {
				return err
			}
		case errors.Is(err, pgx.ErrNoRows):
			// этот платёж не давал бонуса
		default:
			return err
		}
	}

	_, err = tx.Exec(ctx, `INSERT INTO audit_log (actor, user_id, action, metadata)
		VALUES ('system', $1, 'payment_'||$2, jsonb_build_object('payment_id', $3::text))`, userID, status, paymentID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

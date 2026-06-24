package store

import "context"

// Cron-обязанности MVP над аккаунт-слоем (docs/backend-requirements.md §9, §10).
// Все операции — чистый SQL; источник истины по платежам/рефералам — account-api (вебхук).

// ExpireSubscriptions помечает истёкшие активные подписки expired. Возвращает число строк.
func (s *Store) ExpireSubscriptions(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx,
		`UPDATE subscriptions SET status = 'expired'
		 WHERE status = 'active' AND expires_at < now()`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// RevokeExpiredCreds отзывает (reason='expiry') активные креды пользователей, у которых
// больше нет ни одной действующей подписки. Не трогает уже отозванные (в т.ч. 'user'/'account_deleted').
func (s *Store) RevokeExpiredCreds(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE auth_credentials SET revoked_at = now(), revoked_reason = 'expiry'
		WHERE revoked_at IS NULL
		  AND user_id NOT IN (
		      SELECT user_id FROM subscriptions WHERE status = 'active' AND expires_at > now()
		  )`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// EnforceQuota отзывает (reason='quota') креды пользователей, чьё потребление за текущий
// период достигло лимита (квота активной подписки + bonus_bytes).
func (s *Store) EnforceQuota(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx, `
		WITH act AS (
		    SELECT DISTINCT ON (s.user_id) s.user_id, s.traffic_bytes_limit, s.current_period_start
		    FROM subscriptions s
		    WHERE s.status = 'active' AND s.expires_at > now()
		    ORDER BY s.user_id, s.expires_at DESC
		),
		usg AS (
		    SELECT a.user_id,
		           a.traffic_bytes_limit + u.bonus_bytes AS limit_bytes,
		           COALESCE((SELECT SUM(ul.bytes_in + ul.bytes_out) FROM usage_log ul
		                     WHERE ul.user_id = a.user_id AND ul.date >= a.current_period_start::date), 0) AS used
		    FROM act a JOIN users u ON u.id = a.user_id
		)
		UPDATE auth_credentials c SET revoked_at = now(), revoked_reason = 'quota'
		FROM usg WHERE c.user_id = usg.user_id AND c.revoked_at IS NULL AND usg.used >= usg.limit_bytes`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// RestoreUnderQuota повторно включает quota-отозванные креды пользователей, снова уложившихся
// в лимит при действующей подписке (после ролла периода). Только reason='quota'.
func (s *Store) RestoreUnderQuota(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx, `
		WITH act AS (
		    SELECT DISTINCT ON (s.user_id) s.user_id, s.traffic_bytes_limit, s.current_period_start
		    FROM subscriptions s
		    WHERE s.status = 'active' AND s.expires_at > now()
		    ORDER BY s.user_id, s.expires_at DESC
		),
		usg AS (
		    SELECT a.user_id,
		           a.traffic_bytes_limit + u.bonus_bytes AS limit_bytes,
		           COALESCE((SELECT SUM(ul.bytes_in + ul.bytes_out) FROM usage_log ul
		                     WHERE ul.user_id = a.user_id AND ul.date >= a.current_period_start::date), 0) AS used
		    FROM act a JOIN users u ON u.id = a.user_id
		)
		UPDATE auth_credentials c SET revoked_at = NULL, revoked_reason = NULL
		FROM usg WHERE c.user_id = usg.user_id AND c.revoked_reason = 'quota' AND usg.used < usg.limit_bytes`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// RollPeriods продвигает current_period_start на 30 дней для активных подписок, у которых
// месяц истёк (сбрасывает эффективный счётчик квоты). Один шаг; вызывать в цикле для догона.
func (s *Store) RollPeriods(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE subscriptions
		SET current_period_start = current_period_start + interval '30 days'
		WHERE status = 'active' AND expires_at > now()
		  AND current_period_start < now() - interval '30 days'`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// EnqueueThresholdEvents ставит в notification_events пороговые события (idempotent по dedupe_key):
// traffic_100mb / traffic_1gb (остаток квоты), sub_expiring (T-1д), sub_expired.
func (s *Store) EnqueueThresholdEvents(ctx context.Context) (int64, error) {
	var total int64

	// Остаток трафика: на основе активной подписки и потребления периода.
	for _, ev := range []struct {
		typ       string
		threshold int64
	}{
		{"traffic_100mb", 100 * 1024 * 1024},
		{"traffic_1gb", 1024 * 1024 * 1024},
	} {
		ct, err := s.pool.Exec(ctx, `
			WITH act AS (
			    SELECT DISTINCT ON (s.user_id) s.user_id, s.traffic_bytes_limit, s.current_period_start
			    FROM subscriptions s
			    WHERE s.status = 'active' AND s.expires_at > now()
			    ORDER BY s.user_id, s.expires_at DESC
			),
			usg AS (
			    SELECT a.user_id, a.current_period_start,
			           a.traffic_bytes_limit + u.bonus_bytes AS limit_bytes,
			           COALESCE((SELECT SUM(ul.bytes_in + ul.bytes_out) FROM usage_log ul
			                     WHERE ul.user_id = a.user_id AND ul.date >= a.current_period_start::date), 0) AS used
			    FROM act a JOIN users u ON u.id = a.user_id
			)
			INSERT INTO notification_events (user_id, type, payload, dedupe_key)
			SELECT user_id, $1,
			       jsonb_build_object('remaining_bytes', limit_bytes - used),
			       $1 || ':' || user_id::text || ':' || extract(epoch from current_period_start)::bigint::text
			FROM usg
			WHERE limit_bytes - used <= $2 AND used < limit_bytes
			ON CONFLICT (dedupe_key) DO NOTHING`, ev.typ, ev.threshold)
		if err != nil {
			return total, err
		}
		total += ct.RowsAffected()
	}

	// Подписка истекает в течение 24ч.
	ct, err := s.pool.Exec(ctx, `
		INSERT INTO notification_events (user_id, type, payload, dedupe_key)
		SELECT user_id, 'sub_expiring',
		       jsonb_build_object('expires_at', expires_at),
		       'sub_expiring:' || id::text
		FROM subscriptions
		WHERE status = 'active' AND expires_at > now() AND expires_at <= now() + interval '24 hours'
		ON CONFLICT (dedupe_key) DO NOTHING`)
	if err != nil {
		return total, err
	}
	total += ct.RowsAffected()

	// Подписка истекла.
	ct, err = s.pool.Exec(ctx, `
		INSERT INTO notification_events (user_id, type, dedupe_key)
		SELECT user_id, 'sub_expired', 'sub_expired:' || id::text
		FROM subscriptions WHERE status = 'expired'
		ON CONFLICT (dedupe_key) DO NOTHING`)
	if err != nil {
		return total, err
	}
	total += ct.RowsAffected()
	return total, nil
}

// CleanupOTP удаляет протухшие OTP-коды (старше часа от истечения).
func (s *Store) CleanupOTP(ctx context.Context) (int64, error) {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM otp_codes WHERE expires_at < now() - interval '1 hour'`)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// CountStalePendingPayments — число «зависших» pending-платежей (старше 30 мин). Реальный
// poll статуса в Platega — за account-api (там клиент); ADR-0020 «Открытые места».
func (s *Store) CountStalePendingPayments(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM payments WHERE status = 'pending' AND updated_at < now() - interval '30 minutes'`).Scan(&n)
	return n, err
}

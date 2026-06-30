-- MVP (веб-продукт): аккаунт-слой поверх MMVP-схемы (docs/backend-requirements.md §4.2).
-- Только ALTER существующих таблиц; новые таблицы — в 0004. Не ломаем 0001/0002:
-- новые NOT NULL-колонки добавляем с бэкфиллом существующих строк, затем фиксируем NOT NULL.

-- users: идентичность + рефералы. email уже есть (теперь — billing-почта Platega).
ALTER TABLE users ADD COLUMN referral_code TEXT;
ALTER TABLE users ADD COLUMN referred_by   UUID REFERENCES users(id);
ALTER TABLE users ADD COLUMN bonus_bytes   BIGINT NOT NULL DEFAULT 0; -- накопленный реф-бонус (§6)

-- referral_code: уникален среди заданных; NULL допускаем (legacy-строки до бэкфилла кодом сервиса).
CREATE UNIQUE INDEX idx_users_referral_code ON users (referral_code) WHERE referral_code IS NOT NULL;
CREATE INDEX idx_users_referred_by ON users (referred_by);

-- subscriptions: месячная traffic-quota (решения #4, #12). MVP — month-only.
-- Бэкфилл legacy MMVP-подписок (duration-based, без квоты): kind='paid',
-- current_period_start=started_at, лимит — практически безлимит (legacy без cap).
ALTER TABLE subscriptions ADD COLUMN traffic_bytes_limit  BIGINT;
ALTER TABLE subscriptions ADD COLUMN current_period_start TIMESTAMPTZ;
ALTER TABLE subscriptions ADD COLUMN kind                 TEXT;

UPDATE subscriptions
   SET traffic_bytes_limit  = COALESCE(traffic_bytes_limit, 9223372036854775807),
       current_period_start = COALESCE(current_period_start, started_at),
       kind                 = COALESCE(kind, 'paid')
 WHERE traffic_bytes_limit IS NULL
    OR current_period_start IS NULL
    OR kind IS NULL;

ALTER TABLE subscriptions ALTER COLUMN traffic_bytes_limit  SET NOT NULL;
ALTER TABLE subscriptions ALTER COLUMN current_period_start SET NOT NULL;
ALTER TABLE subscriptions ALTER COLUMN kind                 SET NOT NULL;
ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_kind_check CHECK (kind IN ('trial', 'paid'));

-- MVP-подписки (trial / Platega-paid) не имеют Digiseller-кода: ослабляем NOT NULL.
-- UNIQUE сохраняется (в Postgres допускает множество NULL) — идемпотентность Digiseller
-- по plati_order_id не страдает. Связь Platega-платежа с подпиской — через payments (0004).
ALTER TABLE subscriptions ALTER COLUMN plati_order_id DROP NOT NULL;

-- plan: MVP канонизирует месячный план как '30d' (CHECK 0001 уже его допускает).
-- Старые значения 90d/365d остаются валидными для legacy-строк MMVP — CHECK не трогаем,
-- чтобы не ломать 0001 (см. §4.2: "keep 30d").

CREATE INDEX idx_subscriptions_period ON subscriptions (current_period_start);

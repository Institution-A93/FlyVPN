-- Необратимо по данным: 0007 — forward-cleanup растворённого account-слоя (ADR-0021).
-- Откат восстанавливает только КОЛОНКИ users (структурно); таблицы account-слоя
-- (subscriptions/usage_log/identities/billing) при необходимости поднимаются заново
-- из 0001/0003/0004 — но это противоречит цели пивота. Здесь — минимальный шов.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS referral_code TEXT,
    ADD COLUMN IF NOT EXISTS referred_by   UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS bonus_bytes   BIGINT NOT NULL DEFAULT 0;

DROP INDEX IF EXISTS idx_subscriptions_period;
-- Восстанавливаем NOT NULL на plati_order_id (откат возможен лишь если NULL-строк нет).
ALTER TABLE subscriptions ALTER COLUMN plati_order_id SET NOT NULL;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_kind_check;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS kind;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS current_period_start;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS traffic_bytes_limit;

DROP INDEX IF EXISTS idx_users_referred_by;
DROP INDEX IF EXISTS idx_users_referral_code;
ALTER TABLE users DROP COLUMN IF EXISTS bonus_bytes;
ALTER TABLE users DROP COLUMN IF EXISTS referred_by;
ALTER TABLE users DROP COLUMN IF EXISTS referral_code;

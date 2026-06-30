-- ADR-0021: снос мёртвого account-слоя. Он растворился в бинаре control + нативном
-- FreeRADIUS. Источник истины для entitlement — auth_credentials (срок/кап/period_start,
-- 0006) + radacct (учёт). Самописная Go-плоскость (account-api) и её таблицы сняты.
-- Оставляем: users (FK auth_credentials), auth_credentials, nodes/node_secrets, radacct.

-- 0004 (identities/billing/notify) — целиком:
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS notification_events;
DROP TABLE IF EXISTS referrals;            -- FK → payments
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS phone_identities;
DROP TABLE IF EXISTS telegram_identities;

-- 0001 bespoke entitlement → заменено auth_credentials + radacct:
DROP TABLE IF EXISTS usage_log;
DROP TABLE IF EXISTS subscriptions;

-- 0003 реф-колонки на users (рефералка вне MVP):
ALTER TABLE users
    DROP COLUMN IF EXISTS referred_by,
    DROP COLUMN IF EXISTS referral_code,
    DROP COLUMN IF EXISTS bonus_bytes;

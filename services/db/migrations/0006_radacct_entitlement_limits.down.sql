-- Откат 0006.
DROP INDEX IF EXISTS idx_auth_credentials_expires;
ALTER TABLE auth_credentials
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS traffic_cap_bytes,
    DROP COLUMN IF EXISTS period_start;
DROP TABLE IF EXISTS radacct;

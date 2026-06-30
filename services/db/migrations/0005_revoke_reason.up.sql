-- Причина отзыва кредала: нужна оркестратору, чтобы при ролле периода / top-up повторно
-- включать ТОЛЬКО креды, заблокированные по квоте/истечению, и не воскрешать удалённые
-- аккаунтом или вручную отозванные устройства (docs/backend-requirements.md §8, §9).
ALTER TABLE auth_credentials ADD COLUMN revoked_reason TEXT
    CHECK (revoked_reason IN ('user', 'expiry', 'quota', 'account_deleted'));

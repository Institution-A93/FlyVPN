-- MVP: новые таблицы аккаунт-слоя (docs/backend-requirements.md §4.3).
-- Идентичности, сессии, OTP, платежи Platega (ADR-0020), рефералы, нотификации, аудит.

-- Telegram-идентичность (основная display-идентичность кабинета).
CREATE TABLE telegram_identities (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    telegram_id BIGINT NOT NULL UNIQUE,
    username    TEXT,
    first_name  TEXT,
    last_name   TEXT,
    photo_url   TEXT,
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id)
);
CREATE UNIQUE INDEX idx_telegram_identities_tg ON telegram_identities (telegram_id);

-- Телефонная идентичность (E.164).
CREATE TABLE phone_identities (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone       TEXT NOT NULL UNIQUE,
    verified_at TIMESTAMPTZ,
    PRIMARY KEY (user_id)
);
CREATE UNIQUE INDEX idx_phone_identities_phone ON phone_identities (phone);

-- OTP для телефонного логина (Twilio). Храним только хэш кода, не plaintext.
CREATE TABLE otp_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone      TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    request_ip INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_otp_codes_phone ON otp_codes (phone);
CREATE INDEX idx_otp_codes_expires ON otp_codes (expires_at);

-- Сессии: access — stateless JWT; refresh — здесь (ротация, отзыв).
CREATE TABLE sessions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    user_agent         TEXT,
    ip                 INET,
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user ON sessions (user_id);

-- Платежи Platega.io (ADR-0020). Права меняются ТОЛЬКО по подтверждённому вебхуку.
-- Идемпотентность: idempotency_key (наш) + provider_payment_id (провайдера).
CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_id          TEXT NOT NULL,
    provider            TEXT NOT NULL DEFAULT 'platega' CHECK (provider IN ('platega')),
    provider_payment_id TEXT UNIQUE,           -- transactionId Platega; NULL до создания транзакции
    amount_rub          INTEGER NOT NULL,
    method              TEXT CHECK (method IN ('sbp', 'card', 'intl_card')),
    status              TEXT NOT NULL DEFAULT 'created'
                        CHECK (status IN ('created', 'pending', 'succeeded', 'failed', 'refunded')),
    idempotency_key     TEXT UNIQUE,
    raw_payload         JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_payments_user ON payments (user_id);
CREATE INDEX idx_payments_status ON payments (status);

-- Рефералы: квалификация — первый успешный платёж приглашённого; награда +1 ГБ рефереру.
CREATE TABLE referrals (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inviter_user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invitee_user_id      UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    status               TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'qualified', 'rewarded')),
    qualifying_payment_id UUID REFERENCES payments(id),
    inviter_reward_bytes  BIGINT NOT NULL DEFAULT 1073741824, -- 1 ГБ
    invitee_reward_bytes  BIGINT NOT NULL DEFAULT 0,          -- задел под будущие скидки приглашённому
    qualified_at         TIMESTAMPTZ,
    rewarded_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_referrals_inviter ON referrals (inviter_user_id);

-- Нотификации (потребляет бот — доставка отложена, decision #14). dedupe_key — fire-once.
CREATE TABLE notification_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT NOT NULL
               CHECK (type IN ('welcome', 'traffic_100mb', 'traffic_1gb', 'sub_expiring', 'sub_expired')),
    payload    JSONB,
    status     TEXT NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending', 'sent', 'failed')),
    dedupe_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at    TIMESTAMPTZ
);
CREATE INDEX idx_notification_events_status ON notification_events (status);

-- Аудит: логины, выдача/отзыв устройств, платежи, удаления.
CREATE TABLE audit_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor      TEXT NOT NULL CHECK (actor IN ('user', 'system')),
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    action     TEXT NOT NULL,
    metadata   JSONB,
    ip         INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_log_user ON audit_log (user_id);
CREATE INDEX idx_audit_log_created ON audit_log (created_at);

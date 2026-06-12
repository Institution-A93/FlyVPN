-- MMVP: исходная схема control plane (README §4).
-- Runner-agnostic plain SQL; язык сервисов не привязан (ADR-0013).

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- Пользователи (идентификация по plati_buyer_id, не по username/email).
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plati_buyer_id TEXT UNIQUE,
    email          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    status         TEXT NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active', 'blocked', 'churned'))
);

-- Подписки. plati_order_id (=uniquecode Digiseller) уникален (идемпотентность выдачи).
CREATE TABLE subscriptions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plati_order_id TEXT NOT NULL UNIQUE,
    plan           TEXT NOT NULL CHECK (plan IN ('30d', '90d', '365d')),
    started_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ NOT NULL,
    amount_rub     INTEGER NOT NULL,
    status         TEXT NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active', 'expired', 'refunded'))
);
CREATE INDEX idx_subscriptions_user ON subscriptions (user_id);
CREATE INDEX idx_subscriptions_expires ON subscriptions (expires_at);

-- EAP-креды. username — случайная строка; nt_hash = MD4(UTF-16LE(пароль)),
-- hex, 32 симв. (NT-Password для FreeRADIUS/MSCHAPv2; bcrypt несовместим — ADR-0014).
CREATE TABLE auth_credentials (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username     TEXT NOT NULL UNIQUE,
    nt_hash      TEXT NOT NULL CHECK (nt_hash ~ '^[0-9a-fA-F]{32}$'),
    framed_ip    INET NOT NULL,            -- sticky IP из пула 10.8.0.0/14
    issued_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at    TIMESTAMPTZ,             -- NULL = активны
    last_used_at  TIMESTAMPTZ
);
CREATE INDEX idx_auth_credentials_user ON auth_credentials (user_id);
CREATE UNIQUE INDEX idx_auth_credentials_framed_ip ON auth_credentials (framed_ip);

-- Агрегированный учёт трафика по дням.
CREATE TABLE usage_log (
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date      DATE NOT NULL,
    bytes_in  BIGINT NOT NULL DEFAULT 0,
    bytes_out BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);

-- Реестр узлов (egress/ingress/control).
CREATE TABLE nodes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role           TEXT NOT NULL CHECK (role IN ('ingress', 'egress', 'control')),
    region         TEXT NOT NULL,           -- 'ru-msk', 'nl-ams', ...
    public_ip      INET NOT NULL,
    status         TEXT NOT NULL DEFAULT 'up'
                   CHECK (status IN ('up', 'down', 'draining', 'maintenance')),
    last_heartbeat TIMESTAMPTZ,
    deployed_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    config_version INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_nodes_role_status ON nodes (role, status);

-- Секреты узлов (значение зашифровано master-ключом вне БД).
CREATE TABLE node_secrets (
    node_id      UUID NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    secret_type  TEXT NOT NULL
                 CHECK (secret_type IN ('reality_private_key', 'fde_unlock_key', 'tls_cert')),
    secret_value TEXT NOT NULL,
    issued_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ,
    PRIMARY KEY (node_id, secret_type)
);

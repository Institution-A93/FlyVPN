-- ADR-0021: нативный движок entitlement на FreeRADIUS.
-- Добавляет приёмник учёта (radacct) и лимиты прямо на кред (срок + кап трафика),
-- чтобы FreeRADIUS гасил по времени (expiration) и по объёму (sqlcounter) нативно,
-- без cron-плоскости. usage_log/subscriptions — растворяются отдельным шагом.

-- Стандартная FreeRADIUS-таблица учёта (PostgreSQL schema). Сюда strongSwan через
-- eap-radius accounting шлёт Acct-Start/Interim/Stop; кумулятив на сессию дедупится
-- по AcctUniqueId (ON CONFLICT). username = auth_credentials.username.
CREATE TABLE radacct (
    RadAcctId           bigserial PRIMARY KEY,
    AcctSessionId       text NOT NULL,
    AcctUniqueId        text NOT NULL UNIQUE,
    UserName            text,
    Realm               text,
    NASIPAddress        inet NOT NULL,
    NASPortId           text,
    NASPortType         text,
    AcctStartTime       timestamp with time zone,
    AcctUpdateTime      timestamp with time zone,
    AcctStopTime        timestamp with time zone,
    AcctInterval        bigint,
    AcctSessionTime     bigint,
    AcctAuthentic       text,
    ConnectInfo_start   text,
    ConnectInfo_stop    text,
    AcctInputOctets     bigint,
    AcctOutputOctets    bigint,
    CalledStationId     text,
    CallingStationId    text,
    AcctTerminateCause  text,
    ServiceType         text,
    FramedProtocol      text,
    FramedIPAddress     inet,
    Class               text
);
-- открытые сессии (для interim UPDATE и для CoA/bulk-close)
CREATE INDEX radacct_active_session_idx ON radacct (AcctUniqueId) WHERE AcctStopTime IS NULL;
CREATE INDEX radacct_bulk_close ON radacct (NASIPAddress, AcctStartTime) WHERE AcctStopTime IS NULL;
-- быстрый SUM(октеты) по юзеру с момента period_start (sqlcounter)
CREATE INDEX radacct_start_user_idx ON radacct (UserName, AcctStartTime);

-- Лимиты подписки переносятся НА кред (источник истины для FreeRADIUS).
-- Подписка = срок (expires_at) + кап за период (traffic_cap_bytes), БЕЗ переноса:
-- на продлении контракт двигает expires_at и period_start (счётчик radacct
-- эффективно обнуляется, т.к. считается с period_start).
ALTER TABLE auth_credentials
    ADD COLUMN expires_at        TIMESTAMPTZ,             -- срок действия креда (время)
    ADD COLUMN traffic_cap_bytes BIGINT,                 -- кап трафика за период (NULL = безлимит)
    ADD COLUMN period_start      TIMESTAMPTZ NOT NULL DEFAULT now(); -- начало текущего периода (= последнее продление)

-- FreeRADIUS authorize фильтрует ещё и по сроку; индекс под выборку активных.
CREATE INDEX idx_auth_credentials_expires ON auth_credentials (expires_at);

COMMENT ON COLUMN auth_credentials.expires_at IS 'срок действия (ADR-0021); authorize отвергает при now() > expires_at';
COMMENT ON COLUMN auth_credentials.traffic_cap_bytes IS 'кап трафика за период; sqlcounter сравнивает с SUM(radacct) c period_start';
COMMENT ON COLUMN auth_credentials.period_start IS 'начало текущего периода; продление двигает его → счётчик обнуляется (без переноса)';

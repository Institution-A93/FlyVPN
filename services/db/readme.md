# services/db

Схема данных control plane (PostgreSQL) как plain-SQL миграции. Языко-независима:
раннер не привязан к языку backend-сервисов (см. ADR-0013) — `.sql`-файлы совместимы с
golang-migrate, dbmate, Flyway, либо применяются напрямую через `psql`.

## Структура
- `migrations/NNNN_<name>.up.sql` — применение.
- `migrations/NNNN_<name>.down.sql` — откат.

## Схема (README §4)
MMVP (0001–0002): `users`, `subscriptions`, `auth_credentials`, `usage_log`, `nodes`, `node_secrets`.

Идентификация юзера — по `plati_buyer_id`, не по username/email. `auth_credentials.nt_hash` —
NT-hash (MD4) для MSCHAPv2, не bcrypt (ADR-0014). `node_secrets.secret_value` хранится
зашифрованным master-ключом (вне БД).

## MVP-слой (0003–0004, docs/backend-requirements.md §4)
- `0003_accounts_quota` — ALTER: `users` (+`referral_code`, `referred_by`, `bonus_bytes`);
  `subscriptions` (+`traffic_bytes_limit`, `current_period_start`, `kind`=trial/paid) — месячная
  traffic-quota. Не ломает 0001/0002: новые NOT NULL-колонки бэкфилятся на существующих строках
  (legacy MMVP-подписки → `kind=paid`, лимит ≈безлимит). `plan` CHECK не трогаем — MVP канонизирует
  месяц как `30d` (уже допустим), legacy `90d/365d` остаются валидны.
- `0004_identities_billing` — новые таблицы: `telegram_identities`, `phone_identities`, `otp_codes`,
  `sessions`, `payments` (Platega, ADR-0020), `referrals`, `notification_events`, `audit_log`.

## Проверка
Применяется на PostgreSQL 16 (проверено: 0001 создаёт 6 таблиц с FK/индексами/CHECK,
0002 добавляет UNIQUE на `nodes.public_ip`; 0003–0004 доводят до 14 таблиц, бэкфилл legacy-строк
проходит, down-миграции откатывают ровно к исходным 6 таблицам). Локально:
```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_init.up.sql
```

## Открытое
Финальный выбор раннера миграций — вместе с языком сервисов (ADR-0013).

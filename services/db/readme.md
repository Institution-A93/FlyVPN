# services/db

Схема данных control plane (PostgreSQL) как plain-SQL миграции. Языко-независима:
раннер не привязан к языку backend-сервисов (см. ADR-0013) — `.sql`-файлы совместимы с
golang-migrate, dbmate, Flyway, либо применяются напрямую через `psql`.

## Структура
- `migrations/NNNN_<name>.up.sql` — применение.
- `migrations/NNNN_<name>.down.sql` — откат.

## Схема (README §4)
`users`, `subscriptions`, `auth_credentials`, `usage_log`, `nodes`, `node_secrets`.

Идентификация юзера — по `plati_buyer_id`, не по username/email. `password_hash` — только
bcrypt. `node_secrets.secret_value` хранится зашифрованным master-ключом (вне БД).

## Проверка
Применяется на PostgreSQL 16 (проверено: up создаёт 6 таблиц с FK/индексами/CHECK,
down полностью откатывает). Локально:
```sh
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_init.up.sql
```

## Открытое
Финальный выбор раннера миграций — вместе с языком сервисов (ADR-0013).

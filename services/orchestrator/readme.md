# orchestrator

Go-сервис (ADR-0013). Жизненный цикл узлов и состояние сети.

Реализовано (MMVP-срез):
1. **Реестр узлов** в PostgreSQL (`nodes`): идемпотентная регистрация по `public_ip`,
   список, смена статуса, heartbeat.
2. **Health-checking**: периодические активные пробы; узел → `down` после
   `ORCH_HEALTH_THRESHOLD` подряд-неудач, обратно `up` при успехе. egress и control —
   TLS-проба на 443 (Reality презентует серт донора — нам важен факт TLS-ответа).
3. **HTTP/admin API**: `GET /healthz`, `GET /nodes`, `POST /nodes` (register).
4. **Cron-обязанности аккаунт-слоя MVP** (`internal/cron`, docs/backend-requirements.md §9–10):
   - **sweep** (часто, `ORCH_CRON_SWEEP_INTERVAL`): истечение подписок → `expired`;
     отзыв кредов без действующей подписки (`reason='expiry'`); enforcement квоты
     (`reason='quota'`) и обратное включение при возврате под лимит; пороговые
     `notification_events` (`traffic_100mb/1gb`, `sub_expiring`, `sub_expired`) с dedupe.
   - **housekeeping** (`ORCH_CRON_HOUSE_INTERVAL`): ролл месячного периода
     (`current_period_start += 30d`, с догоном), уборка протухших `otp_codes`,
     сигнал о «зависших» `pending`-платежах.
   - Урегулирование рефералов и смена прав по платежу — на стороне `account-api` (вебхук
     Platega, ADR-0020): единый источник истины, оркестратор его не дублирует.
   - `revoked_reason` (миграция 0005) позволяет включать обратно только quota/expiry-блокировки,
     не воскрешая удалённые аккаунтом / вручную отозванные устройства.

Не входит / TODO:
- **ingress** активной пробы не имеет на MMVP (IKEv2/UDP) — статус ведётся heartbeat'ом;
  глубокая IKE_SA_INIT-проба — отдельная задача.
- Выдача узлам секретов (Reality-ключи и т.п.) при старте — phase 2 (сейчас секреты из
  vault, ADR-0012).
- GeoDNS-update, авто-ротация секретов, FDE unlock, alerting, провижн через OpenTofu.

## Структура (Go)
- `cmd/orchestrator` — точка входа (config → pgx → health-loop + cron + HTTP).
- `internal/config` — конфиг из окружения.
- `internal/store` — реестр `nodes` + lifecycle аккаунт-слоя (pgx).
- `internal/health` — пробы (TLS/TCP) + Checker (порог неудач).
- `internal/cron` — периодические обязанности аккаунт-слоя MVP (sweep + housekeeping).
- `internal/httpapi` — admin API.

## Конфигурация (env)
`ORCH_DATABASE_URL` (обяз.), `ORCH_LISTEN` (`:9090`), `ORCH_HEALTH_INTERVAL` (`30s`),
`ORCH_HEALTH_THRESHOLD` (`3`), `ORCH_CRON_SWEEP_INTERVAL` (`2m`), `ORCH_CRON_HOUSE_INTERVAL` (`1h`).

## Проверено
`go build/vet/test ./...` зелёные. Health: TLS-проба (живой/закрытый сервер) и логика
порога. Store: интеграционные тесты на PostgreSQL 16 (идемпотентная регистрация, статус,
heartbeat; cron-lifecycle: quota-отзыв/возврат, истечение+отзыв, пороговые события с dedupe);
включаются `ORCH_TEST_DSN`. Схема — `../db` (миграции 0001–0005).

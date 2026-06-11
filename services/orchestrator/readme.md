# orchestrator

Go-сервис (ADR-0013). Жизненный цикл узлов и состояние сети.

Реализовано (MMVP-срез):
1. **Реестр узлов** в PostgreSQL (`nodes`): идемпотентная регистрация по `public_ip`,
   список, смена статуса, heartbeat.
2. **Health-checking**: периодические активные пробы; узел → `down` после
   `ORCH_HEALTH_THRESHOLD` подряд-неудач, обратно `up` при успехе. egress и control —
   TLS-проба на 443 (Reality презентует серт донора — нам важен факт TLS-ответа).
3. **HTTP/admin API**: `GET /healthz`, `GET /nodes`, `POST /nodes` (register).

Не входит / TODO:
- **ingress** активной пробы не имеет на MMVP (IKEv2/UDP) — статус ведётся heartbeat'ом;
  глубокая IKE_SA_INIT-проба — отдельная задача.
- Выдача узлам секретов (Reality-ключи и т.п.) при старте — phase 2 (сейчас секреты из
  vault, ADR-0012).
- GeoDNS-update, авто-ротация секретов, FDE unlock, alerting, провижн через OpenTofu.

## Структура (Go)
- `cmd/orchestrator` — точка входа (config → pgx → health-loop + HTTP).
- `internal/config` — конфиг из окружения.
- `internal/store` — реестр `nodes` (pgx).
- `internal/health` — пробы (TLS/TCP) + Checker (порог неудач).
- `internal/httpapi` — admin API.

## Конфигурация (env)
`ORCH_DATABASE_URL` (обяз.), `ORCH_LISTEN` (`:9090`), `ORCH_HEALTH_INTERVAL` (`30s`),
`ORCH_HEALTH_THRESHOLD` (`3`).

## Проверено
`go build/vet/test ./...` зелёные. Health: TLS-проба (живой/закрытый сервер) и логика
порога. Store: интеграционный тест на PostgreSQL 16 (идемпотентная регистрация, статус,
heartbeat); включается `ORCH_TEST_DSN`. Схема — `../db` (миграции 0001 + 0002).

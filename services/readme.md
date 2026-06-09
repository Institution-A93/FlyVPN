# services

Backend-сервисы control plane. Самописное минимизируем — это тонкий слой бизнес-логики
поверх готовых компонентов (FreeRADIUS, Xray, strongSwan).

- [`db/`](./db/readme.md) — схема PostgreSQL как plain-SQL миграции (языко-независимо).
- [`config-api/`](./config-api/readme.md) — приём Plati-вебхука, генерация .mobileconfig, выдача кредов.
- [`orchestrator/`](./orchestrator/readme.md) — реестр узлов, health-check, ротация секретов/узлов.

Язык сервисов не зафиксирован (ADR-0013). Деплоятся на control plane (см. ansible-роль `control-plane`).

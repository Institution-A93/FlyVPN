# services

Backend-сервисы control plane. Самописное минимизируем — это тонкий слой бизнес-логики
поверх готовых компонентов (FreeRADIUS, sing-box, strongSwan).

- [`db/`](./db/readme.md) — схема PostgreSQL как plain-SQL миграции (языко-независимо).
- [`config-api/`](./config-api/readme.md) — выдача по коду Plati/Digiseller, генерация .mobileconfig, выдача кредов.
- [`orchestrator/`](./orchestrator/readme.md) — реестр узлов, health-check, ротация секретов/узлов.
- [`bot/`](./bot/readme.md) — Telegram-бот: онбординг, выдача профиля, уведомления (Python/aiogram).

Язык сервисов — Go (ADR-0013); исключение — `bot` на Python/aiogram (ADR-0019).
Деплоятся на control plane (роли `control-plane` и `bot`).

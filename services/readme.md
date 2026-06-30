# services

Backend-сервисы control plane. Самописное минимизируем — тонкий слой бизнес-логики
поверх готовых компонентов (FreeRADIUS, sing-box, strongSwan).

- [`db/`](./db/readme.md) — схема PostgreSQL как plain-SQL миграции (языко-независимо).
- [`control/`](./control/readme.md) — **единый Go-бинарь** (ADR-0021, модульный монолит):
  `contract` (единственный писатель `auth_credentials`), `profiles` (.mobileconfig/.sswan),
  `fleet` (реестр узлов/health/CoA), `panel` (HTML для нетех-суппорта), webhook-коннекторы.
  Поглотил прежние config-api и orchestrator; account-api — растворён.
- [`bot/`](./bot/readme.md) — Telegram-бот: онбординг, доставка профиля, уведомления
  (Python/aiogram, ADR-0019); зовёт API `control`.

Язык сервисов — Go (ADR-0013); исключение — `bot` на Python/aiogram. Доступ к VPN —
нативный FreeRADIUS на PostgreSQL (срок/кап/учёт), не отдельный сервис (ADR-0021).

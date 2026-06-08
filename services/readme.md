# services

Go-сервисы control plane. Самописное минимизируем — это тонкий слой бизнес-логики
поверх готовых компонентов (FreeRADIUS, Xray, strongSwan).

- [`config-api/`](./config-api/readme.md) — приём Plati-вебхука, генерация .mobileconfig, выдача кредов.
- [`orchestrator/`](./orchestrator/readme.md) — реестр узлов, health-check, ротация секретов/узлов.

Язык — Go. Деплоятся на control plane (см. ansible-роль `control-plane`).

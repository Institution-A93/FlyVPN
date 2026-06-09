# role: control-plane

Конфигурирует foreign control plane:
- FreeRADIUS — серверная сторона EAP-аутентификации
- PostgreSQL — основная БД (юзеры, подписки, креды, узлы)
- config-api — сервис (Plati-вебхук, генерация .mobileconfig)
- orchestrator — сервис (реестр узлов, health-check, ротация)

Реализация (tasks/templates) — TODO.

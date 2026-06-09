# orchestrator

Сервис. Жизненный цикл узлов и состояние сети.

Ответственность (MMVP-срез):
1. Реестр узлов в PostgreSQL (роль, регион, статус, last_heartbeat, config_version).
2. Health-checking ingress/egress/mesh; пометка `down` при подряд-неудачах.
3. Выдача узлам секретов при старте (Reality-креды и т.п.) — на узлах не персистятся.
4. Ротация: триггер пересоздания узла через OpenTofu (на MMVP — запуск вручную).

Вне MMVP: GeoDNS-update, авто-ротация секретов, FDE unlock service, alerting.

Реализация (main.go, go.mod, workers) — TODO.

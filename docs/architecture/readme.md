# Архитектурные схемы

Визуальная карта проекта. Рендер — Graphviz (`dot -Tpng -Gdpi=150 target.dot -o target.png`).

| Файл | Что |
|------|-----|
| [current.png](./current.png) (`current.dot`) | **Что построено сейчас** (MMVP): data plane + config-api/account-api/orchestrator + FreeRADIUS/PostgreSQL (своя `auth_credentials`). Зелёное — остаётся, красное — растворяется. |
| [target.png](./target.png) (`target.dot`) | **Целевая архитектура MVP**: один Go-бинарь `control` (модульный монолит) + нативный FreeRADIUS/PostgreSQL + симметричные узлы. |

Решения за схемами — в `docs/adr/` (0021 — операторская поверхность/движок, 0022 — клиентская доставка, 0023 — симметричные узлы).

## Целевая в двух словах

- **Один Go-бинарь `control`** = пакеты `panel` (ручной RADIUS-aware CRUD юзеров + узлы), `contract` (CRUD `create/read/update/delete` — единственный писатель), `profiles` (`.mobileconfig`/`.sswan`), `fleet` (health-пробы + реестр + автопровизия). `config-api` и `orchestrator` растворяются в него пакетами.
- **Движок доступа** — нативный FreeRADIUS на PostgreSQL (своя `auth_credentials` + `radacct` + `sqlcounter` + `Expiration`).
- **Биллинг вне MVP** — платёжные коннекторы (webhook-хендлеры) за швом `renew()`.
- **Инфра без отдельного стека** — оркестрация и health внутри `control`; Prometheus/Grafana отложены (B1).
- Другой процесс только один — Telegram-бот (Python, параллельный трек).

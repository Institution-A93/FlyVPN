# Спайк: нативный движок entitlement (срок + кап + учёт) на FreeRADIUS

**Дата:** 2026-06-25 · **Питает:** ADR-0021, реализацию `sql.j2` · **Вердикт:** ✅

## Что проверяли

Боевой `infra/ansible/roles/control-plane/templates/sql.j2` (не canonical radcheck, а
наша `auth_credentials`): что FreeRADIUS гасит доступ **по всем трём осям сразу** —
отзыв, срок, кап — прямо в `authorize_check_query`, и пишет учёт в `radacct`. Без
отдельных модулей `expiration`/`sqlcounter`.

## Стенд

PostgreSQL 16 + FreeRADIUS 3.2.5. Схема — наши миграции **0001 + 0006** (даёт
`auth_credentials` + `expires_at`/`traffic_cap_bytes`/`period_start` + `radacct`).
Юзер `alice`: NT-hash, срок +30д, кап 100 МБ, period_start = now.

## Результаты

| Тест | Ожидание | Факт |
|------|----------|------|
| EAP-MSCHAPv2 в пределах лимитов (path strongSwan'а) | SUCCESS | ✅ `method 26 (MSCHAPV2)` → SUCCESS |
| MS-CHAP в пределах лимитов | Accept | ✅ Access-Accept |
| Accounting Start/Interim → `radacct` | строка с кумулятивом | ✅ `in=52428800 out=10485760`, interim не задвоился |
| Расход > кап → авторизация | Reject | ✅ Access-Reject (и MS-CHAP, и EAP) |
| Срок истёк → авторизация | Reject | ✅ Access-Reject |
| Восстановить срок/обнулить расход → авторизация | Accept | ✅ Access-Accept |

Механизм: `authorize_check_query` отдаёт `NT-Password` только если
`revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now()) AND
(cap IS NULL OR cap > SUM(radacct с period_start))`. Нарушено любое — нет
NT-Password — Access-Reject. Продление двигает `expires_at` + `period_start` →
счётчик `radacct` эффективно обнуляется (без переноса).

## Граница

- Это срабатывает на **(ре)коннекте**. Обрыв ПОСРЕДИ сессии при превышении —
  отдельно через **CoA/Disconnect** (`dae`), задача P0 #12.
- Остаётся обвязка роли (не логика): **acct-листенер (1813)** и вызов `sql` в секции
  `accounting{}` сайта FreeRADIUS — control-plane роль сейчас сайты не темплейтит;
  добавить дроп-ин листенер. Логика `sql.j2` — проверена.

## Воспроизведение

`apt-get install postgresql freeradius freeradius-postgresql eapoltest`; миграции
0001+0006; рендер `sql.j2` (подставить `cp_*`, убрать `{% raw %}`); acct-листенер с
`virtual_server = default`; `radtest -t mschap` / `eapol_test` / `radclient acct`.

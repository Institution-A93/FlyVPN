# Спайк: нативный движок entitlement (анти-локаут, два пула) на FreeRADIUS

**Дата:** 2026-06-25 · **Питает:** ADR-0021, `sql.j2` · **Вердикт:** ✅

## Модель (анти-локаут)

Доступ к Telegram/оплате должен оставаться **всегда** — даже если подписка истекла
или трафик исчерпан. Поэтому срок/кап **НЕ режут аутентификацию** (иначе lapsed-юзер
не поднимет туннель и не дойдёт до бота). Три состояния:

| Состояние | Auth | Framed-IP пул | Маршрутизация (ingress sing-box) |
|---|---|---|---|
| **active** (в сроке И в капе) | Accept | `10.8.0.0/14` | полный доступ |
| **lapsed** (истёк ИЛИ кап) | **Accept** | `10.12.0.0/14` | **только allowlist** (Telegram + DNS, позже оплата) |
| **revoked** (удалён/фрод) | Reject | — | туннеля нет |

Механизм в `sql.j2`:
- `authorize_check_query` — отдаёт `NT-Password` любому **не-`revoked`** креду (срок/кап
  не режут).
- `authorize_reply_query` — выбирает пул Framed-IP по живому состоянию: active → sticky
  `framed_ip` (10.8/14); lapsed → тот же хост в `10.12.0.0/14` (`framed_ip + (10.12.0.0
  − 10.8.0.0)`).
- Ingress sing-box маршрутизирует `10.12.0.0/14` только в allowlist, остальное —
  reject-route.

## Стенд

PostgreSQL 16 + FreeRADIUS 3.2.5; миграции **0001 + 0006**; юзер `alice`
(framed_ip 10.8.0.5, кап 100 МБ).

## Результаты

| Тест | Ожидание | Факт |
|------|----------|------|
| active | Accept + `10.8.0.5` | ✅ |
| **expired → туннель поднят** | **Accept** + `10.12.0.5` | ✅ |
| **over-cap → туннель поднят** | **Accept** + `10.12.0.5` | ✅ |
| revoked | Reject | ✅ |
| EAP-MSCHAPv2 (path strongSwan) в лимитах | SUCCESS | ✅ (method 26) |
| accounting Start/Interim → `radacct` | кумулятив, interim-дедуп | ✅ |

## Граница

- Пул выбирается на **(ре)коннекте**. Если юзер исчерпал кап **посреди** активной
  сессии (был в 10.8) — нужно его «пересадить» в restricted: **CoA/Disconnect** (P0
  #12) рвёт сессию → always-on клиент переподключается → reply отдаёт уже 10.12 →
  walled garden. Локаута нет (реконнект успешен).
- Остаётся data-plane половина: ingress sing-box — правило restricted-пула → allowlist
  + сам allowlist (Telegram-CIDR + DNS). Логика `sql.j2` — проверена.

## Воспроизведение

Рендер `sql.j2` (подставить `cp_*`, убрать `{% raw %}`); миграции 0001+0006;
`radtest -t mschap` показывает `Framed-IP-Address` в Access-Accept → видно пул.

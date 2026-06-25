# ADR-0021: Плоскость entitlement — нативный FreeRADIUS (PostgreSQL) + единый Go-бинарь `control`

- Статус: proposed
- Дата: 2026-06-25
- Связи: amends ADR-0020 · переутверждает ADR-0005, ADR-0014, ADR-0013 · усиливает
  инвариант «самописное минимизируем» (ADR-0009)

## Контекст

MVP-разработка свернула к самописной Go-плоскости entitlement (`account-api` + cron
оркестратора + bespoke `usage_log`/`subscriptions`). Перебрали готовые платформы
(OpenWISP, SHM, ABillS/Ubilling/Freeside/CGRateS, daloRADIUS, NocoDB) — все либо
машинерия под бизнес крупнее нашего, либо половина задачи. Для команды из трёх
человек (один DevSecOps, один нетех-суппорт, один продукт) задача тривиальна:

> Подписка = лимит по времени + лимит по МБ, **без переходящего остатка**; оплата →
> услуга продлевается.

**Коммерческий учёт — вне MVP** (его делают самописные коннекторы). В скоупе —
**технический учёт**. Среда, которой уступаем, — **FreeRADIUS + PostgreSQL**.

Два уточнения от инвентаризации Go и масштаба «один человек»:
- **Канонический `radcheck` не нужен.** Он требовался только под чужие панели. Раз
  мы пишем свою панель — оставляем **уже построенную схему `auth_credentials`**
  (прошита в `sql.j2`, провижится `config-api`, работает по ADR-0005).
- **Один бинарь, не три.** `config-api` (есть `Issue()` + профили), `orchestrator`
  (health + реестр) и новая панель — это **модульный монолит**, не три сервиса.

## Решение

1. **Движок** — нативный FreeRADIUS на PostgreSQL, **своя `auth_credentials`**
   (не canonical radcheck) + `radacct` (учёт). EAP-MSCHAPv2 из `NT-Password`
   (ADR-0014). Срок (`expires_at`) и кап (`traffic_cap_bytes` vs `SUM(radacct)` с
   `period_start`) — **прямо в SQL-запросах authorize**, без отдельных модулей
   `expiration`/`sqlcounter`. Подписка = срок + кап, без переноса; продление =
   сдвинуть `expires_at` + `period_start`. Проверено спайком (PG16 + FR 3.2.5).
1a. **АНТИ-ЛОКАУТ (walled garden).** Доступ к Telegram/оплате остаётся **всегда** —
   даже у истёкшего/исчерпавшего трафик. Срок/кап **НЕ режут auth**: туннель
   поднимается у любого не-`revoked` креда. Три состояния: **active** → Framed-IP из
   пула `10.8.0.0/14`, полный доступ; **lapsed** (истёк ИЛИ кап) → Framed-IP из
   `10.12.0.0/14`, ingress sing-box пускает только **allowlist** (Telegram+DNS, позже
   оплата); **revoked** (удалён/фрод) → auth reject. Класс выбирается в
   `authorize_reply_query` по живому состоянию. Превышение **посреди** сессии →
   CoA/Disconnect → реконнект пересаживает в restricted-пул (локаута нет).
2. **Операторская поверхность — ОДИН Go-бинарь `control`** (модульный монолит,
   ADR-0013). Один pgx-пул, один HTTP-сервер (внутренний), одна фоновая горутина
   health, операторская авторизация (JWT — из `account-api`). Пакеты:
   - **`panel`** — **ручной RADIUS-aware CRUD юзеров** (формы create/edit/delete/
     search, дропдауны план/регион, ГБ, дата) + обзор узлов (health). Не сырые
     строки — поэтому свой Go, а не NocoDB.
   - **`contract`** — идемпотентный **CRUD `create/read/update/delete`** над
     `auth_credentials` — **ЕДИНСТВЕННЫЙ писатель**. И панель, и коннекторы пишут
     только через него. (= обобщённый `config-api.Issue()`.)
   - **`profiles`** — `.mobileconfig` + `.sswan` (из `config-api`, + `.sswan`).
   - **`fleet`** — health-пробы (TLS/TCP) + реестр узлов + автопровизия (из
     `orchestrator`).
3. **Коммерческий учёт — вне MVP** → платёжные коннекторы (webhook-хендлеры **в том
   же бинаре**) зовут `contract`. Добавить/сменить способ оплаты = правка только
   коннектора, живая инфра не трогается.
4. **Без отдельного observability-стека и пейджера.** Флот виден в **той же панели**
   (`fleet`); оркестрация/health — внутри `control`. Prometheus/Grafana/Alertmanager —
   **отложено (B1)**, добавляется поверх когда вырастет флот. Telegram-бот — только
   доставка профилей (не пейджер).
5. **IaC vs рантайм-state:** IaC владеет инфрой и конфигом узлов; рантайм-данные
   (`auth_credentials`/`radacct`/`nodes`) — в PostgreSQL через `control`. Конфликта
   нет.

## Проверка (спайк, 2026-06-24)

PostgreSQL 16.13 + FreeRADIUS 3.2.5 (`docs/research/spike-openwisp-eap-mschapv2.md`):
EAP-MSCHAPv2/MS-CHAP из одного NT-hash; accounting `Start/Interim/Stop` →`radacct` с
дедупом interim'ов; `SUM` = логика `sqlcounter`. Механизм тот же для `auth_credentials`
(NT-Password). CoA вживую не тестировался.

## Зафиксированные продуктовые решения

- Рефералка / бонус-пул — вне MVP.
- Telegram-вход — координация с параллельным треком бота.
- Ретенция `radacct` (decision #10) — упростить/отложить.
- Лицевой счёт / коммерческий учёт — вне MVP (коннекторы).
- Панель — **ручной CRUD** (не read-only) через `contract`.
- **NT-hash — стабильный**: при продлении кред не меняется (не ротируется). Это
  снимает конфликт с прежней логикой `config-api.Issue()`, которая ротировала
  `nt_hash` на каждую покупку — в `contract` ротации нет.

**Отвергнуто** (`docs/research/billing-and-bss-options.md`,
`docs/research/radius-admin-panels.md`): OpenWISP (тяжёлый), daloRADIUS (панель без
API + MariaDB), NocoDB/Baserow (сырые строки, не RADIUS-aware), SHM/ABillS/Ubilling/
Freeside/CGRateS (биллинг-бизнес), канонический `radcheck` (не нужен — своя панель).

## Последствия

- **Растворяется**: `account-api` (auth/сессии/JWT/entitlements/devices) — но
  **донор каркаса** для `control` (HTTP, store, credentials.NTHash, JWT-auth).
  `usage_log`, bespoke `subscriptions`, cron-квоты оркестратора. Миграции `0003`–
  `0005` — superseded.
- **Сливаются в `control` пакетами**: `config-api` (`Issue()`/профили) и
  `orchestrator` (health/реестр/автопровизия).
- **Добавляется**: `radacct` + `sqlcounter` + `Expiration` + CoA/`dae` (FreeRADIUS/
  strongSwan); бинарь `control`; платёжные коннекторы; операторская авторизация.
- **Переутверждается**: ADR-0005/0014/0013. **Amends ADR-0020**: Platega — один из
  коннекторов за `contract`, не модель подписок.

## Открытые места

- `sqlcounter` под подписку: кап + сброс на **продление** (не календарный месяц).
- CoA — живой тест на развёртывании.
- Окно ретенции `radacct`.
- Публичная vs внутренняя поверхность `control`: в лин-MVP доставку опосредует бот →
  все эндпоинты внутренние; публичный URL выдачи профиля (если понадобится) — отдельный
  листенер в том же бинаре.

После живого CoA-теста и первого end-to-end (коннектор → `contract` →
`auth_credentials` → коннект) — перевести в `accepted`.

# ADR-0021: Плоскость entitlement — уступание RADIUS-native + панель OpenWISP

- Статус: proposed
- Дата: 2026-06-24
- Связи: amends ADR-0020 · переутверждает ADR-0005, ADR-0014 · усиливает инвариант
  «самописное минимизируем» (ADR-0009 / корневой `claude.md`)

## Контекст

MVP-разработка по `backend-requirements` (draft v3) и ADR-0020 свернула к
**самописной Go-плоскости** entitlement: `account-api` (учётки, сессии, подписки,
квота, устройства) + cron-обязанности оркестратора (расход, помесячный сброс,
пороги, реф-расчёт) + bespoke-таблицы `usage_log`/`subscriptions`. При ревизии
выяснилось:

- Это дублирует то, что RADIUS-стек делает нативно: учёт (`radacct`), месячная
  data-квота (`rlm_sqlcounter` / OpenWISP `MonthlyTrafficCounter`), отрезание
  (CoA/Disconnect, RFC 5176 + strongSwan `dae`).
- Прямо противоречит инварианту проекта *«самописное минимизируем: strongSwan,
  FreeRADIUS, sing-box — готовые компоненты»*.
- strongSwan уже шлёт per-user accounting (`eap-radius { accounting = yes }`), но
  принимающая сторона ничего с ним не делала — учёта де-факто не было.
- sing-box для per-user учёта непригоден (агрегатный, per-node); per-user даёт
  именно strongSwan/RADIUS.

## Решение

**Уступаем плоскость entitlement RADIUS-среде целиком (максимальный вариант шва),
держим форму только там, где среда слепа или враждебна нашим инвариантам.**

1. **Источник истины — каноническая FreeRADIUS SQL-схема на PostgreSQL 16**:
   `radcheck`/`radreply` (креды + лимиты), `radacct` (учёт), `radusergroup`/`radgroup*`
   (тарифы как группы). EAP-MSCHAPv2 — через `rlm_sql` из `NT-Password` (ADR-0014
   сохраняется, cleartext в БД нет).
2. **Квота и сброс** — нативный счётчик (`MonthlyTrafficCounter`), не cron.
3. **Отрезание** — **CoA/Disconnect** (strongSwan `dae`), а не только `revoked_at`.
   `revoked_at`/исключение из authorize-запроса остаётся «замком на следующий
   коннект», DM — «обрыв сейчас» (нужен для always-on iOS IKEv2).
4. **Операторская панель — OpenWISP RADIUS** (Django, PostgreSQL-native, REST API,
   CoA, месячные счётчики, телефон/SMS-верификация, само-регистрация). Выбор над
   daloRADIUS (PostgreSQL не для прода; нет REST API) и RADIUSdesk (своя модель,
   MySQL/mesh-ДНК). Обоснование — `docs/research/radius-admin-panels.md`.
5. **Приватность (decision #10).** `radacct` по природе хранит метаданные сессий
   (выданный IP, тайминги, NAS) — то, что инвариант запрещает держать. Берём
   механизм, но **подрезаем память**: окно ретенции сырого `radacct` + агрегация,
   только на control-plane, **никогда на RU-ingress**.
6. **Периферия — тонкие переводчики, не параллельная плоскость.** Telegram-вход,
   Platega, `.mobileconfig`, рефералка пишут в нативную схему / через OpenWISP REST
   API, а не ведут свою модель entitlement.

## Проверка (спайк, 2026-06-24)

На боевых версиях (**PostgreSQL 16.13 + FreeRADIUS 3.2.5**) воспроизведено
(`docs/research/spike-openwisp-eap-mschapv2.md`):

- ✅ **EAP-MSCHAPv2 и MS-CHAP из одного NT-hash** в `radcheck` через `rlm_sql` →
  Access-Accept / EAP-Success; неверный пароль → reject. Точная модель того, что
  релеит strongSwan (`eapol_test`).
- ✅ **Учёт**: `Start/Interim/Stop` → `radacct`, interim-update'ы **схлопнуты**
  (`ON CONFLICT (AcctUniqueId) DO UPDATE`, без двойного счёта), `SUM` за период =
  логика `MonthlyTrafficCounter`.
- ⚠️ **CoA не тестировался вживую** (нет NAS-приёмника в спайке) — заявлен OpenWISP
  v1.1.0 + strongSwan `dae`; проверка переносится на этап развёртывания.

Рискованное звено (EAP-MSCHAPv2 через `rlm_sql` на PostgreSQL) — снято.

## Последствия

**Растворяется** (из ранее написанного MVP-кода):

- `usage_log`, bespoke `subscriptions`/quota-модель → каноническая FR-схема +
  нативные счётчики. Миграции `0003`–`0005` — superseded.
- Оркестратор: cron расхода/помесячного сброса/порогов/реф-расчёта → нативный
  счётчик + CoA. У оркестратора остаётся реестр узлов, health, ротация.
- `account-api`: auth/сессии/entitlements/devices → OpenWISP (панель + REST API);
  остаётся тонкий **переводчик** Platega→entitlement и генерация `.mobileconfig`
  (кандидат на слияние с `config-api`).

**Добавляется**:

- OpenWISP на control-plane: Ansible-роль, зажатая по безопасности (приватная сеть,
  SSO/VPN, TLS, патчи).
- FreeRADIUS: accounting-запросы → `radacct`, нативный месячный счётчик, листенер CoA.
- strongSwan: `dae{}` для приёма Disconnect/CoA.

**Риски**: footprint OpenWISP (Django+Celery+Redis) — больше частей в проде;
приватность vs `radacct` (см. п.5) — глубина истории в панели подрезана осознанно.

## Открытые места

- Telegram-вход: есть ли пригодный django-allauth-провайдер, или это остаётся нашим
  тонким куском.
- Реф-бонус-пул («1 ГБ, не сбрасывается, переносится») в модель счётчика ложится
  криво — возможно упростить продукт под то, что среда выражает нативно (разовый
  bump лимита вместо вечного пула).
- Окно ретенции `radacct` под decision #10 — конкретные сроки/прунинг.
- Точный раздел `rlm_sql` (горячий путь EAP) vs `rlm_rest` (управление) в проде.
- Куда уезжает `.mobileconfig`-генерация: отдельный переводчик или `config-api`.

После проверки CoA на развёртывании и пилота OpenWISP — перевести в `accepted`.

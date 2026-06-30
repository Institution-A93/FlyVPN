# Состояние проекта smart-internet (FlyVPN) — снимок 2026-06-30

Снимок собран по факту из git/CI, а не по памяти. Описывает: что на гите, что в работе,
что сломано/протухло, открытые задачи и бэклог. Для следующего, кто продолжит.

---

## 1. TL;DR (одним абзацем)

Бэкенд переведён с тяжёлой самописной account-плоскости на минимальный MVP:
**нативный FreeRADIUS на PostgreSQL** (своя схема `auth_credentials` + `radacct`) и
**один Go-бинарь `control`** (модульный монолит). Движок (анти-локаут с двумя пулами,
срок/кап, accounting, revoke) **проверён сквозняком в CI** на живом FreeRADIUS+PG.
Вся работа лежит в ветке `claude/kind-pasteur-2IMEz`, оформлена как **PR #17 → `main`,
все 4 проверки зелёные, НО ещё не смержен**. `main` — на состоянии «до пивота».
Остались: мердж + аппрув прод-деплоя (за человеком), нодовая валидация на живых узлах,
переподключение бота к `control`, чистка протухших доков ansible-роли, перевод ADR из
proposed в accepted.

---

## 2. Что сейчас на гите

| | `main` (`3df91f9`) | ветка `claude/kind-pasteur-2IMEz` (`95d964f`) |
|---|---|---|
| Состояние | **до пивота** (последний мердж — PR #15, бот) | MVP-пивот, готов к мерджу |
| Go-сервисы | `config-api`, `orchestrator` | **`control`** (config-api/orchestrator/account-api снесены) |
| Бот | `services/bot` (mock) | `services/bot` (mock, **без изменений**) |
| Миграции БД | 0001–0002 | 0001–0007 |
| Движок entitlement | старый (config-api + NT-hash) | **FreeRADIUS-native** (`sql.j2`) |

- Ветка: **32 коммита впереди** `main` и 14 «позади» (эти 14 — мердж-коммиты наших же
  прошлых PR #1–#15; их содержимое уже в ветке, реального расхождения нет).
- **PR #17** (`claude/kind-pasteur-2IMEz` → `main`) — **открыт**, проверки:

  | Check | Workflow | Статус |
  |---|---|---|
  | `go` | test | ✅ |
  | `python` | test | ✅ |
  | `control-db` | integration (Go-тесты contract/fleet vs Postgres 16) | ✅ |
  | `engine-e2e` | integration (FreeRADIUS+PG, движок сквозняком) | ✅ |

  `engine-e2e` напечатал `ALL ENGINE SCENARIOS PASSED`: active→`10.8`, EAP-MSCHAPv2
  SUCCESS, expired→`10.12`, over-cap→`10.12`, accounting→`radacct` (дедуп interim),
  revoke→Reject.

> **Уклад проекта (важно для следующего):** одна dev-ветка `claude/kind-pasteur-2IMEz`
> переиспользуется → PR в `main` → CI (`test`/`plan`/`integration`) → мердж. Так сделано
> 15 раз. CD: `push` в нужные пути запускает `deploy`/`deploy-ingress` под гейтом
> `environment: production` (ручной аппрув). `pull_request` берёт определение workflow
> из ветки PR (потому новый `integration.yml` гоняется уже на своём PR).

---

## 3. История (как пришли к текущему состоянию)

1. **Account-layer (отменён).** Сначала строили самописный backend: `account-api`
   (accounts/auth/quota/devices/Platega billing), `orchestrator` (cron: expiry, quota,
   referral, period-roll), миграции 0003–0005, ADR-0020 (Platega). Всё это жило на ветке.
2. **Разворот к «уступанию среде».** Поняли, что задача тривиальна, а платформа (OpenWISP/
   SHM/ABillS/Marzban) — оверкилл. Решение: нативный FreeRADIUS + тонкий Go-сервис.
   Зафиксировано в **ADR-0021** (FreeRADIUS-native + единый бинарь `control`).
3. **Уточнение модели.** Подписка = лимит по времени + по мегабайтам, ничего не
   переносится; оплата продлевает; способов оплат много и они меняются → не лезть в живую
   инфру, всё через шов `renew()`. Коммерческий учёт — вне MVP (самописные коннекторы).
4. **Анти-локаут (ключевое требование).** Доступ к Telegram (и будущим платёжным сервисам)
   обязан оставаться **всегда** — даже если подписка истекла или кончился трафик. Решено
   **двумя пулами Framed-IP**: active `10.8.0.0/14` (полный доступ) и restricted
   `10.12.0.0/14` (walled garden — только allowlist Telegram + DNS). Auth **никогда не
   reject'ит** по сроку/капу; reject только при `revoked` (удаление/фрод).
5. **Симметрия узлов (ADR-0023).** Диаспора за рубежом должна доставать РФ-ресурсы через
   РФ-выход — тот же VPN наоборот. Параметризовали ingress (`ingress_self_ruleset`).
6. **Реализация.** Снесли account-слой (`af36ccf`), собрали `control` (contract/profiles/
   fleet/panel/httpapi), движок в `sql.j2`, миграции 0006/0007, walled-garden в sing-box,
   CoA/DAE в strongSwan.
7. **CI движка.** Раньше «движок проверяется только на узле». Раннер GitHub Actions — это
   полноценная VM → написали `integration.yml` (`control-db` + `engine-e2e`). Прогон вскрыл
   и починил цепочку проблем (см. §6), включая **реальный деплой-баг** (дубль acct-листенера
   на 1813, который положил бы живой узел).

---

## 4. Архитектура MVP (что построено в ветке)

- **Плоскость доступа:** нативный FreeRADIUS на PostgreSQL. Своя схема `auth_credentials`
  (НЕ canonical radcheck) + `radacct`. Срок и кап считаются прямо в `authorize_*_query`
  (без отдельных `rlm_sqlcounter`/`expiration`). `accounting{}` пишет `radacct`.
- **Анти-локаут:** `authorize_check_query` отдаёт NT-Password при `revoked_at IS NULL`
  (срок/кап НЕ блокируют auth); `authorize_reply_query` выбирает пул Framed-IP по живому
  состоянию (active→`10.8`, lapsed/over-cap→`10.12`). Живой обрыв сессии — CoA/DAE (`3799`).
- **`control`** (один Go-бинарь, модульный монолит):
  `contract` (единственный писатель в `auth_credentials`: Provision/Renew/Revoke/Usage/
  State; nt-hash стабилен при продлении) · `profiles` (.mobileconfig/.sswan) · `fleet`
  (реестр узлов/health/origination CoA) · `panel` (HTML-CRUD для нетех-суппорта) ·
  `httpapi` (operator bearer + `/webhooks/pay`).
- **Data plane:** ingress strongSwan IKEv2 + sing-box (TUN, VLESS-Reality к egress,
  GeoIP-split + walled-garden для restricted-пула); egress — Reality-сервер.
- **Доставка клиенту (ADR-0022):** iOS/macOS `.mobileconfig`, Android strongSwan + `.sswan`,
  оба через Telegram-бот.

---

## 5. Состояние по компонентам

| Компонент | Состояние | Проверено |
|---|---|---|
| Движок `sql.j2` (анти-локаут/срок/кап/accounting/revoke) | ✅ готов | **CI e2e** (FreeRADIUS+PG) |
| Миграции 0001–0007 | ✅ готовы | CI (apply на чистом PG16) |
| Бинарь `control` (contract/fleet/profiles/httpapi/panel) | ✅ собирается, тесты | CI (`go` + `control-db`) |
| ingress: walled-garden routing, DAE-листенер, симметрия | ✅ код/рендер | рендер JSON; **узел — нет** |
| egress: Reality | ✅ (с прошлых этапов) | вживую (PR #13) |
| Telegram-бот | ⚠️ mock | автономен; к `control` НЕ подключён |
| Нодовая валидация (живой туннель, CoA, sing-box) | ⏳ не начата | 22 пункта чек-листа открыты |

---

## 6. Что сломано / несогласованности / конфликты

### 6.1 «Труп» mock-бота (конфликт пивота)
`services/bot` — самодостаточный in-memory mock. Его код и доки ссылаются на `config-api`/
`orchestrator` («later wire to config-api») — а пивот эти сервисы **удаляет**. После мерджа
бот останется в `main`, но будет указывать на несуществующий код. Деплоится он условно
(`site.yml`: `role: bot when: bot_token`), так что прод не падает, но логически это висяк.
**Нужно:** переключить выдачу профиля на `control` httpapi (`provision` → `.mobileconfig`),
а уведомления — на новый источник. Параллельный трек, мердж не блокирует.

### 6.2 Протухшая ansible-роль control-plane (doc rot, НЕ рантайм)
Задачи роли корректны — собирают `control` (`tasks/main.yml:227`). Но **документация и
defaults роли отстали** от пивота:
- `roles/control-plane/claude.md` описывает `config-api`/`account-api`/`orchestrator`,
  ссылается на **3 несуществующих шаблона** (`config-api.service.j2`, `account-api.service.j2`,
  `orchestrator.service.j2` — их в `templates/` нет) и `depends-on` на удалённые
  `services/config-api`/`services/account-api`; пишет «миграции 0001–0005» (на деле 0001–0007).
- `roles/control-plane/defaults/main.yml` хранит мёртвые переменные `cp_accountapi_*`,
  `cp_digiseller_*`, `cp_configapi_*` и комментарий «Go-сервисы (config-api, orchestrator)».
  Tasks их не используют (собирается только `control`) → безвредно, но гнильё.

Handlers чистые (`restart freeradius` + `restart control`). Пред-имплементационный P3-проход
по докам этот угол пропустил. **Нужно:** переписать `control-plane/claude.md` + почистить
`defaults` под `control`.

### 6.3 ADR в статусе `proposed`
**ADR-0020, 0021, 0022, 0023 — `proposed`**, не `accepted`. Весь MVP стоит на непринятых
формально решениях. После мерджа/валидации — перевести в `accepted`.

### 6.4 Нодовая валидация не проведена
Движок CI-доказан, но **путь на живом узле — нет**: ядерный IPsec, EAP через реальный
strongSwan, CoA/DAE-обрыв, sing-box walled-garden routing, симметрия (диаспора→РФ-выход),
платёжный вебхук сквозняком. См. `docs/research/staging-validation-checklist.md` (22 пункта).

---

## 7. To-Do / открытые задачи

### Сейчас (разблокировать релиз)
- [ ] **Мердж PR #17 → `main`** (решение человека; технически готов).
- [ ] **Аппрув `environment: production`** в GitHub Actions для `deploy`/`deploy-ingress`
      (только человек; мердж в `main` ставит деплой в очередь под гейтом).

### Сразу после мерджа
- [ ] Нодовая валидация по `staging-validation-checklist.md` (туннель, анти-локаут на живом
      `radtest`, accounting→radacct, walled-garden, CoA/DAE, симметрия, вебхук оплаты).
- [ ] Перевести ADR-0021/0022/0023 (и решить судьбу 0020) в `accepted`.
- [ ] Почистить doc rot роли control-plane (§6.2): `claude.md` + `defaults`.
- [ ] Переподключить бот к `control` (§6.1): выдача `.mobileconfig` через `provision`.

### Бэклог (с прошлых этапов, вне MVP)
- [ ] **B3:** сплит по ASN, а не GeoIP-CIDR (сейчас GeoIP).
- [ ] **B4:** резидентный выход для госсайтов/банков (Госуслуги блокируют IP дата-центров —
      недоступны даже напрямую с ingress; не баг маршрутизации).
- [ ] **B1/B2:** observability; пересмотр слоёв шифрования.
- [ ] **Со-локация (ADR-0023, отложено):** один узел = и вход, и выход одновременно
      (комбинированный sing-box: TUN-inbound + Reality-server-inbound + routing по
      inbound-tag) + слияние TF-модулей `ingress`/`egress`. Оптимизация плотности, НЕ
      требование MVP; валидируется только на узле.
- [ ] **B5 (под вопросом):** публичный TLS-фронт — был привязан к account-api, после пивота,
      вероятно, неактуален; перепроверить.
- [ ] Коммерческий учёт — самописные коннекторы за швом `renew()` (вне MVP).

---

## 8. Что нужно от человека

1. **«Да» на мердж PR #17** — после этого вливаю в `main` тем же путём, что прошлые 15 PR.
2. **Аппрув `production`** — клик в GitHub Actions перед раскаткой на живые узлы; без него
   деплой просто висит в очереди, ничего не уезжает.

Остальное (нодовая валидация, чистка доков, бот, бэклог) — после мерджа, по приоритету.

# Пред-имплементационный аудит (read-only)

**Дата:** 2026-06-25 · **Питает:** реализацию ADR-0021/0022/0023 · **Метод:** 5 параллельных read-only агентов (ADR/доки, код, инфра, CI/безопасность, гигиена).

Цель: после серии разворотов архитектуры **доки синхронизированы с целью, но код/инфра/CI ещё отражают старый план.** Этот отчёт картирует разрыв до начала кода.

---

## 0. Резюме

- **Безопасность — чисто:** секретов в репо нет, `.gitignore` покрывает, инвариант «на RU ingress нет юзерских данных/секретов» держится. (Мелочь: `clients.conf.j2` хардкодит `testing123` для localhost-клиента — убрать/рандомизировать.)
- **Разрыв «доки vs цель»** — несколько живых доков ещё противоречат решению (radcheck, OpenWISP-вывод, тело backend-requirements). Чинится правкой текста.
- **Разрыв «код/инфра vs цель»** — это и есть план реализации. Главное: **вся учётно-квотная половина движка (radacct/sqlcounter/Expiration/CoA) физически отсутствует** — ни в миграциях, ни в FreeRADIUS-конфиге.

---

## 1. Несогласованность документации (чинить в docs)

| Файл | Находка | Severity |
|---|---|---|
| `docs/adr/0023:25` | Entitlement-атрибут в **`radcheck`** — но ADR-0021 канонический radcheck отменил (своя `auth_credentials`). Противоречие внутри живого target-набора. | HIGH |
| `docs/research/billing-and-bss-options.md:28` | «webhook → **`radcheck`**» — пишет в отменённую цель. | HIGH |
| `docs/research/radius-admin-panels.md` (Вывод) | Секция «Вывод» всё ещё **рекомендует OpenWISP**, противореча собственному баннеру и ADR-0021. | HIGH |
| `docs/backend-requirements.md` (тело §2/§7/§9/§11/§12/§14) | Под баннером-оверрайдом тело draft v3 по-прежнему описывает account-api, `usage_log`/`subscriptions`, рефералы, **месячный сброс** (ADR-0021: сброс на продление), Prometheus-в-MVP. Самый плотный очаг отменённого. | HIGH |
| `docs/adr/readme.md` индекс | Не кодирует цепочку amends/supersedes (0021↔0020, 0023↔0003/4/6, 0018/0007↔0020-MVP). | MED |
| `docs/adr/0020:68-91` | Поток описан против `account-api`; нет форвард-ссылки на 0021 (Platega → коннектор в `control`). | MED |
| `README.md:199`, `backend-requirements.md:135` | `nodes.role` CHECK = `ingress/egress/control` — ADR-0023 схлопывает в `node`. | MED |
| spike-заметка (title), `backlog.md` B1/B5 | «переезд на OpenWISP», B1 `usage_log`, B5 про `account-api`. | LOW |

---

## 2. Движок / схема — САМЫЙ БОЛЬШОЙ разрыв

`auth_credentials` (своя схема) есть и **корректно прошита** в `sql.j2` (`authorize_check_query`, не radcheck) ✓. Но дальше — пусто:

- **`radacct` не существует** нигде (ни в миграциях, ни в Go). strongSwan шлёт `accounting=yes` (`ingress/eap-radius.conf.j2:11`), порт 1813 открыт — но `sql.j2` **не имеет `accounting{}`-запросов**, и control-plane вообще не управляет `sites-available/default`/acct-листенером → **accounting-пакеты прилетают и отбрасываются.**
- **Нет `rlm_sqlcounter`** → кап по МБ неэнфорсим.
- **Нет `Expiration`** → у `auth_credentials` нет колонки срока; срок жил в `subscriptions.expires_at` и гасился cron'ом.
- **Нет CoA/`dae`** → отозвать живую сессию нельзя, только до следующего ре-auth.

**Нужно создать:** миграцию с `radacct` (+ срок/кап на `auth_credentials` или radcheck-style строки); `accounting{}`-запросы в `sql.j2` + `sites`-конфиг (включить `sql` в `preacct`/`accounting`); модули `sqlcounter` + `expiration`; `dae`-листенер на узле + origination CoA в `control`.

---

## 3. Код — слияние в `control`

**Переиспользование велико для каркаса, тонко для движка:**

| Целевой пакет | Источник |
|---|---|
| `contract` | `config-api` `Issue()` (идемпотентный upsert) + `account-api` device CRUD + sticky-IP |
| `profiles` | `config-api` `mobileconfig` (+ `.sswan` — новое) |
| `fleet` | `orchestrator` node-registry + `health` + `/nodes` |
| `panel` | `account-api` `token` (JWT → операторский) + httpapi-каркас |
| scaffold | pgxpool `Store`, `slog`, `config.FromEnv` — везде одинаковы |

**Растворяется:** `account-api` auth/OTP/Telegram/sessions/billing/referral; `config-api/internal/digiseller`; **оркестраторский account-cron** (`internal/cron` + `store/lifecycle.go` — 8 sweeps `ExpireSubscriptions/EnforceQuota/RollPeriods/...`) — **крупнейший дрифт, чистое удаление**, поведение переезжает в `sqlcounter`+`Expiration`.

**Подводные камни слияния:**
- 3 Go-модуля → 1 (оркестратор пинит более старые `x/*`). Дедуп `credentials.NTHash` (2 копии), `mobileconfig` (2 копии, **разные token-наборы и идентификаторы** `com.smartinternet.vpn` vs `pro.flynet.vpn`), `randFramedIP` (2 копии).
- **Конфликт политики:** `config-api` ротирует `nt_hash` на продление, цель — стабильные креды. Контракт должен выбрать одно.
- **Три текущих писателя в `auth_credentials`** (config-api Issue, account-api device CRUD, orchestrator cron) → схлопнуть в один (contract = единственный писатель).

---

## 4. Инфра / IaC

- **Узлы полностью асимметричны:** раздельные Ansible-роли `ingress`/`egress` + раздельные Terraform-модули + раздельные environments (`mmvp` vs `mmvp-ingress`). Симметричного `node` нет. Нужна одна параметризованная роль/модуль `node` (регион + capability вход/выход).
- **Сплит однонаправленный** (РФ-direct, остальное в туннель), обратного профиля для диаспоры нет. Причём **ASN-split ещё не реализован** — только GeoIP-CIDR (`ingress/claude.md:6` «ASN — backlog B3»), хотя корневой `CLAUDE.md` заявляет «ASN-split с самого начала» → **расхождение док-vs-реальность, предшествующее нашему пивоту.**
- FreeRADIUS-дыры — см. §2.
- 3 systemd-юнита → 1 `control`-юнит.

---

## 5. CI/CD и безопасность

- **Секретов в репо нет** (проверено: ни ключей, ни vault-плейнтекста; `.gitignore` ок; секреты идут GitHub Actions → vault при деплое). Инвариант RU-ingress держится. ✅
- `test.yml` матрица `[config-api, account-api, orchestrator]` → один `control`.
- `deploy.yml` `CONTROL_ACCOUNT` + материализация `zz_account.yml` → убрать (account-api).
- `control-plane/tasks/main.yml` собирает 3 бинаря / 3 юнита / 3 хендлера → один `control`.
- Миграции: `0003`/`0004` — app-схема account-api (растворяется), `0005` расширяет **сохраняемую** `auth_credentials` (оставить). CI против растворяемой схемы не гоняет (integration-тесты DSN-gated, скипаются).

---

## 6. Приоритизированный план приведения (выход аудита)

**P0 — движок (без него MVP не работает):**
1. Миграция: `radacct` + срок/кап на `auth_credentials`.
2. `sql.j2`: `accounting{}` + `sites`-конфиг (acct-листенер, `sql` в preacct/accounting).
3. Модули `sqlcounter` (кап МБ) + `expiration` (срок).
4. CoA: `dae` на узле + origination в `control`.

**P1 — код:**
5. Собрать `control` (contract/profiles/fleet/panel) из существующих пакетов; дедуп; выбрать политику nt_hash; единственный писатель.
6. Снести account-api + оркестраторский account-cron.

**P2 — инфра/CI:**
7. Симметричный `node` (Ansible-роль + TF-модуль), двунаправленный сплит.
8. CI/deploy: один `control` (test-матрица, юнит, секреты); убрать `CONTROL_ACCOUNT`.
9. Ретайр миграций `0003`/`0004`.

**P3 — доки (быстро):**
10. Починить §1 (radcheck→auth_credentials в 0023/research; Вывод radius-admin-panels; тело backend-requirements под «ИСТОРИЯ»; индекс amends; nodes.role; ASN-split-реальность).

---

## 7. Гигиена per-dir доков

**Единственный док, отражающий цель, — `docs/architecture/readme.md`.** Весь операционный слой (root `README.md`, `services/*`, `infra/*`, `.github/*`) написан под MMVP/account-api-мир и отстал.

**Сильнее всего противоречат цели (по приоритету):**
1. `services/account-api/{readme,claude}.md` — документируют **растворяемый** сервис как живой «единственный источник прав» (конфликт со швом `contract.renew()`).
2. `services/{readme,claude}.md` — закрепляют связку config-api(MMVP)+account-api(MVP) как архитектуру; о `control` ни слова.
3. `README.md` (root) — целиком MMVP-спек: Plati/Digiseller-продажа, iOS-only `.mobileconfig`, асимметричные ingress/egress, standalone config-api/orchestrator. Допустимо **только** с явной пометкой «исторический спек».
4. `services/config-api/{readme,claude}.md` — целиком привязаны к Plati/Digiseller (MMVP-путь).
5. Инфра (`terraform/modules/*`, `ansible/roles/{ingress,egress}`) — фиксированные роли vs симметричный `node` (ADR-0023).

**Прочее:** `services/db` — `auth_credentials` на цели ✓, но идемпотентность «по `plati_order_id`» — MMVP-фрейминг; `services/bot` — доставляет только `.mobileconfig`, нужно `.sswan` + ссылка на `control`; `docs/readme.md` не перечисляет подкаталоги `architecture/` и `research/`; `account-api/claude.md:8` — висячая ссылка `§9`. Битых ссылок нет.

**Конвенция:** правило «в КАЖДОМ каталоге readme+claude» — все *компонентные* каталоги соблюдают; ~50 code/leaf-каталогов (Go `internal/*`, ansible `tasks/templates`, миграции) пары не имеют. Это неоднозначность формулировки («каждый» vs «каждый компонент»), а не дефект.

> Правки доков (§1 + §7) — это P3: делаются быстро и независимо от кода. Самое ценное — `services/` индекс и `account-api/`-пара (активно выдают растворяемую архитектуру за канон) + пометка root `README.md` как исторического.

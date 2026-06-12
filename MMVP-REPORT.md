# Smart Internet — MMVP: отчёт о проделанной работе

**Ветка:** `claude/kind-pasteur-2IMEz` · **Репозиторий:** Institution-A93/FlyVPN

Заложен полный каркас MMVP «умного» VPN для РФ: структура проекта, инфраструктура как
код (IaC), дата-плейн (ingress↔egress) и backend-сервисы control plane. Ключевые узлы
проверены на живых компонентах (FreeRADIUS, PostgreSQL, sing-box, Go-тесты).

---

## 1. Что сделано

### Структура и процесс
- Конвенция документации: в каждом каталоге `readme.md` (проза) + `claude.md` (XML для
  агента), канон в корневом `CLAUDE.md`.
- Все ключевые решения зафиксированы как **ADR** (`docs/adr/`, 15 шт.).
- Гигиена секретов: `.gitignore` под tfstate/tfvars/vault/ключи; IP узлов и боевые
  параметры — вне репозитория.

### Инфраструктура (OpenTofu)
- Модули по ролям: **egress** (Hetzner), **control-plane** (Hetzner), **ingress**
  (Selectel/OpenStack).
- Окружение `mmvp`: применимый `tofu apply` поднимает egress + control plane (токен
  через `TF_VAR_*`, state локальный на старте).

### Конфигурация узлов (Ansible)
- **egress**: sing-box (VLESS-Reality **server**) + unbound (рекурсивный DNS, локально).
- **control-plane**: PostgreSQL (роль-владелец + read-only для RADIUS) + FreeRADIUS
  (EAP-MSCHAPv2 по NT-hash из `auth_credentials`, sticky Framed-IP, отзыв по `revoked_at`).
- **ingress**: strongSwan (IKEv2 + EAP-RADIUS на control plane) + sing-box (TUN +
  ASN-split: RU→напрямую, остальное→туннель к egress) + ежедневное обновление
  RU-префиксов (`sing-box rule-set compile`).
- Обвязка: `site.yml`, inventory-пример, `requirements.yml`, boot-секреты через
  ansible-vault.

### Backend-сервисы (Go)
- **config-api**: выдача по уникальному коду Plati/Digiseller (ADR-0018), генерация
  EAP-кредов (username + NT-hash), идемпотентная по `uniquecode` запись в БД, генерация
  `.mobileconfig` (IKEv2/EAP-MSCHAPv2) из шаблона. **На проде**: товар Digiseller 5937891
  (`uniqueunfixed`/`DigisellerCode`, `verify_url=/plati/issue`), сквозная покупка →
  выдача `.mobileconfig` проверена вживую.
- **orchestrator**: реестр узлов (идемпотентная регистрация по `public_ip`),
  health-checking (TLS-пробы egress/control, пометка down после порога неудач),
  admin-API (`/healthz`, `/nodes`).
- **db**: схема PostgreSQL как plain-SQL миграции (0001 — 6 таблиц по README §4;
  0002 — UNIQUE на `nodes.public_ip`).

---

## 2. Архитектурные решения (ADR)

| #    | Решение | Статус |
|------|---------|--------|
| 0001 | Граница MMVP | accepted |
| 0002 | Реальное облако с самого старта | accepted |
| 0003 | Egress + control plane на Hetzner | accepted |
| 0004 | Ingress на Selectel | accepted |
| 0005 | Auth: RADIUS / EAP-MSCHAPv2 | accepted |
| 0006 | ASN-split с самого начала | accepted |
| 0007 | Plati-интеграция входит в MMVP | accepted |
| 0008 | Конвенция документации (readme.md + claude.md) | accepted |
| 0009 | Только OSS-компоненты; IaC-тул — OpenTofu (не Terraform/BUSL) | accepted |
| 0010 | Лицензия проекта — AGPL-3.0 | accepted |
| 0011 | Камуфляж egress через Reality `dest`; DNS на egress | accepted |
| 0012 | Boot-секреты — ansible-vault (MMVP) / оркестратор (ph2) | accepted |
| 0013 | Язык backend-сервисов — Go | accepted |
| 0014 | Хранение пароля при EAP-MSCHAPv2 — NT-hash (bcrypt несовместим) | accepted |
| 0015 | sing-box транспорт mesh (обе стороны); ASN-split в sing-box | accepted |

Существенные отклонения от исходной спеки (обоснованы в ADR):
- **OpenTofu** вместо Terraform (BUSL ≠ OSS).
- **NT-hash** вместо bcrypt — bcrypt криптографически несовместим с EAP-MSCHAPv2
  (проверено: Apple IKEv2 поддерживает только MSCHAPv2/EAP-TLS).
- **sing-box** вместо Xray-core по обе стороны mesh — нативный TUN решает «IP-пакеты в
  Reality»; ASN-split переехал в маршрутизацию sing-box (тот же исход/источник, что в 0006).

---

## 3. Что и как проверено

| Компонент | Проверка | Статус |
|-----------|----------|--------|
| Схема БД (0001/0002) | применение + откат на **PostgreSQL 16**, CHECK на nt_hash, UNIQUE на public_ip | ✅ |
| Auth-путь | **FreeRADIUS 3.2.5 + PostgreSQL 16**: верный/неверный/неизвестный/отозванный → корректный Accept/Reject + Framed-IP | ✅ |
| config-api ↔ FreeRADIUS | E2E: выданный кред аутентифицируется через реальный шаблон роли (Access-Accept) | ✅ |
| config-api (Go) | `go build/vet/test`: NT-hash по известным векторам, клиент Digiseller (токен+проверка кода), рендер .mobileconfig (на боевом шаблоне), store на PG16 | ✅ |
| orchestrator (Go) | `go build/vet/test`: TLS-проба, логика порога, реестр на PG16 | ✅ |
| sing-box egress + ingress | конфиги проходят `sing-box check` (v1.10.7) | ✅ |
| strongSwan (ingress) | офлайн не валидируется (нужен живой узел + клиент) | ⛔️ не проверено |
| OpenTofu-модули | нет `tofu`/аккаунтов в среде; Selectel-модуль особенно требует валидации | ⛔️ не проверено |

---

## 4. Не входило в MMVP / осталось

- **Реальные аккаунты/токены**: есть Hetzner; нет Selectel и домена → ingress пока не
  включён в окружение, реальный `apply` egress/control ждёт токена.
- **Истечение подписки**: принудительного отзыва кредов после `expires_at` пока нет
  (cron/orchestrator) — следующий шаг.
- **Авто-редирект Digiseller**: файл отдаётся по `verify_url`; бесшовность «оплатил →
  сразу профиль» зависит от тумблера `auto_verify` в панели товара — финальная сверка UX.
- **Продление подписки = ротация пароля** (профиль переустанавливается) — MMVP-поведение.
- Phase 2: GeoDNS (OSS-вариант вместо Cloudflare/NS1), авто-ротация секретов, выдача
  секретов узлам оркестратором, FDE remote unlock, метрики, alerting, глубокая
  IKE-проба ingress, Android/Windows.

---

## 5. История (коммиты ветки)

```
Скелет MMVP: структура каталогов, ADR-ы и конвенция документации
Принцип «только OSS»: OpenTofu вместо Terraform, флаг GeoDNS, .gitignore
Лицензия проекта — AGPL-3.0
egress: OpenTofu-модуль под Hetzner (сервер + firewall)
egress: Ansible-роль (изначально Xray; позже sing-box)
mmvp env: root-модуль OpenTofu
ansible: обвязка egress + ADR-0012 boot-секреты
control-plane: OpenTofu-модуль (Hetzner)
fix: сузить gitignore-паттерны секретов; добавить ADR-0012
Язык backend — открытое решение (снят «Go по умолчанию»)
db: схема PostgreSQL (миграции)
config-api: шаблон .mobileconfig + ADR-0014 (конфликт MSCHAPv2/bcrypt)
auth: NT-hash вместо bcrypt (ADR-0014)
control-plane: Ansible-роль PostgreSQL + FreeRADIUS
ADR-0013 accepted: backend на Go
config-api: реализация на Go (выдача по коду Plati/Digiseller, NT-hash, .mobileconfig)
ingress: OpenTofu-модуль под Selectel
mesh: sing-box по обе стороны + ASN-split в sing-box (ADR-0015) + ingress-роль
orchestrator: реестр узлов + health-checking (+ миграция 0002)
```

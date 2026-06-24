# role: control-plane

Ansible-роль. Настраивает foreign control plane после провижна OpenTofu-модулем
`control-plane`.

Реализовано:
- **PostgreSQL** — БД `smartinternet`, роль-владелец `si_app` (config-api/account-api/orchestrator)
  и read-only роль `si_radius` (только SELECT на `auth_credentials`). Миграции из
  `services/db` (0001–0005) доставляются на узел и применяются идемпотентно (маркеры-схема).
- **FreeRADIUS** — EAP-MSCHAPv2 через `rlm_sql` (postgresql): NT-Password и sticky
  Framed-IP-Address из `auth_credentials`, отзыв по `revoked_at`. clients.conf —
  ingress-узлы как RADIUS-клиенты.
- **config-api** (Go) — собирается на узле (пин Go-тулчейна), systemd-юнит, env из vault.
  TLS/ACME (Let's Encrypt) для `cp_configapi_acme_domain` (по умолчанию `api.fly-vpn.net`):
  сервис слушает `:443` (TLS) + `:80` (HTTP-01 challenge).
- **account-api** (Go) — веб-кабинет MVP: собирается на узле, systemd-юнит, env из vault.
  Слушает plain-HTTP `:8080` (TLS/публичный домен `api.flynet.pro` — за reverse-proxy, т.к.
  config-api уже держит `:443`). Запускается только при заданном `cp_accountapi_jwt_secret`
  (иначе юнит установлен, но не активирован — деплой не падает). ADR-0020.
- **orchestrator** (Go) — собирается на узле, systemd-юнит, env из vault (реестр+health +
  cron-обязанности аккаунт-слоя MVP).

## Проверено
Связка FreeRADIUS 3.2.5 + PostgreSQL 16 протестирована локально (radtest -t mschap):
верный пароль → Access-Accept + `Framed-IP-Address`; неверный/неизвестный/отозванный →
Access-Reject. Шаблон `sql.j2` проверен на рендер Jinja2 и `freeradius -XC`.

## Зависимости
Коллекция `community.postgresql` (см. `../../requirements.yml`):
```sh
ansible-galaxy collection install -r infra/ansible/requirements.yml
```

## Секреты (vault, ADR-0012)
`cp_db_app_password`, `cp_db_radius_password`, `cp_radius_clients` — из
`group_vars/control/vault.yml` (в репозитории только `*.example`). Роль падает без обязательных.
Креды Digiseller (`cp_digiseller_seller_id`, `cp_digiseller_api_key`, `cp_plan_by_goods`) —
**опциональны** и живут в отдельном `group_vars/control/zz_digiseller.yml` (на деплое — из
GitHub-секрета `CONTROL_DIGISELLER`; см. `*.example`). Без них config-api стартует,
но `/plati/issue` отдаёт 503 (ADR-0018).
Секреты account-api (`cp_accountapi_jwt_secret` — обяз. для запуска; `cp_accountapi_platega_*`,
`cp_accountapi_twilio_*`, `cp_accountapi_telegram_bot_token` — опц.) — в
`group_vars/control/zz_account.yml` (на деплое — из `CONTROL_ACCOUNT`). Без JWT-секрета
account-api не активируется; без Platega — purchase/webhook отдают 503 (ADR-0020).

## Ключевые переменные (defaults)
`cp_db_name` (`smartinternet`), `cp_db_app_user`/`cp_db_radius_user`, `cp_go_version`,
`cp_configapi_acme_domain` (`api.fly-vpn.net`), `cp_configapi_vpn_remote`
(`vpn.fly-vpn.net`), `cp_orch_listen`. Сборка Go — на узле в `cp_src_dir`.

## Порты (TF-firewall control-plane)
`443` (config-api HTTPS / Plati / ACME TLS-ALPN), `80` (ACME HTTP-01), `22` (SSH admin).
account-api слушает `:8080` локально и **не** публикуется напрямую: для веб-кабинета и
вебхука Platega нужен TLS-фронт (reverse-proxy на `api.flynet.pro`), т.к. `:443` занят
config-api. Установка прокси/домена — отдельная задача (backlog B5).

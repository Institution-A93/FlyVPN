# role: control-plane

Ansible-роль. Настраивает foreign control plane после провижна OpenTofu-модулем
`control-plane`.

Реализовано:
- **PostgreSQL** — БД `smartinternet`, роль-владелец `si_app` (control) и read-only
  роль `si_radius` (только SELECT на `auth_credentials` для FreeRADIUS). Миграции из
  `services/db` доставляются на узел и применяются идемпотентно по маркерам. Целевая
  схема (ADR-0021): применяются `0001/0002/0005/0006/0007` — account-слой `0003/0004`
  растворён и не применяется; `0007` чистит остатки.
- **FreeRADIUS** — EAP-MSCHAPv2 через `rlm_sql` (postgresql) над `auth_credentials`.
  authorize отдаёт NT-Password только активному креду и выбирает **пул Framed-IP** по
  состоянию (active `10.8/14` / restricted `10.12/14` — анти-локаут, ADR-0021);
  accounting пишет `radacct`; срок/кап — в SQL (без отдельных модулей). `sql.j2`.
- **control** (Go, единый бинарь, ADR-0021) — собирается на узле (пин Go-тулчейна),
  один systemd-юнит `control`, env из vault. Внутренний листенер `:8080` (за приватной
  сетью). Пакеты: contract (единственный писатель), profiles, fleet (реестр/health/CoA),
  panel (HTML для суппорта), webhook-коннекторы. Поглотил config-api и orchestrator.

## Проверено
FreeRADIUS 3.2.5 + PostgreSQL 16 (локально, схема `services/db` 0001+0006): active →
Accept + Framed-IP из active-пула; expired/over-cap → Accept + restricted-пул (туннель
поднят); revoked → Reject; accounting → `radacct`. `control` собран и протестирован
(contract/profiles/fleet + смоук). `sql.j2` — рендер Jinja2 + `freeradius -XC`.

## Зависимости
Коллекция `community.postgresql` (см. `../../requirements.yml`):
```sh
ansible-galaxy collection install -r infra/ansible/requirements.yml
```

## Секреты (vault, ADR-0012)
`cp_db_app_password`, `cp_db_radius_password`, `cp_radius_clients` — из
`group_vars/control/vault.yml` (в репозитории только `*.example`). Роль падает без
обязательных. Токены control (`cp_control_operator_token` — для панели/API;
`cp_control_webhook_secret` — для платёжных коннекторов; `cp_control_dae_secret` —
общий с узлами CoA/DAE) — в `group_vars/control/zz_control.yml` (на деплое из
GitHub-секрета `CONTROL_APP`). Без operator-токена панель отвергает вход.

## Ключевые переменные (defaults)
`cp_db_name` (`smartinternet`), `cp_db_app_user`/`cp_db_radius_user`, `cp_go_version`,
`cp_control_listen`, `cp_control_dae_port`, `cp_configapi_vpn_remote`/`_id` (адрес/CN
ingress-узла для профилей). Сборка Go — на узле в `cp_src_dir`.

## Порты (TF-firewall control-plane)
`1812-1813/udp` (RADIUS auth/acct от ingress), `22` (SSH admin). `control` слушает
`:8080` локально (внутренняя сеть/доступ через SSH-туннель или приватный фронт) — публично
не светится. Доставку профилей клиенту опосредует Telegram-бот (ADR-0022).

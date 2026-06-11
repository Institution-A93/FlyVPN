# role: control-plane

Ansible-роль. Настраивает foreign control plane после провижна OpenTofu-модулем
`control-plane`.

Реализовано:
- **PostgreSQL** — БД `smartinternet`, роль-владелец `si_app` (config-api/orchestrator)
  и read-only роль `si_radius` (только SELECT на `auth_credentials`). Миграции из
  `services/db` (0001 + 0002) доставляются на узел и применяются.
- **FreeRADIUS** — EAP-MSCHAPv2 через `rlm_sql` (postgresql): NT-Password и sticky
  Framed-IP-Address из `auth_credentials`, отзыв по `revoked_at`. clients.conf —
  ingress-узлы как RADIUS-клиенты.
- **config-api** (Go) — собирается на узле (пин Go-тулчейна), systemd-юнит, env из vault.
  TLS/ACME (Let's Encrypt) для `cp_configapi_acme_domain` (по умолчанию `api.fly-vpn.net`):
  сервис слушает `:443` (TLS) + `:80` (HTTP-01 challenge).
- **orchestrator** (Go) — собирается на узле, systemd-юнит, env из vault (реестр+health).

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
`cp_db_app_password`, `cp_db_radius_password`, `cp_plati_secret` (HMAC Plati),
`cp_radius_clients` — из `group_vars/control/vault.yml` (в репозитории только `*.example`).
Роль падает без обязательных.

## Ключевые переменные (defaults)
`cp_db_name` (`smartinternet`), `cp_db_app_user`/`cp_db_radius_user`, `cp_go_version`,
`cp_configapi_acme_domain` (`api.fly-vpn.net`), `cp_configapi_vpn_remote`
(`vpn.fly-vpn.net`), `cp_orch_listen`. Сборка Go — на узле в `cp_src_dir`.

## Порты (TF-firewall control-plane)
`443` (config-api HTTPS / Plati / ACME TLS-ALPN), `80` (ACME HTTP-01), `22` (SSH admin).

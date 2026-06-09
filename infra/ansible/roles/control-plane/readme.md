# role: control-plane

Ansible-роль. Настраивает foreign control plane после провижна OpenTofu-модулем
`control-plane`.

Реализовано:
- **PostgreSQL** — БД `smartinternet`, роль-владелец `si_app` (config-api/orchestrator)
  и read-only роль `si_radius` (только SELECT на `auth_credentials`). Миграции из
  `services/db` доставляются на узел и применяются.
- **FreeRADIUS** — EAP-MSCHAPv2 через `rlm_sql` (postgresql): NT-Password и sticky
  Framed-IP-Address из `auth_credentials`, отзыв по `revoked_at`. clients.conf —
  ingress-узлы как RADIUS-клиенты.

Не входит (ждёт выбора языка, ADR-0013): **config-api**, **orchestrator**.

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
`group_vars/control/vault.yml` (в репозитории только `*.example`). Роль падает без них.

## Ключевые переменные (defaults)
`cp_db_name` (`smartinternet`), `cp_db_app_user` (`si_app`), `cp_db_radius_user`
(`si_radius`), `cp_db_host`/`cp_db_port`, `cp_migrations_dir`, `cp_migrations_src`.

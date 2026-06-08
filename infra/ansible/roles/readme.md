# infra/ansible/roles

Ansible-роли по типам узлов. Каждая роль ставит и настраивает стек своей роли.

- [`ingress/`](./ingress/readme.md) — strongSwan, RADIUS-клиент, nftables + ipset RU-префиксов, Xray-client.
- [`egress/`](./egress/readme.md) — Xray Reality-server, nginx-fallback, unbound, NAT.
- [`control-plane/`](./control-plane/readme.md) — FreeRADIUS, PostgreSQL, config-api, orchestrator.

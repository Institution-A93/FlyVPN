# infra/ansible/roles

Ansible-роли по типам узлов. Каждая роль ставит и настраивает стек своей роли.

- [`ingress/`](./ingress/readme.md) — strongSwan, RADIUS-клиент, sing-box (TUN + ASN-split rule_set).
- [`egress/`](./egress/readme.md) — sing-box VLESS-Reality server (DNS — сам sing-box).
- [`control-plane/`](./control-plane/readme.md) — FreeRADIUS, PostgreSQL, config-api, orchestrator.

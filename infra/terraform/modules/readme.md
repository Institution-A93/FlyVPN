# infra/terraform/modules

Переиспользуемые OpenTofu-модули по ролям узлов. Каждый модуль создаёт машину, сеть
и базовые правила доступа для своей роли и отдаёт наружу IP/ID для регистрации в
оркестраторе.

- [`ingress/`](./ingress/readme.md) — RU ingress (Selectel).
- [`egress/`](./egress/readme.md) — foreign egress (Hetzner).
- [`control-plane/`](./control-plane/readme.md) — foreign control plane (Hetzner).

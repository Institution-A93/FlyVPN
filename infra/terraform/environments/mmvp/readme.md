# environment: mmvp

Окружение MMVP: по одному узлу каждой роли.

- egress → модуль `egress` (Hetzner)
- control-plane → модуль `control-plane` (Hetzner)
- ingress → модуль `ingress` (Selectel) — подключается по факту появления RU-аккаунта

Реализация root-модуля (providers.tf, main.tf, backend) — TODO.

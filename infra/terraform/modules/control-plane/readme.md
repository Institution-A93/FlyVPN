# module: control-plane (Hetzner)

Поднимает foreign control plane на Hetzner Cloud: VM(ы), приватная сеть, firewall.
Публично торчат только config-api (HTTPS, вебхук Plati) и admin-доступ. Узел держит
FreeRADIUS, PostgreSQL, config-api, orchestrator (см. ansible-роль `control-plane`).

Inputs (план): локация, тип сервера, ssh-ключ, образ, hcloud-токен.
Outputs (план): public_ip, private_ip, node_id.

Реализация модуля — TODO.

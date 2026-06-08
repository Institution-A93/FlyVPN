# module: egress (Hetzner)

Поднимает foreign egress-узел на Hetzner Cloud (hcloud): VM, сеть, firewall под 443
(Reality) и исходящий NAT. Узел держит Xray Reality-server, nginx-fallback, unbound
(см. ansible-роль `egress`).

Inputs (план): локация, тип сервера, ssh-ключ, образ, hcloud-токен.
Outputs (план): public_ip, node_id.

Реализация модуля — TODO.

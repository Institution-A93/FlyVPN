# module: ingress (Selectel)

Поднимает RU ingress-узел на Selectel: VM, сеть, security-группы под IKEv2
(UDP 500/4500) и исходящий доступ. Узел держит strongSwan, RADIUS-клиент, nftables,
Xray-client (см. ansible-роль `ingress`).

Inputs (план): регион, размер VM, ssh-ключ, ID образа, провайдерские креды.
Outputs (план): public_ip, node_id — для регистрации в оркестраторе.

Реализация модуля — TODO.

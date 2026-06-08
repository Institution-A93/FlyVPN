# role: ingress

Конфигурирует RU ingress:
- strongSwan — терминатор IKEv2/IPsec (UDP 500/4500)
- RADIUS-клиент/прокси — EAP-аутентификация к FreeRADIUS на control plane через туннель
- nftables + ipset `ru_prefixes` — ASN-split (RU напрямую, остальное в туннель), крон обновления префиксов
- Xray-core (VLESS-Reality client) — туннель к egress
- DNS-форвардер запросов клиентов в туннель

Реализация (tasks/templates) — TODO.

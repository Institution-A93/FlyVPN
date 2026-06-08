# role: egress

Конфигурирует foreign egress:
- Xray-core (VLESS-Reality server) — терминатор mesh-туннеля
- nginx — fallback-«легенда» на 443 под реалистичный TLS
- unbound — рекурсивный DNS-резолвер для клиентов
- nftables/iptables — NAT исходящего трафика

Реализация (tasks/templates) — TODO.

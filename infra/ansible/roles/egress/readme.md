# role: egress

Ansible-роль. Настраивает foreign egress-узел после провижна OpenTofu-модулем `egress`.

Состав:
- **Xray-core (VLESS-Reality server)** — терминатор mesh-туннеля на `egress_listen_port` (443).
- **unbound** — рекурсивный DNS-резолвер; слушает только локально, Xray направляет DNS
  клиентов в него (резолв происходит на egress, не утекает из РФ).

Камуфляж — через Reality `dest` на реальный сайт (не nginx-fallback на 443; см. ADR-0011).
Исходящие соединения инициирует сам Xray (`freedom`), отдельный NAT/MASQUERADE для
proxy-режима не нужен. Приватные сети заблокированы routing-правилом.

## Секреты (НЕ в репозитории)

Приходят от оркестратора через vault/inventory; роль падает с понятной ошибкой, если
не переданы:

- `egress_reality_private_key` — x25519 private key.
- `egress_reality_short_ids` — список shortId.
- `egress_clients` — `[{id: <uuid>, email: <tag>}]`.

## Ключевые переменные (defaults)

| Переменная                   | Default            | Назначение |
|------------------------------|--------------------|------------|
| `egress_xray_version`        | `v1.8.24`          | пин версии Xray (воспроизводимость) |
| `egress_reality_dest`        | `www.lovo.ai:443`  | сайт-донор TLS-хендшейка |
| `egress_reality_server_names`| `[www.lovo.ai]`    | SNI cover (соответствует dest) |
| `egress_listen_port`         | `443`              | порт VLESS-Reality |
| `egress_unbound_port`        | `5353`             | локальный порт unbound |

## Проверка

После прогона: `systemctl status xray unbound`; снаружи — подключить VLESS-Reality
клиент с теми же ключами и убедиться, что трафик выходит, а прямой заход на IP отдаёт
TLS как `dest`.

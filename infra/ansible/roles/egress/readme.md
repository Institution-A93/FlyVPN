# role: egress

Ansible-роль. Настраивает foreign egress после провижна OpenTofu-модулем `egress`.

Состав:
- **sing-box (VLESS-Reality server)** — терминатор mesh-туннеля на `egress_listen_port` (443).
  Один компонент mesh по обе стороны (ADR-0015).
- **unbound** — рекурсивный DNS-резолвер, слушает только локально; sing-box направляет DNS
  клиентов в него (резолв на egress, не утекает из РФ).

Камуфляж — Reality `dest`/`server_name` на реальный сайт (ADR-0011). Приватные сети
заблокированы route-правилом (`ip_is_private` → block). Конфиг проверяется
`sing-box check` при деплое (`validate:`).

## Секреты (НЕ в репозитории; vault, ADR-0012)
`egress_reality_private_key`, `egress_reality_short_ids`, `egress_clients`
(`[{uuid, name}]`). Роль падает, если не переданы.

## Ключевые переменные (defaults)
| Переменная                  | Default           | Назначение |
|-----------------------------|-------------------|------------|
| `egress_singbox_version`    | `1.10.7`          | пин версии sing-box |
| `egress_reality_server_name`| `www.lovo.ai`     | SNI, принимаемый сервером |
| `egress_reality_dest_server`| `www.lovo.ai`     | сайт-донор handshake |
| `egress_listen_port`        | `443`             | порт VLESS-Reality |
| `egress_unbound_port`       | `5353`            | локальный порт unbound |

## Проверено
Сгенерированный конфиг sing-box валиден (`sing-box check`, v1.10.7). После прогона:
`systemctl status sing-box unbound`.

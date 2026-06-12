# role: egress

Ansible-роль. Настраивает foreign egress после провижна OpenTofu-модулем `egress`.

Состав:
- **sing-box (VLESS-Reality server)** — терминатор mesh-туннеля на `egress_listen_port` (443).
  Один компонент mesh по обе стороны (ADR-0015).

DNS клиентов резолвит сам sing-box через публичный резолвер (`egress_dns_upstream`,
по умолчанию `1.1.1.1`) — запрос уходит из egress, не утекает из РФ. Отдельный unbound
не используется (ADR-0017). Камуфляж — Reality `dest`/`server_name` на реальный сайт
(ADR-0011); приватные сети заблокированы route-правилом (`ip_is_private` → block).
Конфиг проверяется `sing-box check` при деплое (`validate:`).

## Секреты (НЕ в репозитории; vault, ADR-0012)
`egress_reality_private_key`, `egress_reality_short_ids`, `egress_clients`
(`[{uuid, name}]`). Роль падает, если не переданы.

## Ключевые переменные (defaults)
| Переменная                   | Default           | Назначение |
|------------------------------|-------------------|------------|
| `egress_singbox_version`     | `1.10.7`          | пин версии sing-box |
| `egress_reality_server_name` | `www.lovo.ai`     | SNI, принимаемый сервером (cover) |
| `egress_reality_dest_server` | `www.lovo.ai`     | сайт-донор handshake |
| `egress_listen_port`         | `443`             | порт VLESS-Reality |
| `egress_dns_upstream`        | `1.1.1.1`         | публичный резолвер для DNS клиентов |

## Проверено
Конфиг sing-box валиден (`sing-box check`, v1.10.7) и развёрнут в проде. После прогона:
`systemctl status sing-box`.

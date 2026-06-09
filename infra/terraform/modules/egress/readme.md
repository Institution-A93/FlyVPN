# module: egress (Hetzner)

OpenTofu-модуль. Поднимает foreign egress-узел на Hetzner Cloud: сервер + firewall.
Дальнейшая настройка (Xray Reality-server, nginx-fallback, unbound, NAT) — ansible-роль
`egress`. Конфигурация провайдера `hcloud` задаётся в окружении, не в модуле.

## Inputs

| Переменная        | Тип            | Default      | Назначение |
|-------------------|----------------|--------------|------------|
| `name`            | string         | `egress`     | имя узла |
| `location`        | string         | `hel1`       | локация Hetzner (nbg1/fsn1/hel1/ash/hil) |
| `server_type`     | string         | `cx22`       | тип сервера |
| `image`           | string         | `debian-12`  | образ ОС |
| `ssh_key_names`   | list(string)   | —            | имена SSH-ключей в Hetzner (обязательно) |
| `admin_ssh_cidrs` | list(string)   | —            | CIDR, с которых разрешён SSH (обязательно) |
| `reality_port`    | number         | `443`        | порт VLESS-Reality |
| `labels`          | map(string)    | `{}`         | доп. метки |

## Outputs

`id`, `public_ipv4`, `public_ipv6`, `name`, `location` — для регистрации узла в
оркестраторе и для Ansible inventory.

## Firewall

Снаружи открыты только `reality_port` (443) и SSH (22) — последний лишь с
`admin_ssh_cidrs`. Исходящий трафик не ограничен (egress по роли ходит в интернет).

## Применение

Модуль вызывается из окружения (`environments/mmvp`), где задаётся провайдер и токен.
Напрямую не применяется.

# module: control-plane (Hetzner)

OpenTofu-модуль. Поднимает foreign control plane на Hetzner Cloud: сервер + firewall.
Софт (FreeRADIUS, PostgreSQL, config-api, orchestrator) ставит ansible-роль
`control-plane`. Конфигурация провайдера `hcloud` — в окружении, не в модуле.

## Inputs

| Переменная        | Тип            | Default      | Назначение |
|-------------------|----------------|--------------|------------|
| `name`            | string         | `control`    | имя узла |
| `location`        | string         | `fsn1`       | локация (control plane — стабильная юрисдикция DE/FI) |
| `server_type`     | string         | `cx22`       | тип сервера |
| `image`           | string         | `debian-12`  | образ ОС |
| `ssh_key_names`   | list(string)   | —            | имена SSH-ключей (обязательно) |
| `admin_ssh_cidrs` | list(string)   | —            | CIDR для SSH (обязательно) |
| `api_port`        | number         | `443`        | порт config-api (HTTPS) |
| `labels`          | map(string)    | `{}`         | доп. метки |

## Outputs

`id`, `public_ipv4`, `public_ipv6`, `name`, `location`.

## Firewall

Снаружи открыты только `api_port` (config-api HTTPS, вебхук Plati) и SSH (с
`admin_ssh_cidrs`). RADIUS/PostgreSQL наружу не выставлены — слушают локально/в
приватной сети.

## Применение

Вызывается из окружения (`environments/mmvp`). Напрямую не применяется.

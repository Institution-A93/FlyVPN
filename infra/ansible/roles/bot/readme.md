# role: bot

Ansible-роль. Разворачивает Telegram-бот (`services/bot`, Python/aiogram — ADR-0019)
на control plane. Бот автономен (long-polling, in-memory mock) — на старте ничего из
control plane не трогает.

Реализовано:
- **Python-рантайм** (`python3`, `python3-venv`) — только для бота (исключение из Go,
  ADR-0019).
- Системный пользователь `flyvpnbot` (nologin), код в `/opt/flyvpn-bot`, venv там же,
  зависимости из `requirements.txt`.
- **systemd-юнит `flyvpn-bot`** — `python -m bot`, long-polling, автозапуск, рестарт
  при сбое. Хардненинг (`ProtectSystem=strict`, `NoNewPrivileges`, `PrivateTmp`).
- **EnvironmentFile** `/etc/flyvpn-bot/bot.env` из vault; `BOT_TOKEN` обязателен —
  роль падает без него (гейт).

## Хостинг
Control plane (Hetzner), не egress — камуфляж egress (ADR-0011), производительность
data-plane, blast-radius. Long-polling: входящих публичных портов нет; `/notify`
слушает только `127.0.0.1`.

## Секреты (vault, ADR-0012)
`bot_token` (обязателен) и `bot_notify_secret` — в `group_vars/control/zz_bot.yml`
(на деплое из GitHub-секрета `CONTROL_BOT`; в репозитории только `*.example`).
В `site.yml` роль применяется только при заданном `bot_token`
(`when: bot_token | length > 0`) — без секрета бот пропускается, деплой control plane
не падает. При прямом запуске роли без токена срабатывает `assert`.

## Ключевые переменные (defaults)
`bot_user` (`flyvpnbot`), `bot_dir` (`/opt/flyvpn-bot`), `bot_cabinet_url`,
`bot_notify_host`/`bot_notify_port` (`127.0.0.1:8081`), `bot_vpn_remote`.

## Порты
Входящих публичных нет (long-polling). `/notify` — только localhost; в TF-firewall
открывать ничего не нужно.

## Зависимости
`services/bot` (исходники бота). Применяется в плее `control` (см. `site.yml`).

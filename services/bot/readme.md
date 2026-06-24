# service: bot

Telegram-бот FLY VPN — онбординг + уведомления. **Python/aiogram** (исключение из
Go-инварианта — ADR-0019). На старте автономен: своя in-memory mock-БД, ничего из
control plane не трогает. Позже выдача профиля переключается на `config-api`.

## Что делает
- `/start` — приветствие («Добро пожаловать на территорию свободного интернета») +
  кнопки: Личный кабинет (ссылка на сайт), Подключить устройство, Мой статус, Поддержка.
- `/connect` (кнопка «Подключить устройство») — отдаёт VPN-профиль `.mobileconfig`
  (IKEv2/EAP-MSCHAPv2) со стабильными кредами. **Транзиторно:** профиль сейчас mock;
  позже `bot/vpnconfig.py` → вызов `config-api` (реальные EAP-креды + живой ingress).
- `/status` — подписка и трафик (mock).
- `POST /notify` (на `NOTIFY_HOST:NOTIFY_PORT`, секрет `NOTIFY_SECRET`) — orchestrator
  пушит сюда события (`welcome`, `traffic_100mb`, `traffic_1gb`, `sub_expiring`,
  `sub_expired`); `/demo` присылает пример себе.

## Запуск (локально)
Python 3.10+.
```sh
cd services/bot
python -m venv .venv && . .venv/bin/activate   # Windows: .\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
cp .env.example .env        # вставить BOT_TOKEN из @BotFather
python -m bot
```
Long-polling: исходящие к Telegram, входящих портов нет (кроме локального `/notify`).

## Деплой
Ansible-роль [`bot`](../../infra/ansible/roles/bot/readme.md) на control plane:
venv + systemd-юнит `flyvpn-bot`, `BOT_TOKEN` из vault. Хостинг — control plane
(не egress: камуфляж/производительность/blast-radius).

## Структура
| Файл | Роль |
|---|---|
| `bot/main.py` | entrypoint — polling + `/notify` |
| `bot/handlers.py` | `/start`, `/connect`, `/status`, `/help`, `/demo`, callbacks |
| `bot/vpnconfig.py` | mock-генератор `.mobileconfig` (→ config-api позже) |
| `bot/notifications.py` | отправка + endpoint `POST /notify` |
| `bot/texts.py` | тексты (RU) + шаблоны уведомлений |
| `bot/keyboards.py` | inline-клавиатуры |
| `bot/mock_data.py` | in-memory store + стабильные EAP-креды |
| `bot/config.py` | конфиг из окружения (.env / EnvironmentFile) |

## TODO (выход за mock)
- `vpnconfig.build_mobileconfig` → вызов `config-api` (provision EAP в FreeRADIUS).
- `mock_data` → чтение подписок/трафика из БД control plane или API.
- (опц.) webhook вместо polling → отдельный VPS + публичный TLS.

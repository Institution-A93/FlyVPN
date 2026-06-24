"""User-facing copy (Russian), matching the FLY VPN designs. HTML parse mode."""
from .mock_data import MockUser


def _fmt_bytes(n: int) -> str:
    gb = 1024 ** 3
    mb = 1024 ** 2
    if n >= gb:
        return f"{n / gb:.1f} ГБ".replace(".0", "")
    return f"{round(n / mb)} МБ"


WELCOME = (
    "<b>👋 Добро пожаловать на территорию свободного интернета!</b>\n\n"
    "FLY VPN маскирует трафик и не мешает банкам и маркетплейсам — "
    "заблокированные сервисы работают на полной скорости.\n\n"
    "Перейди в личный кабинет, чтобы активировать VPN 👇"
)

HELP = (
    "<b>Команды</b>\n"
    "/start — начать и открыть кабинет\n"
    "/connect — получить VPN-профиль\n"
    "/status — моя подписка и трафик\n"
    "/help — помощь\n"
    "/demo — прислать пример уведомления"
)

CONNECT_CAPTION = (
    "<b>Профиль готов 📲</b>\n\n"
    "1. Откройте файл и установите профиль VPN.\n"
    "2. На iPhone: «Настройки» → «Профиль загружен» → «Установить».\n"
    "3. Включите FLY VPN в «Настройки» → «VPN».\n\n"
    "Профиль привязан к вашему аккаунту."
)


def status_text(u: MockUser) -> str:
    used = _fmt_bytes(u.used_bytes)
    limit = _fmt_bytes(u.limit_bytes)
    plan = "Premium" if u.plan == "premium" else "Пробный"
    return (
        f"<b>Личный кабинет</b>\n\n"
        f"Подписка: <b>{plan}</b>\n"
        f"Действует до: <b>{u.expires_at}</b>\n"
        f"Трафик: <b>{used} / {limit}</b>"
    )


# Notification templates — keyed by the orchestrator's notification_events.type.
# Each ends with a nudge to the cabinet; the cabinet button is attached separately.
_NOTIFICATIONS = {
    "welcome": (
        "👋 Добро пожаловать на территорию свободного интернета!\n"
        "Перейди в профиль, чтобы активировать VPN."
    ),
    "traffic_100mb": (
        "⚠️ Осталось 100 МБ трафика. Подключи премиум, чтобы не потерять "
        "доступ к свободному интернету."
    ),
    "traffic_1gb": (
        "ℹ️ Осталось менее 1 ГБ трафика. Ты можешь купить дополнительный пакет, "
        "чтобы не потерять доступ к интернету."
    ),
    "sub_expiring": (
        "⏳ Завтра последний день подписки. Не забудь продлить её, чтобы "
        "сохранить доступ к свободному интернету."
    ),
    "sub_expired": (
        "🚫 Подписка закончилась. Продли её в личном кабинете, чтобы вернуть "
        "доступ к свободному интернету."
    ),
}


def notification_text(kind: str, payload: dict | None = None) -> str | None:
    return _NOTIFICATIONS.get(kind)


NOTIFICATION_TYPES = tuple(_NOTIFICATIONS.keys())

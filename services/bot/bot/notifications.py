"""Outbound notifications + the HTTP endpoint the orchestrator would call.

The orchestrator (per backend-requirements §9, decision #14) pushes
notification_events here; the bot turns each into a Telegram message.
"""
import logging

from aiogram import Bot
from aiohttp import web

from . import keyboards, texts
from .config import settings

log = logging.getLogger(__name__)


async def send_notification(bot: Bot, telegram_id: int, kind: str, payload: dict | None = None) -> bool:
    text = texts.notification_text(kind, payload)
    if text is None:
        log.warning("unknown notification type: %s", kind)
        return False
    try:
        await bot.send_message(telegram_id, text, reply_markup=keyboards.cabinet_button())
        return True
    except Exception as exc:  # user blocked the bot, invalid id, etc.
        log.warning("send_notification failed for %s: %s", telegram_id, exc)
        return False


def make_notify_app(bot: Bot) -> web.Application:
    """aiohttp app exposing POST /notify, guarded by a shared secret."""

    async def handle_notify(request: web.Request) -> web.Response:
        if request.headers.get("X-Notify-Secret") != settings.notify_secret:
            return web.json_response({"error": "unauthorized"}, status=401)
        try:
            data = await request.json()
            telegram_id = int(data["telegram_id"])
            kind = str(data["type"])
        except (ValueError, KeyError, TypeError):
            return web.json_response({"error": "bad request"}, status=400)
        ok = await send_notification(bot, telegram_id, kind, data.get("payload"))
        return web.json_response({"ok": ok}, status=200 if ok else 422)

    async def handle_health(_: web.Request) -> web.Response:
        return web.json_response({"status": "ok"})

    app = web.Application()
    app.router.add_post("/notify", handle_notify)
    app.router.add_get("/healthz", handle_health)
    return app

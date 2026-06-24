"""Entry point: long-polling bot + the /notify HTTP endpoint, side by side."""
import logging

from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode
from aiogram.types import BotCommand
from aiohttp import web

from .config import settings
from .handlers import router
from .notifications import make_notify_app

COMMANDS = [
    BotCommand(command="start", description="Начать и открыть кабинет"),
    BotCommand(command="connect", description="Получить VPN-профиль"),
    BotCommand(command="status", description="Моя подписка и трафик"),
    BotCommand(command="help", description="Помощь"),
    BotCommand(command="demo", description="Пример уведомления"),
]


async def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")

    if not settings.bot_token:
        raise SystemExit("BOT_TOKEN is empty — copy .env.example to .env and paste your @BotFather token.")

    bot = Bot(settings.bot_token, default=DefaultBotProperties(parse_mode=ParseMode.HTML))
    dp = Dispatcher()
    dp.include_router(router)
    await bot.set_my_commands(COMMANDS)

    # Start the notification HTTP endpoint alongside polling.
    runner = web.AppRunner(make_notify_app(bot))
    await runner.setup()
    site = web.TCPSite(runner, settings.notify_host, settings.notify_port)
    await site.start()
    logging.info("notify endpoint: http://%s:%s/notify", settings.notify_host, settings.notify_port)

    me = await bot.get_me()
    logging.info("bot @%s is up - open Telegram and send /start", me.username)
    try:
        await dp.start_polling(bot)
    finally:
        await runner.cleanup()
        await bot.session.close()

"""Command and callback handlers."""
from aiogram import F, Router
from aiogram.filters import Command, CommandStart
from aiogram.types import BufferedInputFile, CallbackQuery, Message

from . import keyboards, mock_data, texts
from .notifications import send_notification
from .vpnconfig import build_mobileconfig

router = Router()


async def _send_config(message: Message, telegram_id: int, username: str | None) -> None:
    user_name, password = mock_data.issue_credential(telegram_id, username)
    profile = build_mobileconfig(user_name, password)
    document = BufferedInputFile(profile, filename="flyvpn.mobileconfig")
    await message.answer_document(document, caption=texts.CONNECT_CAPTION)


@router.message(CommandStart())
async def cmd_start(message: Message) -> None:
    mock_data.ensure_user(message.from_user.id, message.from_user.username)
    await message.answer(texts.WELCOME, reply_markup=keyboards.main_menu())


@router.message(Command("status"))
async def cmd_status(message: Message) -> None:
    user = mock_data.ensure_user(message.from_user.id, message.from_user.username)
    await message.answer(texts.status_text(user), reply_markup=keyboards.main_menu())


@router.message(Command("connect"))
async def cmd_connect(message: Message) -> None:
    await _send_config(message, message.from_user.id, message.from_user.username)


@router.message(Command("help"))
async def cmd_help(message: Message) -> None:
    await message.answer(texts.HELP, reply_markup=keyboards.main_menu())


@router.message(Command("demo"))
async def cmd_demo(message: Message) -> None:
    """Send a sample notification to yourself — lets us test without the backend."""
    await send_notification(message.bot, message.from_user.id, "traffic_1gb")


@router.callback_query(F.data == "connect")
async def cb_connect(callback: CallbackQuery) -> None:
    await callback.answer("Готовим профиль…")
    await _send_config(callback.message, callback.from_user.id, callback.from_user.username)


@router.callback_query(F.data == "status")
async def cb_status(callback: CallbackQuery) -> None:
    user = mock_data.ensure_user(callback.from_user.id, callback.from_user.username)
    await callback.message.answer(texts.status_text(user), reply_markup=keyboards.main_menu())
    await callback.answer()

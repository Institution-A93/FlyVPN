"""Inline keyboards."""
from aiogram.types import InlineKeyboardMarkup
from aiogram.utils.keyboard import InlineKeyboardBuilder

from .config import settings


def main_menu() -> InlineKeyboardMarkup:
    kb = InlineKeyboardBuilder()
    kb.button(text="📲 Подключить устройство", callback_data="connect")
    kb.button(text="🚀 Личный кабинет", url=settings.cabinet_url)
    kb.button(text="📊 Мой статус", callback_data="status")
    kb.button(text="💬 Поддержка", url=settings.support_url)
    kb.adjust(1)
    return kb.as_markup()


def cabinet_button() -> InlineKeyboardMarkup:
    """Attached to notifications — single deep link to the cabinet."""
    kb = InlineKeyboardBuilder()
    kb.button(text="Открыть личный кабинет", url=settings.cabinet_url)
    return kb.as_markup()

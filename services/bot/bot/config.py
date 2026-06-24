"""Configuration loaded from environment (.env)."""
import os
from dataclasses import dataclass

from dotenv import load_dotenv

load_dotenv()


@dataclass(frozen=True)
class Settings:
    bot_token: str = os.getenv("BOT_TOKEN", "")
    cabinet_url: str = os.getenv("CABINET_URL", "https://flynet.pro/login")
    support_url: str = os.getenv("SUPPORT_URL", "https://t.me/flyvpn_support")
    notify_secret: str = os.getenv("NOTIFY_SECRET", "dev-secret")
    notify_host: str = os.getenv("NOTIFY_HOST", "127.0.0.1")
    notify_port: int = int(os.getenv("NOTIFY_PORT", "8081"))

    # VPN profile (mock until wired to config-api).
    brand_name: str = os.getenv("BRAND_NAME", "FLY VPN")
    vpn_remote: str = os.getenv("VPN_REMOTE", "gw.flynet.pro")
    vpn_remote_id: str = os.getenv("VPN_REMOTE_ID", "gw.flynet.pro")


settings = Settings()

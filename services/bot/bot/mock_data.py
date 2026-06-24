"""In-memory mock store. Stands in for the orchestrator/account-api until it exists."""
import secrets
from dataclasses import dataclass

_GB = 1024 ** 3


@dataclass
class MockUser:
    telegram_id: int
    username: str | None = None
    plan: str = "premium"  # 'trial' | 'premium'
    expires_at: str = "23.04.2027"
    used_bytes: int = 34 * _GB
    limit_bytes: int = 100 * _GB
    # Stable EAP credentials, issued once (decision: stable creds).
    eap_username: str | None = None
    eap_password: str | None = None


_users: dict[int, MockUser] = {}


def ensure_user(telegram_id: int, username: str | None) -> MockUser:
    u = _users.get(telegram_id)
    if u is None:
        u = MockUser(telegram_id=telegram_id, username=username)
        _users[telegram_id] = u
    elif username and u.username != username:
        u.username = username
    return u


def get_user(telegram_id: int) -> MockUser:
    return _users.get(telegram_id) or ensure_user(telegram_id, None)


def issue_credential(telegram_id: int, username: str | None = None) -> tuple[str, str]:
    """Return the user's stable EAP credentials, generating them on first use.

    Later: replace with a call to config-api, which provisions the credential in
    FreeRADIUS and returns username + password.
    """
    u = ensure_user(telegram_id, username)
    if not u.eap_username or not u.eap_password:
        u.eap_username = "fly_" + secrets.token_hex(5)
        u.eap_password = secrets.token_urlsafe(12)
    return u.eap_username, u.eap_password

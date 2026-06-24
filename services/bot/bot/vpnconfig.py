"""Mock VPN profile generation.

Transitionary: produces an Apple .mobileconfig (IKEv2 / EAP-MSCHAPv2) with mock
credentials, mirroring the real `config-api` output. Later, replace
`build_mobileconfig` with a call to config-api so creds are provisioned in
FreeRADIUS and the profile points at a live ingress node.
"""
import uuid
from xml.sax.saxutils import escape

from .config import settings

_TEMPLATE = """<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadDisplayName</key><string>{brand}</string>
  <key>PayloadIdentifier</key><string>{profile_id}</string>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadUUID</key><string>{profile_uuid}</string>
  <key>PayloadVersion</key><integer>1</integer>
  <key>PayloadOrganization</key><string>{brand}</string>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key><string>com.apple.vpn.managed</string>
      <key>PayloadIdentifier</key><string>{profile_id}.vpn</string>
      <key>PayloadUUID</key><string>{vpn_uuid}</string>
      <key>PayloadVersion</key><integer>1</integer>
      <key>PayloadDisplayName</key><string>{brand}</string>
      <key>UserDefinedName</key><string>{brand}</string>
      <key>VPNType</key><string>IKEv2</string>
      <key>IKEv2</key>
      <dict>
        <key>RemoteAddress</key><string>{remote}</string>
        <key>RemoteIdentifier</key><string>{remote_id}</string>
        <key>LocalIdentifier</key><string>{username}</string>
        <key>AuthenticationMethod</key><string>None</string>
        <key>ExtendedAuthEnabled</key><integer>1</integer>
        <key>AuthName</key><string>{username}</string>
        <key>AuthPassword</key><string>{password}</string>
        <key>EAP</key><true/>
      </dict>
    </dict>
  </array>
</dict>
</plist>
"""


def build_mobileconfig(username: str, password: str) -> bytes:
    """Render a .mobileconfig for the given EAP credentials."""
    xml = _TEMPLATE.format(
        brand=escape(settings.brand_name),
        profile_id=f"net.flyvpn.profile.{uuid.uuid4()}",
        profile_uuid=str(uuid.uuid4()),
        vpn_uuid=str(uuid.uuid4()),
        remote=escape(settings.vpn_remote),
        remote_id=escape(settings.vpn_remote_id),
        username=escape(username),
        password=escape(password),
    )
    return xml.encode("utf-8")

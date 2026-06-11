# role: common

Базовая настройка, применяется ко всем узлам перед ролевой настройкой.

- **SSH-харданинг** (вариант A, ADR-0016): только ключи — `PasswordAuthentication no`,
  `PermitRootLogin prohibit-password`, `KbdInteractiveAuthentication no`.
- **fail2ban**: jail `sshd` (systemd backend), бан после 5 неудач на 1ч.

Порт 22 при этом открыт (firewall `0.0.0.0/0`) — защита держится на ключе + fail2ban.
unattended-upgrades по решению оператора не ставим (патчи вручную). WireGuard (вариант B)
— возможное ужесточение на потом.

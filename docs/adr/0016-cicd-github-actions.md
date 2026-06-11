# ADR-0016: CI/CD на GitHub Actions (hosted), remote state, доступ к узлам

- Статус: accepted
- Дата: 2026-06-11

## Контекст
Оператор не хочет применять инфру руками — нужен GitOps. Репозиторий публичный (AGPL).
Деплой = OpenTofu (API) + Ansible (SSH).

## Решение
1. **CI — GitHub Actions, hosted-раннеры.** Публичный репозиторий → раннеры бесплатны
   без лимита минут; self-hosted не нужен. `plan` на PR, `apply` + Ansible на push в
   защищённую ветку (Environment с обязательным ревью). Форк-PR секретов не получают.
2. **Remote state — Hetzner Object Storage** (S3-совместимый, backend `s3`). Бакет
   создаётся вручную (один раз); ключи доступа — в GitHub Secrets. State не локальный.
3. **Секреты — в GitHub Secrets, в git ничего секретного (даже зашифрованного).**
   Содержимое vault'ов узлов и токены — отдельные секреты; workflow материализует их в
   файлы на раннере на время прогона. Это безопаснее, чем коммитить даже vault-зашифрованное
   в публичный репо.
4. **Доступ к узлам по SSH — вариант A: порт 22 открыт, только по ключу + fail2ban.**
   Hosted-раннер имеет динамический IP из общего пула Actions → IP-allowlist не является
   границей доверия, а WireGuard избыточен для MMVP. Защита — ключ (PasswordAuthentication
   no, PermitRootLogin prohibit-password) + fail2ban. Автопатчи (unattended-upgrades) —
   по решению оператора НЕ включаем (патчим вручную).

## Отвергнутые/отложенные альтернативы
- Self-hosted runner — не нужен при публичном репо (нет лимитов).
- IP-allowlist диапазонов GitHub Actions — пул общий для всех, не граница доверия.
- WireGuard-бастион (вариант B) — отложен; кандидат на ужесточение ingress в Selectel-фазе.

## Следствия
- `infra/terraform/.../backend.tf` — s3 на Object Storage; `admin_ssh_cidrs` по умолчанию
  `0.0.0.0/0`.
- Роль `common`: SSH-харданинг + fail2ban на всех узлах.
- `.github/workflows`: plan/apply + прогон Ansible; inventory из `tofu output`.

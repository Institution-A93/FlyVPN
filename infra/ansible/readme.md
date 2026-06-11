# infra/ansible

Конфигурация поднятых узлов: пакеты, сервисы, шаблоны конфигов. OpenTofu создаёт
машину — Ansible доводит её до рабочего состояния. Запуск идемпотентен, чтобы повтор
прогона на cattle-узле был безопасен.

- [`roles/`](./roles/readme.md) — роли по типам узлов (ingress / egress / control-plane).
- `site.yml` — плейбук-точка входа (прогон ролей по группам inventory).
- `ansible.cfg` — настройки (inventory, roles_path, become).
- `inventory.ini.example` — пример inventory; реальный `inventory.ini` в `.gitignore`
  (IP узлов оперативно чувствительны).
- `group_vars/<group>/vault.yml` — boot-секреты под `ansible-vault` (в `.gitignore`,
  в репозитории только `*.example`; см. ADR-0012).

## Применение
```sh
cd infra/ansible
cp inventory.ini.example inventory.ini                 # вписать IP из tofu output
cp group_vars/egress/vault.yml.example group_vars/egress/vault.yml
ansible-vault encrypt group_vars/egress/vault.yml      # вписать реальные секреты
ansible-playbook site.yml --ask-vault-pass
```

Секреты в плейбуках/репозитории не лежат — на MMVP через vault, на phase 2 их подаёт
оркестратор (контракт переменных роли не меняется).

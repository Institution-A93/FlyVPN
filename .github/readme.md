# .github

GitHub Actions — CI/CD (ADR-0016). Hosted-раннеры, публичный репо.

Workflow'ы — в [`workflows/`](./workflows/readme.md):
- `test.yml` — go build/vet/test сервисов (на PR/push).
- `plan.yml` — `tofu plan` на PR (infra/terraform/**).
- `deploy.yml` — `tofu apply` + Ansible на push в `main` / `workflow_dispatch`.

## Что завести в репозитории (один раз)

### Secrets (Settings → Secrets and variables → Actions → Secrets)
| Секрет                  | Что это |
|-------------------------|---------|
| `HCLOUD_TOKEN`          | Hetzner Cloud API-токен (Read&Write) |
| `HCLOUD_S3_ACCESS_KEY`  | Access key Object Storage (remote state) |
| `HCLOUD_S3_SECRET_KEY`  | Secret key Object Storage |
| `SSH_PRIVATE_KEY`       | Приватный ключ для деплоя (Ansible SSH) |
| `EGRESS_VAULT`          | Содержимое `group_vars/egress/vault.yml` (YAML целиком) |
| `CONTROL_VAULT`         | Содержимое `group_vars/control/vault.yml` (YAML целиком) |

### Variables (там же → Variables)
| Переменная     | Что это |
|----------------|---------|
| `SSH_KEY_NAME` | Имя SSH-ключа, загруженного в Hetzner (для tofu) |

> Секреты в git НЕ коммитятся (даже зашифрованные) — vault'ы материализуются на раннере
> из `EGRESS_VAULT`/`CONTROL_VAULT` на время прогона (ADR-0016). `ingress` подключается в
> Selectel-фазе (добавится `INGRESS_VAULT` + правка `--limit`).

## Предусловия
- Бакет Hetzner Object Storage `flyvpn-tfstate` (регион `fsn1`) создан вручную.
- Защита ветки `main` + Environment `production` с обязательным ревью.
- DNS `api.fly-vpn.net` → IP control-plane (для ACME-серта config-api).

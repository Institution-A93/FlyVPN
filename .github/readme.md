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
| `CONTROL_VAULT`         | Содержимое `group_vars/control/vault.yml` (db-пароли, RADIUS-клиенты) |
| `CONTROL_DIGISELLER`    | Креды Plati/Digiseller (YAML): `cp_digiseller_seller_id`, `cp_digiseller_api_key`, `cp_plan_by_goods`. Опционально — без него `/plati/issue` отдаёт 503 |
| `OPENSTACK_RC`          | RC-блок Selectel/OpenStack (`export OS_*`) для провижна ingress (`deploy-ingress`) |
| `INGRESS_VAULT`         | Содержимое `group_vars/ingress/vault.yml`: Reality-mesh к egress + RADIUS + LE-email |
| `CONTROL_RADIUS`        | `cp_radius_clients` (ingress как RADIUS-клиент: name/ipaddr/secret). Отдельно от `CONTROL_VAULT` |
| `CONTROL_BOT`           | Креды Telegram-бота (YAML): `bot_token`, `bot_notify_secret`. Опционально — без него роль `bot` пропускается (ADR-0019) |
| `CONTROL_ACCOUNT`       | Секреты account-api (YAML): `cp_accountapi_jwt_secret` (обяз. для запуска), опц. `cp_accountapi_platega_*`, `cp_accountapi_twilio_*`, `cp_accountapi_telegram_bot_token`. Без JWT-секрета account-api не активируется; без Platega — purchase/webhook 503 (ADR-0020) |

### Variables (там же → Variables)
| Переменная             | Что это |
|------------------------|---------|
| `SSH_KEY_NAME`         | Имя SSH-ключа, загруженного в Hetzner (для tofu) |
| `RADIUS_CLIENT_CIDRS`  | JSON-список CIDR ingress-узлов для RADIUS-firewall на control (напр. `["203.0.113.5/32"]`). IP RU-узла в git не хранится |

> Секреты в git НЕ коммитятся (даже зашифрованные) — vault'ы материализуются на раннере
> из `EGRESS_VAULT`/`CONTROL_VAULT` на время прогона (ADR-0016). `CONTROL_DIGISELLER`
> кладётся в `group_vars/control/zz_digiseller.yml` (имя `zz_*` сортируется после
> `vault.yml` → перекрывает его). Креды Plati вынесены в отдельный секрет нарочно: GitHub
> не показывает сохранённое значение секрета, и менять их в монолитном `CONTROL_VAULT`
> (с db-паролями) пришлось бы вслепую. `ingress` подключается в Selectel-фазе (добавится
> `INGRESS_VAULT` + правка `--limit`).

## Предусловия
- Бакет Hetzner Object Storage `flyvpn-tfstate` (регион `hel1`) создан вручную.
- Защита ветки `main` + Environment `production` с обязательным ревью.
- DNS `api.fly-vpn.net` → IP control-plane (для ACME-серта config-api).

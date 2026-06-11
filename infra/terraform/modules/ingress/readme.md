# module: ingress (Selectel / OpenStack)

OpenTofu-модуль RU ingress. Selectel — это OpenStack под капотом, поэтому сам сервер
создаётся ресурсами провайдера `openstack` (secgroup/инстанс/floating IP). Софт
(strongSwan, RADIUS-клиент, nftables ASN-split, Reality-клиент) ставит ansible-роль
`ingress`.

## Предусловия (вне модуля)
- Проект Selectel и сервис-пользователь OpenStack (через `selectel`-провайдер или консоль).
- Загруженные в проект: образ Debian 12, keypair (SSH).
- Конфигурация провайдера `openstack` — в окружении, не в модуле.

## Inputs

| Переменная             | Назначение |
|------------------------|------------|
| `name`                 | имя узла |
| `flavor_name`          | тип сервера (уточнить по проекту) |
| `image_name`           | образ ОС (Debian 12) |
| `boot_volume_size`     | размер тома, ГБ |
| `availability_zone`    | зона (ru-1a/…) |
| `external_network_id`  | внешняя сеть для floating IP |
| `key_pair_name`        | OpenStack keypair |
| `admin_ssh_cidrs`      | CIDR для SSH |

## Outputs
`id`, `public_ip` (floating), `name`.

## Firewall (security group)
Снаружи открыты только IKEv2 (UDP 500/4500) и SSH (с `admin_ssh_cidrs`).

## Статус
**Не проверено** против живого Selectel/OpenStack API (нет аккаунта и `tofu` в среде).
Имена флейвора/образа/сети уточняются по проекту. Структура — по реальному пути
(OpenStack-ресурсы), требует `tofu validate`/`plan` при подключении аккаунта.

# environment: mmvp

Root-модуль OpenTofu для окружения MMVP. Сейчас поднимает **egress** на Hetzner
(единственный доступный ресурс). control-plane и ingress добавляются модулями по мере
реализации и появления RU-аккаунта/домена — без переделки egress.

## Предусловия
- Установлен OpenTofu (`tofu`).
- В Hetzner загружен SSH-ключ; его имя укажешь в `ssh_key_names`.
- API-токен Hetzner Cloud.

## Применение
```sh
cd infra/terraform/environments/mmvp
cp terraform.tfvars.example terraform.tfvars   # отредактировать (без токена)
export TF_VAR_hcloud_token=xxxxxxxx            # токен — через окружение, не в файл

tofu init
tofu plan
tofu apply
```

После apply `tofu output` отдаст `egress_public_ipv4` — его дальше скармливаешь
Ansible-роли `egress` (inventory) и регистрируешь в оркестраторе.

## Состояние
На старте — локальный backend (`*.tfstate` рядом, в `.gitignore`, НЕ коммитится).
TODO: вынести в S3-совместимый backend (Hetzner Object Storage) при командной работе.

## Переменные
`hcloud_token` (sensitive), `ssh_key_names`, `admin_ssh_cidrs` — обязательные;
`egress_location`/`egress_server_type`/`egress_reality_port` — с дефолтами.

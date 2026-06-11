# environment: mmvp

Root-модуль OpenTofu для окружения MMVP. Поднимает **egress** и **control plane** на
Hetzner. ingress (Selectel) добавляется отдельным модулем по факту появления
RU-аккаунта/домена — без переделки egress/control-plane.

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

После apply `tofu output` отдаст `egress_public_ipv4` и `control_plane_public_ipv4` —
их дальше скармливаешь Ansible (inventory) и регистрируешь в оркестраторе.

## Состояние
Remote backend **Hetzner Object Storage** (S3-совместимый) — см. `backend.tf`. Бакет
`flyvpn-tfstate` (регион `hel1`) создаётся вручную один раз; ключи доступа — через
`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` (в CI — из GitHub Secrets, ADR-0016).

## Переменные
`hcloud_token` (sensitive, `TF_VAR_hcloud_token`), `ssh_key_names` (`TF_VAR_ssh_key_names`)
— обязательные; `admin_ssh_cidrs` по умолчанию `0.0.0.0/0` (вариант A SSH, ADR-0016);
`egress_location`/`control_plane_location`/прочее — с дефолтами.

# infra/terraform

Провижн облачных ресурсов. Модули — по роли узла, провайдер передаётся параметром,
чтобы одну роль можно было поднимать у разных хостеров.

- [`modules/`](./modules/readme.md) — переиспользуемые модули по ролям.
- [`environments/`](./environments/readme.md) — конкретные окружения (root-модули + бэкенд состояния).

Провайдеры: `hcloud` (Hetzner) для egress/control-plane, `selectel` для ingress.

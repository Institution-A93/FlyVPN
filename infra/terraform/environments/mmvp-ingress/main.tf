# RU ingress на Selectel (ru-3). Значения подтверждены разведкой проекта (os-discover):
#   external-network 966826e6-… — единственная сеть (внешняя); приватную создаём в модуле.
#   зоны ru-3a/ru-3b; образ "Debian 12 (Bookworm) 64-bit"; флейвор SL1.2-4096-32 (2 vCPU/4 ГБ).
module "ingress" {
  source = "../../modules/ingress"

  name                  = "edge-1" # нейтральное имя (RU-сторона не светит назначение)
  flavor_name           = "SL1.2-4096-32"
  image_name            = "Debian 12 (Bookworm) 64-bit"
  availability_zone     = "ru-3a"
  external_network_id   = "966826e6-d301-4bb5-aa13-77a324d15f0d"
  external_network_name = "external-network"

  ssh_public_key  = var.ssh_public_key
  admin_ssh_cidrs = var.admin_ssh_cidrs
}

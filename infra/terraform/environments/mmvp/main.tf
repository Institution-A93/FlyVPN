# Окружение MMVP. Сейчас поднимается только egress (доступен Hetzner-аккаунт).
# control-plane и ingress добавляются модулями по мере их реализации и появления
# RU-аккаунта/домена — без переделки egress.
module "egress" {
  source = "../../modules/egress"

  name            = "egress-mmvp"
  location        = var.egress_location
  server_type     = var.egress_server_type
  ssh_key_names   = var.ssh_key_names
  admin_ssh_cidrs = var.admin_ssh_cidrs
  reality_port    = var.egress_reality_port

  labels = {
    env = "mmvp"
  }
}

# Окружение MMVP на Hetzner: egress + control plane.
# ingress (Selectel) добавляется отдельным модулем по факту появления RU-аккаунта/домена,
# без переделки egress/control-plane.

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

module "control_plane" {
  source = "../../modules/control-plane"

  name            = "control-mmvp"
  location        = var.control_plane_location
  server_type     = var.control_plane_server_type
  ssh_key_names   = var.ssh_key_names
  admin_ssh_cidrs = var.admin_ssh_cidrs

  labels = {
    env = "mmvp"
  }
}

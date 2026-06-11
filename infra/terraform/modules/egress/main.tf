locals {
  labels = merge({
    project = "smart-internet"
    role    = "egress"
    managed = "opentofu"
  }, var.labels)
}

# Firewall: наружу торчит только Reality-порт (443); SSH — лишь с админских CIDR.
# Исходящий трафик не ограничиваем — egress по своей роли ходит в интернет.
resource "hcloud_firewall" "egress" {
  name   = "${var.name}-fw"
  labels = local.labels

  rule {
    description = "VLESS-Reality (маскировка под TLS)"
    direction   = "in"
    protocol    = "tcp"
    port        = tostring(var.reality_port)
    source_ips  = ["0.0.0.0/0", "::/0"]
  }

  rule {
    description = "SSH (admin only)"
    direction   = "in"
    protocol    = "tcp"
    port        = "22"
    source_ips  = var.admin_ssh_cidrs
  }
}

resource "hcloud_server" "egress" {
  name        = var.name
  location    = var.location
  server_type = var.server_type
  image       = var.image
  ssh_keys    = var.ssh_key_names
  labels      = local.labels

  firewall_ids = [hcloud_firewall.egress.id]

  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
}

locals {
  labels = merge({
    project = "smart-internet"
    role    = "control"
    managed = "opentofu"
  }, var.labels)
}

# Публично торчит только config-api (HTTPS, вебхук Plati) и SSH с админских CIDR.
# RADIUS/PostgreSQL слушают локально/в приватной сети — наружу не открыты.
resource "hcloud_firewall" "control" {
  name   = "${var.name}-fw"
  labels = local.labels

  rule {
    description = "config-api HTTPS (вебхук Plati) + ACME TLS-ALPN"
    direction   = "in"
    protocol    = "tcp"
    port        = tostring(var.api_port)
    source_ips  = ["0.0.0.0/0", "::/0"]
  }

  rule {
    description = "ACME HTTP-01 challenge (Let's Encrypt)"
    direction   = "in"
    protocol    = "tcp"
    port        = "80"
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

resource "hcloud_server" "control" {
  name        = var.name
  location    = var.location
  server_type = var.server_type
  image       = var.image
  ssh_keys    = var.ssh_key_names
  labels      = local.labels

  firewall_ids = [hcloud_firewall.control.id]

  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
}

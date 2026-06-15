locals {
  # Метаданные нейтральны (RU-сторона не светит назначение). Никаких project/role/vpn.
  metadata = merge({ managed = "opentofu" }, var.labels)
}

# В проекте Selectel приватной сети нет — создаём свою + подсеть + роутер к внешней сети.
resource "openstack_networking_network_v2" "ingress" {
  name           = "${var.name}-net"
  admin_state_up = true
}

resource "openstack_networking_subnet_v2" "ingress" {
  name            = "${var.name}-subnet"
  network_id      = openstack_networking_network_v2.ingress.id
  cidr            = var.subnet_cidr
  ip_version      = 4
  dns_nameservers = var.dns_nameservers
}

resource "openstack_networking_router_v2" "ingress" {
  name                = "${var.name}-router"
  admin_state_up      = true
  external_network_id = var.external_network_id
}

resource "openstack_networking_router_interface_v2" "ingress" {
  router_id = openstack_networking_router_v2.ingress.id
  subnet_id = openstack_networking_subnet_v2.ingress.id
}

# SSH-keypair заводим из публичного ключа деплоя (приватный — секрет CI, в state не попадает).
resource "openstack_compute_keypair_v2" "ingress" {
  name       = "${var.name}-key"
  public_key = var.ssh_public_key
}

# Security group: снаружи только IKEv2 (UDP 500/4500) и SSH с админских CIDR.
resource "openstack_networking_secgroup_v2" "ingress" {
  name        = "${var.name}-sg"
  description = "edge node: IKEv2 + admin SSH"
}

resource "openstack_networking_secgroup_rule_v2" "ike_500" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "udp"
  port_range_min    = 500
  port_range_max    = 500
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = openstack_networking_secgroup_v2.ingress.id
}

resource "openstack_networking_secgroup_rule_v2" "ike_4500" {
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "udp"
  port_range_min    = 4500
  port_range_max    = 4500
  remote_ip_prefix  = "0.0.0.0/0"
  security_group_id = openstack_networking_secgroup_v2.ingress.id
}

resource "openstack_networking_secgroup_rule_v2" "ssh" {
  for_each          = toset(var.admin_ssh_cidrs)
  direction         = "ingress"
  ethertype         = "IPv4"
  protocol          = "tcp"
  port_range_min    = 22
  port_range_max    = 22
  remote_ip_prefix  = each.value
  security_group_id = openstack_networking_secgroup_v2.ingress.id
}

data "openstack_images_image_v2" "os" {
  name        = var.image_name
  most_recent = true
}

# Явный neutron-порт: security group вешается на порт (провайдер v3), к нему же — floating IP.
resource "openstack_networking_port_v2" "ingress" {
  name               = "${var.name}-port"
  network_id         = openstack_networking_network_v2.ingress.id
  admin_state_up     = true
  security_group_ids = [openstack_networking_secgroup_v2.ingress.id]

  fixed_ip {
    subnet_id = openstack_networking_subnet_v2.ingress.id
  }

  depends_on = [openstack_networking_router_interface_v2.ingress]
}

# Инстанс грузится с локального диска флейвора (фикс-линейки SL/VDS включают диск),
# поэтому без отдельного тома.
resource "openstack_compute_instance_v2" "ingress" {
  name              = var.name
  flavor_name       = var.flavor_name
  image_id          = data.openstack_images_image_v2.os.id
  key_pair          = openstack_compute_keypair_v2.ingress.name
  availability_zone = var.availability_zone != "" ? var.availability_zone : null
  metadata          = local.metadata

  network {
    port = openstack_networking_port_v2.ingress.id
  }
}

# Публичный адрес (floating IP из внешней сети) → на порт инстанса.
resource "openstack_networking_floatingip_v2" "ingress" {
  pool = var.external_network_name
}

resource "openstack_networking_floatingip_associate_v2" "ingress" {
  floating_ip = openstack_networking_floatingip_v2.ingress.address
  port_id     = openstack_networking_port_v2.ingress.id
}

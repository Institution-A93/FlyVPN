locals {
  metadata = merge({
    project = "smart-internet"
    role    = "ingress"
    managed = "opentofu"
  }, var.labels)
}

# Security group: снаружи только IKEv2 (UDP 500/4500) и SSH с админских CIDR.
resource "openstack_networking_secgroup_v2" "ingress" {
  name        = "${var.name}-sg"
  description = "RU ingress: IKEv2 + admin SSH"
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

resource "openstack_compute_instance_v2" "ingress" {
  name              = var.name
  flavor_name       = var.flavor_name
  availability_zone = var.availability_zone
  key_pair          = var.key_pair_name
  metadata          = local.metadata
  security_groups   = [openstack_networking_secgroup_v2.ingress.name]

  block_device {
    uuid                  = data.openstack_images_image_v2.os.id
    source_type           = "image"
    destination_type      = "volume"
    volume_size           = var.boot_volume_size
    boot_index            = 0
    delete_on_termination = true
  }

  network {
    name = "private" # приватная сеть проекта; floating IP даёт публичный адрес
  }
}

data "openstack_images_image_v2" "os" {
  name        = var.image_name
  most_recent = true
}

# Публичный адрес.
resource "openstack_networking_floatingip_v2" "ingress" {
  pool = var.external_network_id
}

resource "openstack_compute_floatingip_associate_v2" "ingress" {
  floating_ip = openstack_networking_floatingip_v2.ingress.address
  instance_id = openstack_compute_instance_v2.ingress.id
}

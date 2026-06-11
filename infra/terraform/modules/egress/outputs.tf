# Outputs достаточны для регистрации узла в оркестраторе и для inventory Ansible.
output "id" {
  description = "ID сервера в Hetzner."
  value       = hcloud_server.egress.id
}

output "public_ipv4" {
  description = "Публичный IPv4 egress-узла."
  value       = hcloud_server.egress.ipv4_address
}

output "public_ipv6" {
  description = "Публичный IPv6 egress-узла."
  value       = hcloud_server.egress.ipv6_address
}

output "name" {
  description = "Имя узла."
  value       = hcloud_server.egress.name
}

output "location" {
  description = "Локация узла."
  value       = hcloud_server.egress.location
}

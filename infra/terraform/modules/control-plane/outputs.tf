output "id" {
  description = "ID сервера в Hetzner."
  value       = hcloud_server.control.id
}

output "public_ipv4" {
  description = "Публичный IPv4 control plane."
  value       = hcloud_server.control.ipv4_address
}

output "public_ipv6" {
  description = "Публичный IPv6 control plane."
  value       = hcloud_server.control.ipv6_address
}

output "name" {
  description = "Имя узла."
  value       = hcloud_server.control.name
}

output "location" {
  description = "Локация узла."
  value       = hcloud_server.control.location
}

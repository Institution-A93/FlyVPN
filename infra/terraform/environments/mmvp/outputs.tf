# --- egress ---
output "egress_id" {
  description = "ID egress-сервера в Hetzner."
  value       = module.egress.id
}

output "egress_public_ipv4" {
  description = "Публичный IPv4 egress (для Ansible inventory и регистрации в оркестраторе)."
  value       = module.egress.public_ipv4
}

output "egress_public_ipv6" {
  description = "Публичный IPv6 egress."
  value       = module.egress.public_ipv6
}

# --- control plane ---
output "control_plane_id" {
  description = "ID control-plane-сервера в Hetzner."
  value       = module.control_plane.id
}

output "control_plane_public_ipv4" {
  description = "Публичный IPv4 control plane."
  value       = module.control_plane.public_ipv4
}

output "control_plane_public_ipv6" {
  description = "Публичный IPv6 control plane."
  value       = module.control_plane.public_ipv6
}

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

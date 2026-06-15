output "ingress_public_ipv4" {
  description = "Публичный (floating) IPv4 ingress — для DNS vpn.* и RADIUS-firewall на control."
  value       = module.ingress.public_ip
}

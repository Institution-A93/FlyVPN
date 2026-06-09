output "id" {
  description = "ID инстанса."
  value       = openstack_compute_instance_v2.ingress.id
}

output "public_ip" {
  description = "Публичный (floating) IPv4 ingress — для GeoDNS и регистрации в оркестраторе."
  value       = openstack_networking_floatingip_v2.ingress.address
}

output "name" {
  description = "Имя узла."
  value       = openstack_compute_instance_v2.ingress.name
}

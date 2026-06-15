variable "name" {
  description = "Имя узла (нейтральное — RU-сторона не светит назначение)."
  type        = string
  default     = "edge-1"
}

variable "flavor_name" {
  description = "Flavor (тип) сервера Selectel/OpenStack (напр. SL1.2-4096-32)."
  type        = string
  default     = "SL1.2-4096-32" # 2 vCPU / 4 ГБ / 32 ГБ
}

variable "image_name" {
  description = "Имя образа ОС в проекте (точное имя из `openstack image list`)."
  type        = string
  default     = "Debian 12 (Bookworm) 64-bit"
}

variable "availability_zone" {
  description = "Зона доступности (напр. ru-3a). Пусто = планировщик выбирает сам (надёжнее для local-disk флейворов)."
  type        = string
  default     = ""
}

variable "external_network_id" {
  description = "ID внешней сети (gateway роутера + источник floating IP)."
  type        = string
}

variable "external_network_name" {
  description = "Имя внешней сети (pool для floating IP)."
  type        = string
  default     = "external-network"
}

variable "subnet_cidr" {
  description = "CIDR приватной подсети узла."
  type        = string
  default     = "192.168.100.0/24"
}

variable "dns_nameservers" {
  description = "DNS-резолверы подсети."
  type        = list(string)
  default     = ["1.1.1.1", "8.8.8.8"]
}

variable "ssh_public_key" {
  description = "Публичный SSH-ключ деплоя (для OpenStack keypair)."
  type        = string
}

variable "admin_ssh_cidrs" {
  description = "CIDR, с которых разрешён SSH (22/tcp)."
  type        = list(string)
}

variable "labels" {
  description = "Доп. метаданные инстанса (нейтральные)."
  type        = map(string)
  default     = {}
}

variable "name" {
  description = "Имя RU ingress-узла."
  type        = string
  default     = "ingress"
}

variable "flavor_name" {
  description = "Flavor (тип) сервера в проекте Selectel/OpenStack."
  type        = string
  default     = "SL1.1-2048" # пример; уточнить по доступным флейворам проекта
}

variable "image_name" {
  description = "Имя образа ОС (Debian 12) в проекте."
  type        = string
  default     = "Debian 12 64-bit"
}

variable "boot_volume_size" {
  description = "Размер загрузочного тома, ГБ."
  type        = number
  default     = 20
}

variable "availability_zone" {
  description = "Зона доступности (напр. ru-1a / ru-7a)."
  type        = string
}

variable "external_network_id" {
  description = "ID внешней сети для floating IP (публичный адрес)."
  type        = string
}

variable "key_pair_name" {
  description = "Имя OpenStack keypair (SSH-ключ) для доступа."
  type        = string
}

variable "admin_ssh_cidrs" {
  description = "CIDR, с которых разрешён SSH (22/tcp)."
  type        = list(string)
}

variable "labels" {
  description = "Метаданные на инстанс."
  type        = map(string)
  default     = {}
}

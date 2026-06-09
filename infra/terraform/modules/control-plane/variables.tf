variable "name" {
  description = "Имя узла control plane."
  type        = string
  default     = "control"
}

variable "location" {
  description = "Локация Hetzner. Control plane — в стабильной юрисдикции (DE/FI)."
  type        = string
  default     = "fsn1" # Falkenstein, DE

  validation {
    condition     = contains(["nbg1", "fsn1", "hel1", "ash", "hil"], var.location)
    error_message = "location должна быть валидной Hetzner-локацией (nbg1/fsn1/hel1/ash/hil)."
  }
}

variable "server_type" {
  description = "Тип сервера Hetzner."
  type        = string
  default     = "cx22"
}

variable "image" {
  description = "Базовый образ ОС."
  type        = string
  default     = "debian-12"
}

variable "ssh_key_names" {
  description = "Имена SSH-ключей в Hetzner для доступа к узлу."
  type        = list(string)
}

variable "admin_ssh_cidrs" {
  description = "CIDR, с которых разрешён SSH (22/tcp)."
  type        = list(string)
}

variable "api_port" {
  description = "Порт config-api (HTTPS, принимает вебхук Plati)."
  type        = number
  default     = 443
}

variable "labels" {
  description = "Дополнительные метки на ресурсы."
  type        = map(string)
  default     = {}
}

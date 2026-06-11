variable "name" {
  description = "Имя узла egress (используется в имени сервера и метках)."
  type        = string
  default     = "egress"
}

variable "location" {
  description = "Локация Hetzner. Выбор зависит от latency до аудитории и регулировки трафика (nl/de/fi)."
  type        = string
  default     = "hel1" # Helsinki

  validation {
    condition     = contains(["nbg1", "fsn1", "hel1", "ash", "hil"], var.location)
    error_message = "location должна быть валидной Hetzner-локацией (nbg1/fsn1/hel1/ash/hil)."
  }
}

variable "server_type" {
  description = "Тип сервера Hetzner. На MMVP достаточно небольшого узла."
  type        = string
  default     = "ccx13"
}

variable "image" {
  description = "Базовый образ ОС."
  type        = string
  default     = "debian-12"
}

variable "ssh_key_names" {
  description = "Имена уже загруженных в Hetzner SSH-ключей для доступа к узлу."
  type        = list(string)
}

variable "admin_ssh_cidrs" {
  description = "С каких CIDR разрешён SSH (22/tcp). Не оставлять 0.0.0.0/0 в проде."
  type        = list(string)
}

variable "reality_port" {
  description = "Порт VLESS-Reality сервера (маскируется под TLS)."
  type        = number
  default     = 443
}

variable "labels" {
  description = "Дополнительные метки на ресурсы."
  type        = map(string)
  default     = {}
}

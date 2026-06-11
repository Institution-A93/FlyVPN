variable "hcloud_token" {
  description = "API-токен Hetzner Cloud. Задавать через TF_VAR_hcloud_token, не в коде."
  type        = string
  sensitive   = true
}

variable "ssh_key_names" {
  description = "Имена SSH-ключей, уже загруженных в Hetzner, для доступа к узлам."
  type        = list(string)
}

variable "admin_ssh_cidrs" {
  description = "CIDR, с которых разрешён SSH. Вариант A (ADR-0016): открыто всем, защита — только ключ + fail2ban."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

# --- egress ---
variable "egress_location" {
  description = "Локация egress-узла (Hetzner)."
  type        = string
  default     = "hel1"
}

variable "egress_server_type" {
  description = "Тип сервера egress."
  type        = string
  default     = "cax11" # ARM, 2 vCPU/4GB (CPX/CX сняты с продажи в EU)
}

variable "egress_reality_port" {
  description = "Порт VLESS-Reality на egress."
  type        = number
  default     = 443
}

# --- control plane ---
variable "control_plane_location" {
  description = "Локация control plane (стабильная юрисдикция, DE/FI)."
  type        = string
  default     = "fsn1"
}

variable "control_plane_server_type" {
  description = "Тип сервера control plane."
  type        = string
  default     = "cax21" # ARM, 4 vCPU/8GB
}

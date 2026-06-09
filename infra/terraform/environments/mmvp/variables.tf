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
  description = "CIDR, с которых разрешён SSH к узлам. Не использовать 0.0.0.0/0."
  type        = list(string)
}

variable "egress_location" {
  description = "Локация egress-узла (Hetzner)."
  type        = string
  default     = "hel1"
}

variable "egress_server_type" {
  description = "Тип сервера egress."
  type        = string
  default     = "cx22"
}

variable "egress_reality_port" {
  description = "Порт VLESS-Reality на egress."
  type        = number
  default     = 443
}

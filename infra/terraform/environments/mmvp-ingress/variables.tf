variable "ssh_public_key" {
  description = "Публичный SSH-ключ деплоя (CI вычисляет из приватного ключа-секрета)."
  type        = string
}

variable "admin_ssh_cidrs" {
  description = "CIDR, с которых разрешён SSH на ingress."
  type        = list(string)
  default     = ["0.0.0.0/0"] # MMVP: key-only + fail2ban (ADR-0016)
}

# Модуль объявляет только требования к провайдеру; конфигурация провайдера — в окружении.
terraform {
  required_version = ">= 1.6"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.48"
    }
  }
}

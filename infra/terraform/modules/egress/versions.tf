# Модуль объявляет только требования к провайдеру; конфигурация провайдера
# (токен и т.п.) задаётся в окружении (environments/<env>), не здесь.
terraform {
  required_version = ">= 1.6"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.48"
    }
  }
}

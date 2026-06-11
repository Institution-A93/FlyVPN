# Selectel Cloud Servers создаются через OpenStack API (Selectel = OpenStack под капотом).
# Провайдер настраивается в окружении после создания проекта/сервис-юзера Selectel
# (через selectel-провайдер или консоль). Модуль объявляет только требования.
terraform {
  required_version = ">= 1.6"

  required_providers {
    openstack = {
      source  = "terraform-provider-openstack/openstack"
      version = "~> 3.0"
    }
  }
}

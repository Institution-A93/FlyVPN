terraform {
  required_version = ">= 1.6"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.48"
    }
  }

  # Состояние: на старте — локальный backend (один оператор).
  # TODO: вынести в S3-совместимый backend (Hetzner Object Storage) при командной
  # работе. *.tfstate в .gitignore и НЕ коммитится.
}

# Remote state в Hetzner Object Storage (S3-совместимый). Бакет создаётся вручную в
# консоли Hetzner (имя/регион ниже), ключи доступа — через окружение CI:
#   AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY (из GitHub Secrets).
# Если регион/имя бакета другие — поправь endpoint/bucket здесь.
terraform {
  backend "s3" {
    bucket = "flyvpn-tfstate"
    key    = "mmvp/terraform.tfstate"
    region = "fsn1"

    endpoints = {
      s3 = "https://fsn1.your-objectstorage.com"
    }

    # Hetzner Object Storage ≠ AWS — отключаем AWS-специфичные проверки.
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    use_path_style              = true
  }
}

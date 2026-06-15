# Remote state в Hetzner Object Storage (тот же бакет, отдельный ключ — изолирован
# от Hetzner-окружения mmvp). Ключи доступа — AWS_ACCESS_KEY_ID/SECRET из GitHub Secrets.
terraform {
  backend "s3" {
    bucket = "flyvpn-tfstate"
    key    = "mmvp-ingress/terraform.tfstate"
    region = "hel1"

    endpoints = {
      s3 = "https://hel1.your-objectstorage.com"
    }

    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
    use_path_style              = true
  }
}

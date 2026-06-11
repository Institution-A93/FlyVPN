# Токен задаётся через переменную окружения TF_VAR_hcloud_token (предпочтительно)
# либо через terraform.tfvars (в .gitignore). В код секрет не попадает.
provider "hcloud" {
  token = var.hcloud_token
}

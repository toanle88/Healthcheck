variable "location" { type = string }
variable "resource_group_name" { type = string }
variable "environment" { type = string }
variable "subnet_id" { type = string }
variable "acr_id" { type = string }
variable "acr_login_server" { type = string }
variable "keyvault_id" { type = string }
variable "keyvault_uri" { type = string }
variable "api_image" { type = string }
variable "worker_image" { type = string }
variable "web_image" { type = string }
variable "app_version" { type = string }
variable "db_host" { type = string }
variable "db_name" { type = string }
variable "db_user" { type = string }
variable "entra_client_id" {
  type    = string
  default = ""
}
variable "tenant_id" {
  type    = string
  default = ""
}

variable "app_insights_connection_string" {
  type      = string
  default   = ""
  sensitive = true
}

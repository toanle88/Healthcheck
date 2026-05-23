variable "resource_group_name" { type = string }
variable "resource_group_id" { type = string }
variable "environment" { type = string }
variable "project" {
  type    = string
  default = "healthcheck"
}

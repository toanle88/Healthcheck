variable "location" { type = string }
variable "resource_group_name" { type = string }
variable "resource_group_id" { type = string }
variable "environment" { type = string }
variable "github_org_or_user" { type = string }
variable "github_repo_name" { type = string }
variable "tags" {
  type        = map(string)
  description = "A map of tags to apply to all resources"
  default     = {}
}

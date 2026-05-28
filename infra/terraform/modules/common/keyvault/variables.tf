variable "location" { type = string }
variable "resource_group_name" { type = string }
variable "environment" { type = string }
variable "tags" {
  type        = map(string)
  description = "A map of tags to apply to all resources"
  default     = {}
}

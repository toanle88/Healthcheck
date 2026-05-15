variable "location" {
  type        = string
  description = "Azure region"
}

variable "resource_group_name" {
  type        = string
  description = "Resource group name"
}

variable "resource_group_id" {
  type        = string
  description = "Resource group ID"
}

variable "environment" {
  type        = string
  description = "Environment name (dev, prod)"
}

variable "alert_email" {
  type        = string
  description = "Email address for alerts"
  default     = "alerts@example.com"
}

variable "container_app_environment_id" {
  type        = string
  description = "The ID of the Container App Environment for scoping alerts"
}

variable "api_container_app_id" {
  type        = string
  description = "The ID of the API Container App"
}

variable "worker_job_id" {
  type        = string
  description = "The ID of the Worker Container App Job"
}

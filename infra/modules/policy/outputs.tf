output "environment_tag_policy_id" {
  description = "Resource ID of the 'require environment tag' policy definition."
  value       = azurerm_policy_definition.require_environment_tag.id
}

output "project_tag_policy_id" {
  description = "Resource ID of the 'require project tag' policy definition."
  value       = azurerm_policy_definition.require_project_tag.id
}

output "environment_tag_assignment_id" {
  description = "Resource ID of the 'require environment tag' policy assignment."
  value       = azurerm_resource_group_policy_assignment.require_environment_tag.id
}

output "project_tag_assignment_id" {
  description = "Resource ID of the 'require project tag' policy assignment."
  value       = azurerm_resource_group_policy_assignment.require_project_tag.id
}

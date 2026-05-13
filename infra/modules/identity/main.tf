resource "azurerm_user_assigned_identity" "github_actions" {
  name                = "id-github-actions-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
}

resource "azurerm_role_assignment" "allow_github" {
  scope                = var.resource_group_id
  role_definition_name = "Contributor"
  principal_id         = azurerm_user_assigned_identity.github_actions.principal_id
}

output "client_id" {
  value = azurerm_user_assigned_identity.github_actions.client_id
}

output "tenant_id" {
  value = azurerm_user_assigned_identity.github_actions.tenant_id
}

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

resource "azurerm_user_assigned_identity" "apps" {
  name                = "id-healthcheck-apps-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
}

output "client_id" {
  value = azurerm_user_assigned_identity.github_actions.client_id
}

output "tenant_id" {
  value = azurerm_user_assigned_identity.github_actions.tenant_id
}

output "app_identity_id" {
  value = azurerm_user_assigned_identity.apps.id
}

output "app_identity_principal_id" {
  value = azurerm_user_assigned_identity.apps.principal_id
}

output "app_identity_name" {
  value = azurerm_user_assigned_identity.apps.name
}

output "app_identity_client_id" {
  value = azurerm_user_assigned_identity.apps.client_id
}

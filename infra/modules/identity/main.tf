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

resource "azurerm_federated_identity_credential" "github" {
  name      = "fed-github-actions-${var.environment}"
  audience  = ["api://AzureADTokenExchange"]
  issuer    = "https://token.actions.githubusercontent.com"
  
  user_assigned_identity_id = azurerm_user_assigned_identity.github_actions.id
  subject                   = "repo:${var.github_org_or_user}/${var.github_repo_name}:ref:refs/heads/main"
}

output "client_id" {
  value = azurerm_user_assigned_identity.github_actions.client_id
}

output "tenant_id" {
  value = azurerm_user_assigned_identity.github_actions.tenant_id
}

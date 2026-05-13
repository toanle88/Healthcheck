provider "azurerm" {
  features {}
}

variable "github_org" { default = "toanle88" }
variable "github_repo" { default = "Healthcheck" }
variable "location" { default = "East Asia" }

resource "azurerm_resource_group" "bootstrap" {
  name     = "rg-healthcheck-bootstrap"
  location = var.location
}

# 1. THE IDENTITY
resource "azurerm_user_assigned_identity" "github_actions" {
  name                = "id-github-actions-bootstrap"
  location            = azurerm_resource_group.bootstrap.location
  resource_group_name = azurerm_resource_group.bootstrap.name
}

# 2. THE PERMISSIONS (Contributor on the subscription so it can create other RGs)
data "azurerm_subscription" "primary" {}

resource "azurerm_role_assignment" "allow_github" {
  scope                = data.azurerm_subscription.primary.id
  role_definition_name = "Contributor"
  principal_id         = azurerm_user_assigned_identity.github_actions.principal_id
}

# 3. THE FEDERATED CREDENTIALS (The "Badge")
resource "azurerm_federated_identity_credential" "main" {
  name                      = "fed-github-main"
  resource_group_name       = azurerm_resource_group.bootstrap.name
  audience                  = ["api://AzureADTokenExchange"]
  issuer                    = "https://token.actions.githubusercontent.com"
  user_assigned_identity_id = azurerm_user_assigned_identity.github_actions.id
  subject                   = "repo:${var.github_org}/${var.github_repo}:ref:refs/heads/main"
}

resource "azurerm_federated_identity_credential" "manual" {
  name                      = "fed-github-manual"
  resource_group_name       = azurerm_resource_group.bootstrap.name
  audience                  = ["api://AzureADTokenExchange"]
  issuer                    = "https://token.actions.githubusercontent.com"
  user_assigned_identity_id = azurerm_user_assigned_identity.github_actions.id
  subject                   = "repo:${var.github_org}/${var.github_repo}:event:workflow_dispatch"
}

# 4. THE STORAGE (For Terraform Remote State)
resource "random_string" "storage_suffix" {
  length  = 6
  special = false
  upper   = false
}

resource "azurerm_storage_account" "tfstate" {
  name                     = "sthctfstate${random_string.storage_suffix.result}"
  resource_group_name      = azurerm_resource_group.bootstrap.name
  location                 = azurerm_resource_group.bootstrap.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
}

resource "azurerm_storage_container" "tfstate" {
  name                  = "tfstate"
  storage_account_name  = azurerm_storage_account.tfstate.name
  container_access_type = "private"
}

output "AZURE_STORAGE_ACCOUNT" {
  value = azurerm_storage_account.tfstate.name
}

output "AZURE_STORAGE_CONTAINER" {
  value = azurerm_storage_container.tfstate.name
}

output "AZURE_CLIENT_ID" {
  value = azurerm_user_assigned_identity.github_actions.client_id
}

output "AZURE_TENANT_ID" {
  value = azurerm_user_assigned_identity.github_actions.tenant_id
}

output "AZURE_SUBSCRIPTION_ID" {
  value = data.azurerm_subscription.primary.subscription_id
}

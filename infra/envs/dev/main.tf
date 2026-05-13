terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
}

# 1. Variables (Kept from your original code)
variable "github_org_or_user" { default = "toanle88" }
variable "github_repo_name" { default = "healthcheck" }
variable "location" { default = "East Asia" }
variable "environment" { default = "dev" }

# 2. Resource Group (The container for everything)
resource "azurerm_resource_group" "dev" {
  name     = "rg-healthcheck-${var.environment}"
  location = var.location
}

# 3. IDENTITY MODULE (Refactored OIDC)
module "identity" {
  source              = "../../modules/identity"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  resource_group_id   = azurerm_resource_group.dev.id
  environment         = var.environment
  github_org_or_user  = var.github_org_or_user
  github_repo_name    = var.github_repo_name
}

# 4. NETWORK MODULE (Day 6)
module "network" {
  source              = "../../modules/network"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
}

# 5. ACR MODULE (Day 6)
module "acr" {
  source              = "../../modules/acr"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
}

# OUTPUTS for GitHub Actions
output "AZURE_CLIENT_ID" { value = module.identity.client_id }
output "AZURE_TENANT_ID" { value = module.identity.tenant_id }
output "ACR_LOGIN_SERVER" { value = module.acr.login_server }
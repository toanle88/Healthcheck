terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

# 1. Variables
variable "github_org_or_user" { default = "toanle88" }
variable "github_repo_name" { default = "healthcheck" }
variable "location" { default = "East Asia" }
variable "environment" { default = "dev" }
variable "api_image" { default = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" }
variable "worker_image" { default = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" }

# 2. Resource Group
resource "azurerm_resource_group" "dev" {
  name     = "rg-healthcheck-${var.environment}"
  location = var.location
}

# 3. IDENTITY MODULE (OIDC)
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

# 6. POSTGRES MODULE (Day 7)
resource "random_password" "db_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

module "postgres" {
  source              = "../../modules/postgres"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
  vnet_id             = module.network.vnet_id
  subnet_id           = module.network.db_subnet_id
  admin_password      = random_password.db_password.result
}

# 7. KEY VAULT MODULE (Day 7)
module "keyvault" {
  source              = "../../modules/keyvault"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
}

# 8. CONTAINER APPS MODULE (Day 8)
module "containerapp" {
  source              = "../../modules/containerapp"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
  subnet_id           = module.network.apps_subnet_id
  acr_id              = module.acr.id
  acr_login_server    = module.acr.login_server
  keyvault_id         = module.keyvault.id
  api_image           = var.api_image
  worker_image        = var.worker_image
}

# Store the DB password in Key Vault for later use by the App
resource "azurerm_key_vault_secret" "db_password" {
  name         = "database-password"
  value        = random_password.db_password.result
  key_vault_id = module.keyvault.id
  
  # Ensure the role assignment is created before trying to write the secret
  depends_on = [module.keyvault]
}

# OUTPUTS for GitHub Actions
output "AZURE_CLIENT_ID" { value = module.identity.client_id }
output "AZURE_TENANT_ID" { value = module.identity.tenant_id }
output "ACR_LOGIN_SERVER" { value = module.acr.login_server }
output "API_URL" { value = module.containerapp.api_url }
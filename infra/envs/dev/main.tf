terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 4.72.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = ">= 3.0.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
  backend "azurerm" {
    resource_group_name  = "rg-healthcheck-bootstrap"
    container_name       = "tfstate"
    key                  = "dev.terraform.tfstate"
  }
}

provider "azurerm" {
  features {}
  use_oidc = true
}

# No azuread provider needed here - we use a variable for the Client ID

# 1. Variables
variable "github_org_or_user" { default = "toanle88" }
variable "github_repo_name" { default = "Healthcheck" }
variable "location" { default = "East Asia" }
variable "environment" { default = "dev" }
variable "api_image" { default = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" }
variable "worker_image" { default = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" }
variable "web_image" { default = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" }

# 2. Resource Group
resource "azurerm_resource_group" "dev" {
  name     = "rg-healthcheck-${var.environment}"
  location = var.location
}

# 4. NETWORK MODULE (Day 6)
module "network" {
  source              = "../../modules/network"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
}

variable "acr_name" { type = string }

# 5. THE REGISTRY (Linked from Bootstrap)
data "azurerm_container_registry" "main" {
  name                = var.acr_name
  resource_group_name = "rg-healthcheck-bootstrap"
}

# 6. POSTGRES MODULE (Day 7)
resource "random_password" "db_password" {
  length  = 16
  special = false # Remove complex symbols to avoid URL encoding issues
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

# 9. ENTRA ID CONFIG (Clean Split Pattern)
variable "entra_client_id" { 
  type        = string
  description = "The Client ID of the CIAM app registration (managed separately)"
}

variable "ciam_tenant_id" {
  type        = string
  description = "The Tenant ID of the CIAM Sandbox"
  default     = "cea4bf39-5592-4b9c-bed9-0729bbf40cd4"
}

# 8. CONTAINER APPS MODULE (Day 8)
module "containerapp" {
  source              = "../../modules/containerapp"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
  subnet_id           = module.network.apps_subnet_id
  acr_id              = data.azurerm_container_registry.main.id
  acr_login_server    = data.azurerm_container_registry.main.login_server
  keyvault_id         = module.keyvault.id
  keyvault_uri        = module.keyvault.vault_uri
  api_image           = var.api_image
  worker_image        = var.worker_image
  web_image           = var.web_image
  app_version         = var.api_image
  db_host             = module.postgres.host
  db_name             = "healthcheck"
  db_user             = "psqladmin"

  # Entra ID Config for Frontend
  entra_client_id = var.entra_client_id
  tenant_id       = var.ciam_tenant_id
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
output "ACR_LOGIN_SERVER" { value = data.azurerm_container_registry.main.login_server }
output "API_URL" { value = module.containerapp.api_url }
output "WEB_URL" { value = module.containerapp.web_url }
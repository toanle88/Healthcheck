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
    resource_group_name = "rg-healthcheck-bootstrap"
    container_name      = "tfstate"
    key                 = "dev.terraform.tfstate"
    use_azuread_auth    = true
  }
}

provider "azurerm" {
  features {
    resource_group {
      prevent_deletion_if_contains_resources = false
    }
  }
  use_oidc = true
  default_tags {
    tags = {
      environment = var.environment
      project     = "healthcheck"
    }
  }
}

# REFACTORING: Move the identity state from containerapp to identity module
moved {
  from = module.containerapp.azurerm_user_assigned_identity.apps
  to   = module.identity.azurerm_user_assigned_identity.apps
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
variable "migrate_image" { default = "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" }

# 2. Resource Group
resource "azurerm_resource_group" "dev" {
  name     = "rg-healthcheck-${var.environment}"
  location = var.location
}

# 3. IDENTITY MODULE (The "Security Passports")
module "identity" {
  source              = "../../modules/common/identity"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  resource_group_id   = azurerm_resource_group.dev.id
  environment         = var.environment
  github_org_or_user  = var.github_org_or_user
  github_repo_name    = var.github_repo_name
}

# 4. NETWORK MODULE (Day 6)
module "network" {
  source              = "../../modules/common/network"
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
module "postgres" {
  source              = "../../modules/common/postgres"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
  environment         = var.environment
  vnet_id             = module.network.vnet_id
  subnet_id           = module.network.db_subnet_id
  aad_admin_object_id = module.identity.app_identity_principal_id
  aad_admin_name      = module.identity.app_identity_name
}

# 7. KEY VAULT MODULE (Day 7)
module "keyvault" {
  source              = "../../modules/common/keyvault"
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

variable "alert_webhook_url" {
  type        = string
  description = "The Slack/Discord Webhook URL for alerting"
  default     = ""
  sensitive   = true
}

# 8. CONTAINER APPS MODULE (Day 8)
module "containerapp" {
  source              = "../../modules/common/containerapp"
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
  migrate_image       = var.migrate_image
  app_version         = var.api_image
  db_host             = module.postgres.host
  db_name             = "healthcheck"
  db_user             = module.identity.app_identity_name

  # Entra ID Config for Frontend
  entra_client_id = var.entra_client_id
  tenant_id       = var.ciam_tenant_id

  # Alert Webhook Secret reference from Key Vault
  alert_webhook_secret_id = azurerm_key_vault_secret.alert_webhook.id

  # Monitoring
  app_insights_connection_string = module.monitor.app_insights_connection_string

  # Identity for Cloud Auth
  app_identity_id           = module.identity.app_identity_id
  app_identity_principal_id = module.identity.app_identity_principal_id
  app_identity_client_id    = module.identity.app_identity_client_id

  # Ensure the secret is created in Key Vault BEFORE the apps try to mount it
  depends_on = [azurerm_key_vault_secret.alert_webhook]
}

# Store the alert webhook in Key Vault for later use by the App
resource "azurerm_key_vault_secret" "alert_webhook" {
  name            = "alert-webhook-url"
  value           = var.alert_webhook_url == "" ? "dummy" : var.alert_webhook_url
  key_vault_id    = module.keyvault.id
  content_type    = "text/plain"
  expiration_date = "2027-12-31T23:59:59Z"

  # Ensure the role assignment/Key Vault is created before trying to write the secret
  depends_on = [module.keyvault]
}

# 9. MONITORING MODULE (Day 12)
module "monitor" {
  source                       = "../../modules/common/monitor"
  location                     = azurerm_resource_group.dev.location
  resource_group_name          = azurerm_resource_group.dev.name
  resource_group_id            = azurerm_resource_group.dev.id
  environment                  = var.environment
  container_app_environment_id = module.containerapp.container_app_environment_id
  api_container_app_id         = module.containerapp.api_app_id
  worker_job_id                = module.containerapp.worker_job_id
  alert_email                  = "toanle88@outlook.com"
}

# 10. POLICY MODULE — enforce required tags on all resources in this resource group
module "policy" {
  source              = "../../modules/common/policy"
  resource_group_name = azurerm_resource_group.dev.name
  resource_group_id   = azurerm_resource_group.dev.id
  environment         = var.environment
  project             = "healthcheck"
}



# OUTPUTS for GitHub Actions
output "ACR_LOGIN_SERVER" { value = data.azurerm_container_registry.main.login_server }
output "API_URL" { value = module.containerapp.api_url }
output "WEB_URL" { value = module.containerapp.web_url }
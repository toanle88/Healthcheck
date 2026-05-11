# ----------------------------------------------------------------------------------
# 1. VARIABLES
# Change these to match your specific GitHub environment.
# ----------------------------------------------------------------------------------
variable "github_org_or_user" {
  description = "Your GitHub username or organization name"
  type        = string
  default     = "toanle88" 
}

variable "github_repo_name" {
  description = "The name of your GitHub repository"
  type        = string
  default     = "healthcheck"
}

# ----------------------------------------------------------------------------------
# 2. PROVIDER CONFIGURATION
# Tells Terraform to use the Azure Resource Manager (ARM)
# ----------------------------------------------------------------------------------
terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0" # Ensures you use the modern provider version
    }
  }
}

provider "azurerm" {
  features {} # Required block for the Azure provider
}

# ----------------------------------------------------------------------------------
# 3. RESOURCES
# ----------------------------------------------------------------------------------

# Create a Resource Group (The logical container for all your learning resources)
resource "azurerm_resource_group" "dev" {
  name     = "rg-healthcheck-dev"
  location = "East Asia"
}

# Create a User Assigned Managed Identity (The "Service Account" for GitHub)
resource "azurerm_user_assigned_identity" "github_actions" {
  name                = "id-github-actions"
  location            = azurerm_resource_group.dev.location
  resource_group_name = azurerm_resource_group.dev.name
}

# Grant "Contributor" permissions to the Identity over the Resource Group
resource "azurerm_role_assignment" "allow_github" {
  scope                = azurerm_resource_group.dev.id
  role_definition_name = "Contributor"
  principal_id         = azurerm_user_assigned_identity.github_actions.principal_id
}

# FEDERATED CREDENTIAL (The Security Bridge)
# This is what allows GitHub to exchange its OIDC token for an Azure access token.
resource "azurerm_federated_identity_credential" "github" {
  name      = "fed-github-actions"
  audience  = ["api://AzureADTokenExchange"]
  issuer    = "https://token.actions.githubusercontent.com"
  
  # Use this attribute name to resolve the warning and avoid the "Unexpected attribute" error
  user_assigned_identity_id = azurerm_user_assigned_identity.github_actions.id
  
  subject   = "repo:${var.github_org_or_user}/${var.github_repo_name}:ref:refs/heads/main"
}

# ----------------------------------------------------------------------------------
# 4. OUTPUTS
# These values will be printed in your terminal after 'terraform apply'
# Use these as "Secrets" in your GitHub Repository settings.
# ----------------------------------------------------------------------------------
output "AZURE_CLIENT_ID" {
  description = "The Client ID of the Managed Identity"
  value       = azurerm_user_assigned_identity.github_actions.client_id
}

output "AZURE_TENANT_ID" {
  description = "The Tenant ID of your Azure Active Directory"
  value       = azurerm_user_assigned_identity.github_actions.tenant_id
}

# We use a data source to fetch your current subscription ID automatically
data "azurerm_subscription" "current" {}

output "AZURE_SUBSCRIPTION_ID" {
  description = "The Subscription ID where resources are deployed"
  value       = data.azurerm_subscription.current.subscription_id
}
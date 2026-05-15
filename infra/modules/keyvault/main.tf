data "azurerm_client_config" "current" {}

resource "random_string" "kv_suffix" {
  length  = 8
  special = false
  upper   = false
}

resource "azurerm_key_vault" "main" {
  #checkov:skip=CKV2_AZURE_32:Private endpoint not required for learning project
  # Key Vault names must be globally unique and between 3-24 characters.
  # We use a random suffix to avoid soft-delete naming conflicts across redeployments.
  name                        = "kv-hc-${var.environment}-${random_string.kv_suffix.result}"
  location                    = var.location
  resource_group_name         = var.resource_group_name
  enabled_for_disk_encryption = true
  tenant_id                   = data.azurerm_client_config.current.tenant_id

  # Soft delete allows you to recover a deleted vault. 7 days is a safe minimum for dev.
  soft_delete_retention_days = 7
  purge_protection_enabled   = false
  sku_name                   = "standard"

  # Modern standard (v4.0+): Use this instead of enable_rbac_authorization
  rbac_authorization_enabled = true
}

# IMPORTANT: Even if you are an "Owner" of the subscription, you cannot read/write
# secrets in a Key Vault by default if RBAC is enabled. 
# We must explicitly grant you the "Secrets Officer" role so Terraform can upload the DB password.
resource "azurerm_role_assignment" "current_user_secrets" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets Officer"
  principal_id         = data.azurerm_client_config.current.object_id
}

output "id" {
  value = azurerm_key_vault.main.id
}

output "vault_uri" {
  value = azurerm_key_vault.main.vault_uri
}

data "azuread_client_config" "current" {}

resource "azuread_application" "dashboard" {
  display_name     = "Healthcheck Dashboard (${var.environment})"
  owners           = [data.azuread_client_config.current.object_id]
  sign_in_audience = "AzureADMyOrg"

  web {
    redirect_uris = [
      "${var.web_reply_url}/.auth/login/aad/callback",
      "${var.api_reply_url}/.auth/login/aad/callback"
    ]
    implicit_grant {
      access_token_issuance_enabled = false
      id_token_issuance_enabled     = true
    }
  }
}

resource "azuread_service_principal" "dashboard" {
  client_id                    = azuread_application.dashboard.client_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}

resource "azuread_application_password" "dashboard" {
  application_id = azuread_application.dashboard.id
  display_name   = "Dashboard Secret"
  end_date       = "2099-01-01T00:00:00Z"
}

# Store in Key Vault
resource "azurerm_key_vault_secret" "entra_secret" {
  name         = "entra-client-secret"
  value        = azuread_application_password.dashboard.value
  key_vault_id = var.keyvault_id
}

output "client_id" {
  value = azuread_application.dashboard.client_id
}

output "client_secret" {
  value     = azuread_application_password.dashboard.value
  sensitive = true
}

output "tenant_id" {
  value = data.azuread_client_config.current.tenant_id
}

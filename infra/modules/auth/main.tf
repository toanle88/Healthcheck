terraform {
  required_providers {
    azuread = {
      source  = "hashicorp/azuread"
      version = ">= 3.0.0"
    }
  }
}

data "azuread_client_config" "current" {}

resource "azuread_application" "dashboard" {
  display_name     = "Healthcheck Dashboard (${var.environment})"
  owners           = [data.azuread_client_config.current.object_id]
  sign_in_audience = "AzureADMyOrg"

  spa {
    redirect_uris = [
      var.web_reply_url,
      var.api_reply_url,
      "http://localhost:5173"
    ]
  }
}

resource "azuread_service_principal" "dashboard" {
  client_id                    = azuread_application.dashboard.client_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}

output "client_id" {
  value = azuread_application.dashboard.client_id
}

output "tenant_id" {
  value = data.azuread_client_config.current.tenant_id
}

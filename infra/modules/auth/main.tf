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

  identifier_uris = ["api://${azuread_application.dashboard.client_id}"]

  single_page_application {
    redirect_uris = [
      var.web_reply_url,
      var.api_reply_url,
      "http://localhost:5173"
    ]
  }

  api {
    requested_access_token_version = 2
    oauth2_permission_scope {
      admin_consent_description  = "Allow the application to access the Healthcheck API on behalf of the user."
      admin_consent_display_name = "Access Healthcheck API"
      enabled                    = true
      id                         = "da567a5b-9d4d-456d-886d-368735586616" # Just a random UUID
      type                       = "User"
      user_consent_description   = "Allow the application to access the Healthcheck API on your behalf."
      user_consent_display_name  = "Access Healthcheck API"
      value                      = "access_as_user"
    }
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

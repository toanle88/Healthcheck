terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 4.72.0"
    }
  }
}

# 1. LOG ANALYTICS (The Brain for Logs)
resource "azurerm_log_analytics_workspace" "main" {
  name                = "log-healthcheck-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

# 2. CONTAINER APPS ENVIRONMENT (The Cluster)
resource "azurerm_container_app_environment" "main" {
  name                       = "cae-healthcheck-${var.environment}"
  location                   = var.location
  resource_group_name        = var.resource_group_name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id

  # Link to our VNet Subnet from Day 6
  infrastructure_subnet_id = var.subnet_id
}

# 3. MANAGED IDENTITY (The "Security Passport" for the Apps)
resource "azurerm_user_assigned_identity" "apps" {
  name                = "id-healthcheck-apps-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
}

# 4. PERMISSIONS (The "Weld")

# Allow the Apps to pull images from ACR
resource "azurerm_role_assignment" "acr_pull" {
  scope                = var.acr_id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_user_assigned_identity.apps.principal_id
}

# Allow the Apps to read secrets from Key Vault
resource "azurerm_role_assignment" "kv_secrets" {
  scope                = var.keyvault_id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = azurerm_user_assigned_identity.apps.principal_id
}

# 5. THE API (Publicly Accessible)
resource "azurerm_container_app" "api" {
  name                         = "ca-healthcheck-api-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  revision_mode                = "Single"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.apps.id]
  }

  registry {
    server   = var.acr_login_server
    identity = azurerm_user_assigned_identity.apps.id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 8080
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
    cors {
      allowed_origins = ["https://ca-healthcheck-web-${var.environment}.${azurerm_container_app_environment.main.default_domain}"]
      allowed_methods = ["GET", "POST", "OPTIONS"]
      allowed_headers = ["*"]
    }
  }

  secret {
    name                = "db-password"
    key_vault_secret_id = "${var.keyvault_uri}secrets/database-password"
    identity            = azurerm_user_assigned_identity.apps.id
  }

  template {
    container {
      name   = "api"
      image  = var.api_image
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "PORT"
        value = "8080"
      }

      env {
        name  = "ENV"
        value = var.environment
      }

      env {
        name  = "DB_HOST"
        value = var.db_host
      }

      env {
        name  = "DB_NAME"
        value = var.db_name
      }

      env {
        name  = "DB_USER"
        value = var.db_user
      }

      env {
        name        = "DB_PASSWORD"
        secret_name = "db-password"
      }

      env {
        name  = "ENTRA_TENANT_ID"
        value = var.tenant_id
      }

      env {
        name  = "ENTRA_CLIENT_ID"
        value = var.entra_client_id
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].container[0].image
    ]
  }
}

# 6. THE WORKER (Scheduled Job)
resource "azurerm_container_app_job" "worker" {
  name                         = "caj-healthcheck-worker-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  location                     = var.location

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.apps.id]
  }

  registry {
    server   = var.acr_login_server
    identity = azurerm_user_assigned_identity.apps.id
  }

  schedule_trigger_config {
    cron_expression = "*/1 * * * *" # Every minute
  }

  replica_timeout_in_seconds = 60
  replica_retry_limit        = 1

  secret {
    name                = "db-password"
    key_vault_secret_id = "${var.keyvault_uri}secrets/database-password"
    identity            = azurerm_user_assigned_identity.apps.id
  }

  template {
    container {
      name   = "worker"
      image  = var.worker_image
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "WORKER_MODE"
        value = "job"
      }

      env {
        name  = "DB_HOST"
        value = var.db_host
      }

      env {
        name  = "DB_NAME"
        value = var.db_name
      }

      env {
        name  = "DB_USER"
        value = var.db_user
      }

      env {
        name        = "DB_PASSWORD"
        secret_name = "db-password"
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].container[0].image
    ]
  }
}

# 7. THE FRONTEND (Publicly Accessible)
resource "azurerm_container_app" "web" {
  name                         = "ca-healthcheck-web-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  revision_mode                = "Single"

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.apps.id]
  }

  registry {
    server   = var.acr_login_server
    identity = azurerm_user_assigned_identity.apps.id
  }

  ingress {
    allow_insecure_connections = false
    external_enabled           = true
    target_port                = 80
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    container {
      name   = "web"
      image  = var.web_image
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "VITE_API_URL"
        value = "https://${azurerm_container_app.api.ingress[0].fqdn}"
      }

      env {
        name  = "VITE_APP_VERSION"
        value = var.app_version
      }

      env {
        name  = "VITE_ENTRA_CLIENT_ID"
        value = var.entra_client_id
      }

      env {
        name  = "VITE_ENTRA_TENANT_ID"
        value = var.tenant_id
      }

      env {
        name  = "ENV"
        value = var.environment
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].container[0].image
    ]
  }
}

output "default_domain" {
  value = azurerm_container_app_environment.main.default_domain
}

output "api_url" {
  value = azurerm_container_app.api.ingress[0].fqdn
}

output "web_url" {
  value = azurerm_container_app.web.ingress[0].fqdn
}

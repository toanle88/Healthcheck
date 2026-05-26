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

# 3. PERMISSIONS (The "Weld")

# Allow the Apps to pull images from ACR
resource "azurerm_role_assignment" "acr_pull" {
  scope                = var.acr_id
  role_definition_name = "AcrPull"
  principal_id         = var.app_identity_principal_id
}

# Allow the Apps to read secrets from Key Vault
resource "azurerm_role_assignment" "kv_secrets" {
  scope                = var.keyvault_id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = var.app_identity_principal_id
}

# Azure AD Role Assignments take time to propagate. 
# We MUST wait before letting the Container Apps try to pull secrets.
resource "time_sleep" "wait_for_rbac" {
  depends_on      = [azurerm_role_assignment.kv_secrets]
  create_duration = "30s"
}

# 5. THE API (Publicly Accessible)
resource "azurerm_container_app" "api" {
  name                         = "ca-healthcheck-api-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  revision_mode                = "Multiple"

  identity {
    type         = "UserAssigned"
    identity_ids = [var.app_identity_id]
  }

  depends_on = [time_sleep.wait_for_rbac]

  registry {
    server   = var.acr_login_server
    identity = var.app_identity_id
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
      allowed_methods = ["GET", "POST", "DELETE", "OPTIONS"]
      allowed_headers = ["*"]
    }
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
        name  = "AZURE_CLIENT_ID"
        value = var.app_identity_client_id
      }

      env {
        name  = "ENTRA_TENANT_ID"
        value = var.tenant_id
      }

      env {
        name  = "ENTRA_CLIENT_ID"
        value = var.entra_client_id
      }

      env {
        name  = "APPLICATIONINSIGHTS_CONNECTION_STRING"
        value = var.app_insights_connection_string
      }
    }
    min_replicas = 0
    max_replicas = 3
  }

  lifecycle {
    ignore_changes = [
      template[0].container[0].image,
      ingress[0].traffic_weight
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
    identity_ids = [var.app_identity_id]
  }

  dynamic "secret" {
    for_each = var.alert_webhook_secret_id != "" ? [1] : []
    content {
      name                = "alert-webhook"
      key_vault_secret_id = var.alert_webhook_secret_id
      identity            = var.app_identity_id
    }
  }

  depends_on = [time_sleep.wait_for_rbac]

  registry {
    server   = var.acr_login_server
    identity = var.app_identity_id
  }

  schedule_trigger_config {
    cron_expression = "*/1 * * * *" # Every minute
  }

  replica_timeout_in_seconds = 60
  replica_retry_limit        = 1

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
        name  = "AZURE_CLIENT_ID"
        value = var.app_identity_client_id
      }

      env {
        name  = "APPLICATIONINSIGHTS_CONNECTION_STRING"
        value = var.app_insights_connection_string
      }

      dynamic "env" {
        for_each = var.alert_webhook_secret_id != "" ? [1] : []
        content {
          name        = "ALERT_WEBHOOK_URL"
          secret_name = "alert-webhook"
        }
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
  revision_mode                = "Multiple"

  identity {
    type         = "UserAssigned"
    identity_ids = [var.app_identity_id]
  }

  depends_on = [time_sleep.wait_for_rbac]

  registry {
    server   = var.acr_login_server
    identity = var.app_identity_id
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
    min_replicas = 0
    max_replicas = 3
  }

  lifecycle {
    ignore_changes = [
      template[0].container[0].image,
      ingress[0].traffic_weight
    ]
  }
}

# 8. THE MIGRATION JOB (Run manually before container updates)
resource "azurerm_container_app_job" "migrate" {
  name                         = "caj-healthcheck-migrate-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = var.resource_group_name
  location                     = var.location

  identity {
    type         = "UserAssigned"
    identity_ids = [var.app_identity_id]
  }

  depends_on = [time_sleep.wait_for_rbac]

  registry {
    server   = var.acr_login_server
    identity = var.app_identity_id
  }

  manual_trigger_config {
    parallelism              = 1
    replica_completion_count = 1
  }

  replica_timeout_in_seconds = 180
  replica_retry_limit        = 0 # No retry for migration jobs, fail immediately if it fails

  template {
    container {
      name   = "migrate"
      image  = var.migrate_image
      cpu    = 0.25
      memory = "0.5Gi"

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
        name  = "AZURE_CLIENT_ID"
        value = var.app_identity_client_id
      }

      env {
        name  = "APPLICATIONINSIGHTS_CONNECTION_STRING"
        value = var.app_insights_connection_string
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

output "container_app_environment_id" {
  value = azurerm_container_app_environment.main.id
}

output "api_app_id" {
  value = azurerm_container_app.api.id
}

output "worker_job_id" {
  value = azurerm_container_app_job.worker.id
}

output "migrate_job_name" {
  value = azurerm_container_app_job.migrate.name
}


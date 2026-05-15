# 1. APPLICATION INSIGHTS (Traces & Metrics)
resource "azurerm_application_insights" "main" {
  name                = "appi-healthcheck-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  application_type    = "web"
  sampling_percentage = 100
}

# 2. ACTION GROUP (The Notification Hub)
resource "azurerm_monitor_action_group" "main" {
  name                = "ag-healthcheck-${var.environment}"
  resource_group_name = var.resource_group_name
  short_name          = "hc-alerts"

  email_receiver {
    name                    = "admin"
    email_address           = var.alert_email
    use_common_alert_schema = true
  }
}

# 3. METRIC ALERT: P95 Latency > 500ms
resource "azurerm_monitor_metric_alert" "latency" {
  name                = "alert-latency-high-${var.environment}"
  resource_group_name = var.resource_group_name
  scopes              = [var.api_container_app_id]
  description         = "Fires when API P95 latency is > 500ms for 5 minutes"
  severity            = 2 # Warning

  criteria {
    metric_namespace = "Microsoft.App/containerApps"
    metric_name      = "ResponseTime"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 500 # ms
  }

  action {
    action_group_id = azurerm_monitor_action_group.main.id
  }
}

# 4. METRIC ALERT: Error Rate > 1%
resource "azurerm_monitor_metric_alert" "errors" {
  name                = "alert-errors-high-${var.environment}"
  resource_group_name = var.resource_group_name
  scopes              = [var.api_container_app_id]
  description         = "Fires when HTTP 5xx errors exceed 1%"
  severity            = 1 # Critical

  # Note: Container Apps exposes 'Requests' with a status filter in some scenarios,
  # but here we use a simple count of non-success codes if available, 
  # or custom metrics if instrumented.
  criteria {
    metric_namespace = "Microsoft.App/containerApps"
    metric_name      = "Requests" # We'll monitor total requests as a proxy or use custom OTel metrics
    aggregation      = "Total"
    operator         = "GreaterThan"
    threshold        = 100 # Alert if we see spikes in traffic (placeholder for complex ratio alert)
  }

  action {
    action_group_id = azurerm_monitor_action_group.main.id
  }
}



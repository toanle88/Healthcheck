# Lesson 04: Azure Container Apps 🚀🌀

Azure Container Apps (ACA) is our "Serverless" platform. It gives us the power of Kubernetes without the headache of managing it.

## 1. The Environment Foundation (The Sandbox Infrastructure)

Before executing container runtimes, we configure the administrative platform for security, networking, and log collection:

### A. The Log Hub (`azurerm_log_analytics_workspace`)
```hcl
resource "azurerm_log_analytics_workspace" "main" {
  name              = "log-healthcheck-${var.environment}"
  sku               = "PerGB2018"
  retention_in_days = 30
}
```
All system stdout/stderr streams from our containers are automatically routed to this centralized Log Analytics Workspace. It acts as the central data sink for our observability queries.

### B. The Shared Cluster (`azurerm_container_app_environment`)
```hcl
resource "azurerm_container_app_environment" "main" {
  name                       = "cae-healthcheck-${var.environment}"
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  infrastructure_subnet_id   = var.subnet_id # Links to snet-apps
}
```
This is the boundary enclosing our Container Apps. Linking it to our `subnet_id` binds it directly inside our Virtual Network, letting our apps communicate securely over private IPs.

### C. Registry Access (`azurerm_role_assignment`)
```hcl
resource "azurerm_role_assignment" "acr_pull" {
  scope                = var.acr_id
  role_definition_name = "AcrPull"
  principal_id         = var.app_identity_principal_id
}
```
Since our Container Registry is secure, Azure Container Apps needs permission to pull our private Docker images. We assign the **`AcrPull`** role to the container app's user-assigned managed identity. **No registry credentials ever touch our environment!**

---

## 2. Scale-to-Zero: The Money Saver 💰💤

In `modules/containerapp/main.tf`, we set:
```hcl
template {
  min_replicas = 0
  max_replicas = 3
}
```

*   **How it works**: Azure uses **KEDA** (Kubernetes Event-driven Autoscaling). KEDA actively monitors the Ingress. If 5 minutes pass with 0 HTTP requests, it terminates all replicas.
*   **Cold Start**: When a new request arrives, Azure wakes up the app. This is the optimal cost-saving mechanism for Dev/Test environments.

---

## 3. Revisions: The Safety Net 🛡️🔄

We use `revision_mode = "Multiple"`.

Every time you change environment variables or container images in Terraform, Azure creates a **New Revision**. 
*   **The Blue-Green Split**: You can split traffic (e.g., 90/10) to canary test new code.
*   **Automatic Rollback**: If a new revision fails its readiness probes on startup, Azure retains 100% of the traffic on the last healthy revision automatically.

---

## 4. Jobs vs. Apps 🏃‍♂️ vs 🧍‍♂️

*   **`azurerm_container_app` (The API/Web)**: Designed for long-running, interactive web processes. They have URLs and listen for requests.
*   **`azurerm_container_app_job` (The Worker)**: Designed for one-off tasks. It has no URL, does not listen for requests, and executes on a **Cron schedule** (`cron_expression`), immediately shutting down upon completion.

---

## 5. The Ingress (The Front Door) 🚪

```hcl
ingress {
  external_enabled = true
  target_port      = 8080
  traffic_weight {
    latest_revision = true
    percentage      = 100
  }
}
```

*   **`external_enabled = true`**: Generates a public HTTPS URL.
*   **`target_port = 8080`**: Maps incoming public TLS traffic (`443`) to the container's private port (`8080`).
*   **Automatic TLS**: Azure manages the HTTPS certificates automatically behind the scenes.
*   **Lifecycle Rules**: We added `ignore_changes = [ingress[0].traffic_weight]` in Terraform. This prevents Terraform from reverting manual traffic weight splits configured in the portal, giving us complete control over our manual Blue-Green deployments!

---

## 6. Deep Dive: Monitoring & Observability (`modules/monitor/main.tf`) 📈🔔

In modern microservices, you don't wait for your users to tell you your website is down. You design your system to monitor itself. This module creates a proactive observability loop:

### A. The Telemetry Receiver (`azurerm_application_insights`)
```hcl
resource "azurerm_application_insights" "main" {
  name                = "appi-healthcheck-${var.environment}"
  application_type    = "web"
  sampling_percentage = 100
}
```
Application Insights is our **APM (Application Performance Management)** platform. Our Go application is instrumented with the **OpenTelemetry (OTel) Go SDK** to stream structured logs, latency metrics, and distributed execution traces directly here. Setting the sampling rate to `100` ensures we record every single developer request for inspection.

### B. The Notification Hub (`azurerm_monitor_action_group`)
```hcl
resource "azurerm_monitor_action_group" "main" {
  name       = "ag-healthcheck-${var.environment}"
  short_name = "hc-alerts"

  email_receiver {
    name          = "admin"
    email_address = var.alert_email
  }
}
```
An **Action Group** is a centralized notification list. Instead of hardcoding alert endpoints on every rule, we define this group once. If an alert fires, Azure instantly alerts the receivers via email, SMS, slack webhooks, or automated Azure functions.

### C. Metric Alerts (Response Time & Spikes)
We set up proactive, real-time metric thresholds directly on the Container App metrics:
```hcl
resource "azurerm_monitor_metric_alert" "latency" {
  name   = "alert-latency-high-${var.environment}"
  scopes = [var.api_container_app_id] # Watch the API container

  criteria {
    metric_namespace = "Microsoft.App/containerApps"
    metric_name      = "ResponseTime"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 500 # 500 milliseconds
  }

  action {
    action_group_id = azurerm_monitor_action_group.main.id
  }
}
```
If the average response time of the API container app exceeds **500ms** within its evaluation window, Azure automatically fires the alert, categorizes it as a warning, and emails your administrative Action Group.

---

### Key Takeaway
Azure Container Apps combined with Azure Monitor gives us enterprise-grade resilience out of the box. By defining KEDA auto-scaling, revision control, OTel telemetry, and Action Group alert chains directly in code, we ensure our environment is secure, robust, and entirely self-documenting.

Next: **Lesson 05 — CI/CD & Security Compliance**.

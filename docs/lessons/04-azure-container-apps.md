# Lesson 04: Azure Container Apps 🚀🌀

Azure Container Apps (ACA) is our "Serverless" platform. It gives us the power of Kubernetes without the headache of managing it.

## 1. Scale-to-Zero: The Money Saver 💰💤

In `modules/containerapp/main.tf`, we set:
```hcl
template {
  min_replicas = 0
  max_replicas = 3
}
```

**How it works:**
ACA uses **KEDA** (Kubernetes Event-driven Autoscaling). It watches for HTTP requests. If 5 minutes pass with 0 requests, it kills the container. When a request finally comes in, Azure "Wakes up" the container. This is why our project can run for very low cost!

## 2. Revisions: The Safety Net 🛡️🔄

We use `revision_mode = "Multiple"`. 

Every time you change a single variable or a line of code and run Terraform, Azure creates a **New Revision**. 
- **The Blue-Green Split**: You can have Revision A (Old) at 90% and Revision B (New) at 10% to test things safely.
- **Automatic Rollback**: If Revision B crashes on startup (like our "Poison Pill" experiment), Azure detects the failure and never switches traffic to it. Your users stay safe on Revision A.

## 3. Jobs vs. Apps 🏃‍♂️ vs 🧍‍♂️

We use two different types of compute:
- **`azurerm_container_app` (The API/Web)**: These stay "standing" (or sleeping) waiting for requests. They are meant for long-running interaction.
- **`azurerm_container_app_job` (The Worker)**: This is for "One-off" tasks. It uses a `cron_expression` (e.g., `*/1 * * * *`) to wake up, execute the Go worker code, and then completely disappear. This is much cheaper than keeping a worker running 24/7.

## 4. The Ingress (The Front Door) 🚪

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

This block handles the HTTPS certificates and the routing. We've told it to always send 100% of traffic to the latest version by default, but we added a **Lifecycle Ignore** in Terraform so we can manually split traffic in the Portal for testing!

---

### Key Takeaway
Azure Container Apps is "Opinionated Serverless." It forces us to use best practices like Revisions and Health Probes, which makes our application much more reliable than a standard Virtual Machine.

Next: **Lesson 05 — CI/CD & Security Compliance**.

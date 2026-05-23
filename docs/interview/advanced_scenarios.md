# Advanced DevOps Scenario & Grilling Guide

This guide contains high-level, advanced DevOps scenarios and architectural troubleshooting questions. These are the kinds of questions senior engineers use to "grill" candidates to see if they understand the deeper implications of their design decisions.

---

## 💥 Scenario 1: The Zero-Replica Cold Start Problem
**Question:**
> *"You configured `min_replicas = 0` on your API to save costs. However, during low-traffic periods, the first user who hits the website after it has scaled to zero experiences a 5 to 10-second delay (a 'cold start') while Azure spins up the container and the Go runtime initializes. 
> 
> In a production environment, this is unacceptable for user experience. How would you solve this cold start issue without losing all of your cost savings? If this app suddenly received 10,000 requests per second, how would you configure autoscaling?"*

**Answer Key:**
*   **Preventing Cold Starts:**
    1.  **Keep Warm Strategy:** Instead of scaling to 0 in production, set `min_replicas = 1` for the production environment only, while keeping it at `0` for Dev. In Terraform, this is managed by passing `min_replicas = var.environment == "pro" ? 1 : 0`.
    2.  **Fast Startup Optimization:** Since the Go binary is compiled statically and runs in a distroless container, it already starts in milliseconds. The primary bottleneck is container provisioning.
*   **Scale-up Configuration (KEDA):**
    *   Azure Container Apps uses **KEDA** (Kubernetes Event-driven Autoscaling) under the hood.
    *   We should configure scaling rules based on **Concurrent HTTP requests** (e.g., scale up if concurrent requests per replica exceed 50) rather than just CPU/Memory. CPU/Memory are trailing indicators (a container can crash from memory exhaustion before it finishes scaling), whereas active request queues scale proactively.

---

## ⚡ Scenario 2: Rolling Releases & Database Migration Race Conditions
**Question:**
> *"You run your database migrations as a standalone container job (`caj-healthcheck-migrate`) right before updating the API container image. 
> 
> If your migration drops a database column that the *currently running* (old) API version still requires, how do you prevent production errors during the 1-2 minutes it takes to roll out the new API version? How do you design database schema changes for zero-downtime deployments?"*

**Answer Key:**
*   **The Principle of Backward Compatibility:** Database migrations must always be backward-compatible with the *currently running* version of the application.
*   **Two-Phase Migration Pattern (Expand and Contract):**
    *   You never perform a breaking database change in a single deployment.
    *   *To rename a column:*
        1.  **Deploy 1 (Expand):** Add the *new* column to the database. Modify the application code to write to both columns but read from the old column.
        2.  **Data Migration:** Run a background script to copy old data to the new column.
        3.  **Deploy 2:** Modify the application code to read and write only to the *new* column.
        4.  **Deploy 3 (Contract):** Run a migration to drop the old column.
    *   This ensures that at no point does a running container query a column that does not exist.

---

## 🔍 Scenario 3: Troubleshooting Key Vault Inaccessible (403 / Network Timeout)
**Question:**
> *"Your API container fails to start in Production. The logs show: `dial tcp 10.0.3.4:443: i/o timeout` or `403 Forbidden` when attempting to fetch secrets from the Azure Key Vault. 
> 
> Walk me through your step-by-step troubleshooting process to isolate if this is a VNet DNS routing issue, a Private Endpoint issue, a firewall ACL issue, or an Entra ID RBAC permissions issue."*

**Answer Key:**
*   **Step 1: Check Private DNS Resolution (DNS Issue vs. Routing Issue)**
    *   Run a diagnostics container or check the container logs to see what IP address the Key Vault DNS name (e.g., `kv-healthcheck.vault.azure.net`) resolves to.
    *   If it resolves to a public IP (e.g., `191.x.x.x`), the Private DNS Zone link to the VNet is broken, and traffic is escaping the VNet.
    *   If it resolves to `10.0.3.4` (our Private Endpoint IP) but times out, DNS is correct, but routing is blocked.
*   **Step 2: Verify Private Endpoint & Network Security Groups (NSG/Routing Issue)**
    *   Ensure the Private Endpoint is in a `Succeeded` provisioning state.
    *   Check NSG rules on the App Subnet and the Endpoints Subnet. Verify that egress traffic from `snet-apps` is allowed to `snet-endpoints` on port 443.
*   **Step 3: Inspect Key Vault Firewall ACLs (403 Network Issue)**
    *   If the error is `403 Forbidden` with a network-related message, check Key Vault firewall settings. Even with Private Endpoints, Key Vault must be configured to allow 'Trusted Microsoft Services' and verify that VNet routing is permitted.
*   **Step 4: Check Entra ID RBAC Assignments (403 Auth/IAM Issue)**
    *   If the connection succeeds but returns `403 Access Denied: Caller is not authorized to perform action`, the issue is IAM.
    *   Verify that the API's User-Assigned Managed Identity has the **Key Vault Secrets User** role assigned at the Key Vault scope, and that the identity is correctly mounted in the Container App configuration.

---

## 🔄 Scenario 4: Cron Job Execution Overlap
**Question:**
> *"The worker job pings target sites and is configured to run every minute via a Cron trigger. 
> 
> What happens if one of the target websites becomes extremely slow, causing the worker job to take 90 seconds to finish, but the cron trigger fires the next job at the 60-second mark? How do you prevent overlapping executions from corrupting metrics or creating database transaction locks?"*

**Answer Key:**
*   **Concurrency Policy:** Azure Container App Jobs support concurrency settings. We can set the concurrency policy to `Forbid` or `Replace`. 
    *   `Forbid` (Recommended here) prevents a new job replica from starting if the previous one is still running.
    *   `Replace` terminates the currently running job and starts a new one.
*   **Application-Level Locks (Idempotency):**
    *   We should design the database queries to use `UPSERT` (e.g., `INSERT ... ON CONFLICT DO UPDATE`) rather than raw `INSERT` if duplicate pings for the same minute arrive.
    *   Configure connection and network timeouts in the Go HTTP client (e.g. max timeout of 10s per target) so that a slow target can never cause the worker to exceed its 60-second runtime limit.

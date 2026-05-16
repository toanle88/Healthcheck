# Lesson 03: Infrastructure as Code (Terraform) 🏗️

Terraform is the "Source of Truth" for our environment. If it's not in Terraform, it doesn't exist.

## 1. The Structure: "The Clean Split"

We split our Terraform into three logical layers:

### A. The Bootstrap (`infra/bootstrap/`)
This is the "Foundational" layer. It creates the resources that *store* our infrastructure, like the **Storage Account** for our `.tfstate` and the **Azure Container Registry (ACR)**. You only run this once.

### B. The Modules (`infra/modules/`)
These are our reusable "LEGO bricks." 
- **`network/`**: The VNet and NSGs.
- **`postgres/`**: The Flexible Server.
- **`identity/`**: The Managed Identities.
- **`containerapp/`**: The logic for our containers.

### C. The Environments (`infra/envs/dev/`)
This is where we "instantiate" the bricks. It’s like a recipe that says: *"Take the networking brick, the identity brick, and the container brick, and connect them together to make the 'Dev' environment."*

## 2. Deep Dive: The Network Castle 🛡️

The `network` module (`modules/network/main.tf`) is our primary line of defense. It builds a private, multi-tier virtual space:

```hcl
resource "azurerm_virtual_network" "main" {
  name                = "vnet-healthcheck-${var.environment}"
  address_space       = ["10.0.0.0/16"] # 65,536 private IP addresses
}
```

### A. Subnet Delegation
We slice this VNet into two `/24` subnets (256 private IPs each) and delegate them to specific services:
*   **`snet-apps` (`10.0.1.0/24`)**: Delegated exclusively to `Microsoft.App/environments` for our containers.
*   **`snet-db` (`10.0.2.0/24`)**: Delegated to `Microsoft.DBforPostgreSQL/flexibleServers`. This isolates the database inside our network sandbox.

### B. Network Security Groups (NSGs)
We attach strict firewalls to each subnet to enforce the **Least Privilege** network access rule:
1.  **Database Firewall (`nsg-db`)**: Only allows inbound traffic on port `5432` from the apps subnet (`10.0.1.0/24`). Everything else is dropped (`DenyAllInbound` rule).
2.  **Apps Firewall (`nsg-apps`)**: Allows standard HTTPS (`443`) and HTTP (`80`) inbound from the `Internet`. It explicitly denies SSH (`22`) from all sources to pass Checkov security policies (`CKV_AZURE_10`).

---

## 3. The "Secret" to No Secrets 🚫🔑

Look at `modules/identity/main.tf`. Instead of managing certificates or passwords, we split our security identities into two:

1.  **Deployment Identity (`github_actions` / `id-github-actions-${var.environment}`)**:
    *   Assigned the **"Contributor"** role strictly over our Resource Group.
    *   GitHub Actions logs in *as* this identity via secure **OIDC Workload Identity Federation**.
2.  **Runtime Identity (`apps` / `id-healthcheck-apps-${var.environment}`)**:
    *   Has zero permission to create or delete cloud infrastructure.
    *   Assigned only `Key Vault Secrets User` and `PostgreSQL Administrator` roles on those specific instances.
    *   **Scope Isolation**: Even if a hacker compromised our container API, they cannot modify or delete anything inside your Azure subscription.

### A. The Client Config Helper (`azurerm_client_config`)
Inside `modules/postgres/main.tf` and `modules/keyvault/main.tf`, you'll see:
```hcl
data "azurerm_client_config" "current" {}
```
This is a built-in provider utility that dynamically fetches the active deployment session's metadata (e.g., `tenant_id`, `subscription_id`, and `object_id`). We use it to automatically link our resources to your active Entra ID tenant without hardcoding sensitive IDs.

---

## 4. Deep Dive: The Database (`modules/postgres/main.tf`) 🐘

This module implements a fully secure, scalable, and isolated PostgreSQL Flexible Server database tier:

### A. Private DNS Zone & Link
```hcl
resource "azurerm_private_dns_zone" "postgres" {
  name = "healthcheck.postgres.database.azure.com"
}

resource "azurerm_private_dns_zone_virtual_network_link" "main" {
  virtual_network_id  = var.vnet_id
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
}
```
Because our database has zero public internet access (`public_network_access_enabled = false`), it only has a private IP address within our VNet. This DNS Zone maps the public database hostname to its private internal IP, so our apps can securely resolve it.

### B. Flexible Server Engine
```hcl
resource "azurerm_postgresql_flexible_server" "main" {
  name                = "psql-healthcheck-${var.environment}"
  delegated_subnet_id = var.subnet_id
  private_dns_zone_id = azurerm_private_dns_zone.postgres.id
  sku_name            = "B_Standard_B1ms" # Burstable tier perfect for dev ($0.017/hour)

  authentication {
    active_directory_auth_enabled = true
    password_auth_enabled         = true
  }
}
```
We set up dual authentication modes (`authentication`), enabling both traditional password logins (used as a fallback) and modern, highly secure **Active Directory (Entra ID) token-based authentication**.

---

## 5. Deep Dive: Key Vault Secrets (`modules/keyvault/main.tf`) 🔑

Key Vault secures our system configuration, but it has its own strict security rules:

```hcl
resource "azurerm_key_vault" "main" {
  name                        = "kv-hc-${var.environment}-${random_string.kv_suffix.result}"
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  sku_name                    = "standard"
  rbac_authorization_enabled  = true # Uses modern RBAC roles instead of legacy access policies
}
```

### A. Role-Based Vault Access
Normally, even a subscription "Owner" cannot read or write secrets inside a Key Vault by default. To solve this, we explicitly assign our deployment identity the **Secrets Officer** role:
```hcl
resource "azurerm_role_assignment" "current_user_secrets" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Secrets Officer"
  principal_id         = data.azurerm_client_config.current.object_id
}
```

---

### Key Takeaway
Terraform allows us to build complex security (VNets, NSGs, Postgres Private Links, RBAC Key Vaults, and OIDC Identities) in a repeatable, documentable blueprint. We don't "hope" the security is right; we *code* it to be right.

Next: **Lesson 04 — Azure Container Apps (Scaling & Resilience)**.

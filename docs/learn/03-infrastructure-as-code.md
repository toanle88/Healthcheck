# Lesson 03: Infrastructure as Code (Terraform & Bicep) 🏗️

Terraform is the "Source of Truth" for our cloud environments. Rather than clicking buttons manually inside the Azure Portal (which is slow, unrepeatable, and prone to errors), we define our entire infrastructure as HCL (HashiCorp Configuration Language) code.

---

## 📁 1. The Directory Structure: "The Clean Split"

To build enterprise-ready infrastructure, we partition our Terraform into three layers:

### A. The Bootstrap (`infra/terraform/bootstrap/`)
This is the foundational layer. It creates the resources that *store* our infrastructure, like the **Storage Account** for our Terraform state file (`.tfstate`) and the **Azure Container Registry (ACR)**.
*   **Why it's split:** You only run this once. We build the registry first so we have a place to upload our Docker images *before* we deploy our container apps.

### B. Reusable Modules (`infra/terraform/modules/`)
These are our reusable "LEGO bricks." They do not declare environments directly; instead, they define parameterized components, split by compliance and hardening requirements:
*   **Common Baseline Modules (`infra/terraform/modules/common/`)**: Shared modules across all environments.
    *   `acr/`: Azure Container Registry (ACR) configuration.
    *   `auth/`: Entra ID/CIAM authorization setup.
    *   `containerapp/`: Configures container app environments, API/Web apps, and worker jobs.
    *   `identity/`: Manages federated OIDC credentials and user-assigned managed identities.
    *   `monitor/`: Sets up Log Analytics, Application Insights, and monitoring alerts.
    *   `policy/`: Enforces required Azure tags.
    *   `network/`, `postgres/`, `keyvault/`: Default dev configurations (cost-optimized, public/HTTP accessible).
*   **Production Hardened Overrides (`infra/terraform/modules/pro/`)**:
    *   `network/`: Builds the VNet with an additional private endpoint subnet (`snet-endpoints`), NSG blocking public HTTP (port 80) access (`CKV_AZURE_160`), and NSG isolating endpoints to container app subnet traffic only (`CKV2_AZURE_31`).
    *   `postgres/`: Provisions PostgreSQL Flexible Server with geo-redundant backups enabled (`CKV_AZURE_136`).
    *   `keyvault/`: Provisions a Key Vault with purge protection enabled (`CKV_AZURE_115`), public network access disabled (`CKV_AZURE_189`), default network rules set to Deny (`CKV_AZURE_109`), and a **Private Endpoint** (`pe-kv-pro`) in the endpoints subnet (`CKV2_AZURE_32`).

### C. The Environments (`infra/terraform/environments/`)
This is where we combine our modules to build a specific environment. It acts as a configuration file, passing environment-specific inputs (like different subnets or backups) to our modules:
*   `dev/`: Dev configuration utilizing `common/` modules for all components (optimized for cost and simplicity).
*   `pro/`: Production configuration utilizing `common/` modules for identity, apps, and monitoring, but overriding them with the hardened `pro/` modules for network, postgres, and keyvault.

---

## 🛡️ 2. Deep Dive: Networking & Subnets

The network module (baseline `infra/terraform/modules/common/network/main.tf` and production `infra/terraform/modules/pro/network/main.tf`) is our primary defense. It constructs our private virtual space:

```hcl
resource "azurerm_virtual_network" "main" {
  name                = "vnet-healthcheck-${var.environment}"
  address_space       = ["10.0.0.0/16"]
}
```

### Understanding CIDR Notation (For Beginners)
*   **`10.0.0.0/16`**: The `/16` means the first 16 bits of the IP address (corresponding to `10.0.`) are locked. This leaves 16 bits for hosts, giving us a range from `10.0.0.0` to `10.0.255.255` (65,536 private IP addresses).
*   **`/24` Subnets**: We slice this large block into smaller `/24` subnets (first 24 bits locked, like `10.0.1.`). This leaves 8 bits for hosts, giving us 256 IPs per subnet.

### Subnet Delegation
Standard subnets are just generic pools of IP addresses. However, certain Azure services need deep integration with the network card layer.
We tell Azure that specific subnets are reserved exclusively for specific services:

```hcl
delegation {
  name = "aca-delegation"
  service_delegation {
    name    = "Microsoft.App/environments"
    actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
  }
}
```
*   **`Microsoft.App/environments`**: Delegated to Azure Container Apps (`snet-apps` / `10.0.1.0/24`).
*   **`Microsoft.DBforPostgreSQL/flexibleServers`**: Delegated to PostgreSQL (`snet-db` / `10.0.2.0/24`).

### Network Security Groups (NSGs)
NSGs act as network firewalls at the subnet level. We use them to enforce the principle of **Least Privilege**:
1.  **Database Subnet Security (`nsg-db`)**: We add a rule that only allows inbound traffic on port `5432` (Postgres) if it originates from our apps subnet IP range (`10.0.1.0/24`). All other traffic is blocked.
2.  **Apps Subnet Security (`nsg-apps`)**:
    *   **In Dev**: Allows HTTP (80) and HTTPS (443) from the Internet, and blocks incoming SSH traffic on port `22` to satisfy Checkov policy audits (`CKV_AZURE_10`).
    *   **In Production**: **Denies HTTP (80)** to comply with Checkov policy audits (`CKV_AZURE_160`), allowing only HTTPS (443) from the Internet, and blocks incoming SSH on port `22`.
3.  **Endpoints Subnet Security (`nsg-endpoints` - Production Only)**:
    *   Restricts inbound traffic on port `443`, allowing it **only** if it originates from our apps subnet (`10.0.1.0/24`) to reach the endpoints subnet (`10.0.3.0/24`), denying all other inbound to satisfy Checkov policy audits (`CKV2_AZURE_31`).

---

## 🚫 3. OIDC and Identity Isolation

Instead of using credentials, we split our system into two distinct active identities:

1.  **Deployment Identity (`id-github-actions`)**:
    *   Assigned the `Contributor` role over our specific Resource Group.
    *   GitHub Actions logs in *as* this identity via secure **OIDC (OpenID Connect) Workload Identity Federation**.
    *   No credentials or client secrets are saved in GitHub secrets!
2.  **Runtime Identity (`id-healthcheck-apps`)**:
    *   This is the identity our Go containers run as.
    *   It has zero rights to create, modify, or delete Azure resources.
    *   It only has `Key Vault Secrets User` (to read configs) and `PostgreSQL Administrator` roles.

---

## 🐘 4. VNet Database Isolation (baseline `modules/common/postgres/main.tf` / pro `modules/pro/postgres/main.tf`)

Because our PostgreSQL server has no public IP address, we must address two problems: how does the app find it, and how does the database authenticate users?

### Private DNS Resolution
Because our database is entirely private, standard public DNS servers cannot resolve its IP address. We create a **Private DNS Zone** inside our VNet:

```hcl
resource "azurerm_private_dns_zone" "postgres" {
  name = "healthcheck.postgres.database.azure.com"
}

resource "azurerm_private_dns_zone_virtual_network_link" "main" {
  virtual_network_id    = var.vnet_id
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
}
```
This maps the database hostname (e.g., `psql-healthcheck-dev.postgres.database.azure.com`) directly to its private network IP address (`10.0.2.x`), allowing our Go applications to connect.

### Enabling Entra ID Authentication
We configure PostgreSQL to enforce passwordless Entra ID authentication and disable password authentication entirely:

```hcl
authentication {
  active_directory_auth_enabled = true
  password_auth_enabled         = false
}
```

### Geo-Redundant Backups (Production Only)
To satisfy production disaster recovery audits (`CKV_AZURE_136`), the production PostgreSQL database module enables geo-redundant backups:
```hcl
geo_redundant_backup_enabled = true
```

---

## 🔑 5. Secure Configurations (baseline `modules/common/keyvault/main.tf` / pro `modules/pro/keyvault/main.tf`)

Key Vault is where we store application configurations (like API keys or endpoints). We configure it to use modern **Role-Based Access Control (RBAC)** instead of legacy access policies:

```hcl
resource "azurerm_key_vault" "main" {
  name                        = "kv-hc-${var.environment}-${random_string.kv_suffix.result}"
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  sku_name                    = "standard"
  rbac_authorization_enabled  = true
}
```

### Access Isolation
By setting `rbac_authorization_enabled = true`, access to the vault is managed using standard Azure role assignments. Even the user who created the subscription cannot read secrets inside the Key Vault unless they are explicitly assigned the **Key Vault Secrets Officer** or **Key Vault Secrets User** role!

### Production-Hardened Security (Production Only)
To comply with enterprise security and Checkov compliance, the production Key Vault overrides the default configuration as follows:
1. **Purge Protection**: Enabled (`purge_protection_enabled = true`) to prevent accidental permanent deletion (`CKV_AZURE_115`).
2. **Public Network Access**: Disabled (`public_network_access_enabled = false`) to enforce private routing (`CKV_AZURE_189`).
3. **Default Network Action**: Set to Deny (`default_action = "Deny"`) to block all unauthorized network access (`CKV_AZURE_109`).
4. **Private Endpoint**: Provisions a private endpoint (`azurerm_private_endpoint.kv`) on the dedicated `snet-endpoints` subnet so that container apps in `snet-apps` can securely access the vault via private Azure routing (`CKV2_AZURE_32`).

---

### Next Steps 🚀
Now that we've coded our environment blueprint, let's explore **[Lesson 04: Azure Container Apps](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/04-azure-container-apps.md)** to see how we deploy, scale, and monitor our Go containers.

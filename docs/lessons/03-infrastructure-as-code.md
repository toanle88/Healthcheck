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

The `network` module is the most complex. It implements **VNet Injection**:

```hcl
# This tells Postgres: "Don't go to the internet. Stay in this subnet."
resource "azurerm_postgresql_flexible_server" "main" {
  delegated_subnet_id = var.subnet_id
  private_dns_zone_id = var.dns_zone_id
}
```

By delegating a subnet to Postgres, we create a private bridge between our apps and our data. No one from the outside can cross that bridge.

## 3. The "Secret" to No Secrets 🚫🔑

Look at `modules/identity/main.tf`. We create a **User-Assigned Managed Identity**.

In `envs/dev/main.tf`, we perform a **Role Assignment**:
```hcl
resource "azurerm_role_assignment" "pg_admin" {
  scope                = module.postgres.id
  role_definition_name = "PostgreSQL Flexible Server Active Directory Administrator"
  principal_id         = module.identity.principal_id
}
```

This is the exact moment we "link" the Go code to the Database. Instead of giving the app a password, we give the app a **Role**. Azure then checks that role every time the app tries to connect.

---

### Key Takeaway
Terraform allows us to build complex security (VNets, Roles, Managed Identities) in a way that is repeatable and documentable. We don't "hope" the security is right; we *code* it to be right.

Next: **Lesson 04 — Azure Container Apps (Scaling & Resilience)**.

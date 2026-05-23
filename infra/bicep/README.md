# 📖 Declarative IaC Standards: Bicep vs. Terraform

This directory provides Azure Bicep templates designed as a direct equivalent to the primary Terraform configurations under `infra/terraform/`. This dual-template setup acts as an educational comparison tool for comparing declarative IaC standards frequently tested in Azure certifications:
- **AZ-104 (Azure Administrator)**: Basic infrastructure deployment, resource scoping, virtual networking, identity mapping, and policies.
- **AZ-400 (DevOps Engineer Expert)**: IaC state security, pipeline execution models, OIDC authentication, drift detection, and modular release gates.

---

## 🗺️ Architectural Comparison Matrix

| Objective / Feature | 🚀 Azure Bicep (Domain-Specific Language) | 🌍 HashiCorp Terraform (HCL) |
| :--- | :--- | :--- |
| **State Management** | **Stateless**: Directly queries the Azure Resource Manager (ARM) API. No state files to secure, lock, or corrupt. | **Stateful**: Relies on a `terraform.tfstate` database (stored in Azure Blob Storage here) to track resource metadata. |
| **Azure Zero-Day Support**| **Immediate**: Native to Azure; supports new resource types or preview API versions immediately. | **Delayed**: Relies on the HashiCorp `azurerm` provider team updating resources and attributes. |
| **Multi-Cloud Capabilities**| **Azure Only**: Designed specifically for the Azure Resource Manager control plane. | **Multi-Cloud**: Extensible provider model supports AWS, GCP, Kubernetes, SaaS products, etc. |
| **Target Scope Handling** | Managed natively via the `scope` parameter in module calls (deploy across RG/Sub/Tenant easily). | Managed using explicit `provider` aliases or varying configurations passed to modules. |
| **Compilation Model** | Transpiles into standard Azure Resource Manager (ARM) JSON templates before deployment. | Directly compiles code into memory and executes API requests against the Azure endpoint. |
| **Drift Detection** | Handled natively by Azure Policy and ARM templates during deployment using the `what-if` analysis. | Handled locally or in CI/CD by comparing current live state against the recorded state database. |

---

## 🏛️ Deep-Dive Exam Concepts (AZ-104 & AZ-400)

### 1. State Management & Lifecycle

*   **Terraform (Stateful)**:
    *   **The State File**: Maps declarative resources to real-world infrastructure. In this project, `tfstate` is saved in a private Azure Storage Account container with lease locking to prevent concurrent deployment conflicts.
    *   **Implications**: If the state database gets out of sync (e.g., resources deleted out-of-band), Terraform might fail to deploy or attempt to destroy/recreate active components. Security of the state file is critical since it contains secrets in plaintext.
*   **Bicep (Stateless)**:
    *   **Direct API Discovery**: Bicep queries the Azure Resource Manager database on-demand. When you run `az deployment`, Azure compares your template against the live control plane.
    *   **Implications**: No state file to secure or configure backend locking for. However, since there is no cached mapping, detecting drift or deletions is harder without using Azure Policy or dry-run tools.

### 2. Syntax & Modularity

Compare the structure of a Virtual Network module between the two formats:

```carousel
#### Terraform (infra/terraform/modules/common/network/main.tf)
```hcl
resource "azurerm_virtual_network" "main" {
  name                = "vnet-healthcheck-${var.environment}"
  location            = var.location
  resource_group_name = var.resource_group_name
  address_space       = ["10.0.0.0/16"]
}

resource "azurerm_subnet" "container_apps" {
  name                 = "snet-apps"
  resource_group_name  = var.resource_group_name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.1.0/24"]
  delegation {
    name = "aca-delegation"
    service_delegation {
      name    = "Microsoft.App/environments"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}
```
<!-- slide -->
#### Bicep (infra/bicep/modules/network.bicep)
```bicep
resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'vnet-healthcheck-${environment}'
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [ '10.0.0.0/16' ]
    }
    subnets: [
      {
        name: 'snet-apps'
        properties: {
          addressPrefix: '10.0.1.0/24'
          delegations: [
            {
              name: 'aca-delegation'
              properties: {
                serviceName: 'Microsoft.App/environments'
              }
            }
          ]
        }
      }
    ]
  }
}
```
```

### 3. Deployment Scopes (Subscription vs. Resource Group)

In Bicep, you can declare target scopes natively. Our `main.bicep` runs at the `subscription` scope to create the environment resource group (`rg-healthcheck-dev`), then deploys resource-scoped modules inside it.

In Terraform, this is managed by structuring modules or using providers. For example:
- **Bicep Target Scope**:
  ```bicep
  targetScope = 'subscription'
  resource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {
    name: 'rg-healthcheck-${environment}'
    location: location
  }
  module network './modules/network.bicep' = {
    name: 'network-deployment'
    scope: rg // Explicitly setting deployment scope!
    params: { ... }
  }
  ```
- **Terraform Target Scope**:
  ```hcl
  # Implicitly scoped via provider authentication
  resource "azurerm_resource_group" "dev" {
    name     = "rg-healthcheck-dev"
    location = var.location
  }
  module "network" {
    source              = "../../modules/common/network"
    resource_group_name = azurerm_resource_group.dev.name
  }
  ```

---

## 🛠️ How to Validate & Deploy

### A. Pre-Deployment Validation (Dry-Runs)

To test Bicep configurations without modifying Azure resources, compile the template to see raw ARM JSON or run a What-If dry-run analysis.

#### 1. Compile to ARM JSON
```bash
az bicep build --file infra/bicep/main.bicep
```
*Creates `infra/bicep/main.json` which is the underlying ARM template.*

#### 2. Run What-If Analysis (Drift/Change Preview)
```bash
az deployment sub what-if \
  --location eastasia \
  --template-file infra/bicep/environments/dev/main.bicep \
  --parameters \
    acrName=crhealthcheckbootstrap \
    entraClientId=00000000-0000-0000-0000-000000000000
```

---

### B. Deployment Commands

#### 1. Deploy Bootstrap Layer
```bash
az deployment sub create \
  --location eastasia \
  --template-file infra/bicep/bootstrap/main.bicep \
  --parameters \
    githubOrg=your-github-org \
    githubRepo=your-repo-name
```

#### 2. Deploy Full Environment (Dev)
```bash
az deployment sub create \
  --location eastasia \
  --template-file infra/bicep/environments/dev/main.bicep \
  --parameters \
    acrName=crhealthcheckxxxx \
    entraClientId=your-entra-ciam-client-id \
    deployerPrincipalId=$(az ad signed-in-user show --query id -o tsv)
```

---

## 🏁 Exam Cheat Sheet Summary

1. **State Locking**: Terraform supports it natively for backends like Consul or Azure Blob Storage (uses Blob Lease). Bicep has no state locking, as concurrent deployments are serialized by the ARM control plane.
2. **Importing Existing Infrastructure**:
   - **Terraform**: Requires `terraform import` commands or writing `import {}` blocks in code to match state with the cloud.
   - **Bicep**: Declared using the `existing` keyword (e.g., `resource acr 'Microsoft.ContainerRegistry/registries@...'' existing = { name: 'myacr' }`).
3. **Secrets Handling**:
   - **Terraform**: Secrets in state files are saved in **plaintext**. Always secure state storage.
   - **Bicep**: No state file exists, but outputting sensitive variables as outputs is an anti-pattern. Use the `@secure()` decorator for parameters, and read secrets dynamically from Key Vault references rather than passing them through parameters.

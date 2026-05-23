# 🚀 Healthcheck Dashboard: Professional Deployment Guide

This document outlines the **"Clean Split"** architectural model for the Healthcheck Dashboard. In this model, core infrastructure is fully automated via Terraform, while Customer Identity (CIAM) is managed as a one-time curated setup for maximum stability.

## 🏗️ The Hybrid Flow
| Feature | Management | Frequency |
| :--- | :--- | :--- |
| **Azure Infrastructure** (DB, VNet, Apps) | **Fully Automated** (Terraform) | Every deployment |
| **CIAM App Registration** | **One-time Manual** (Azure Portal) | Once per environment |
| **CI/CD Pipeline** | **Automated** (GitHub Actions) | On every push |

## 🧱 Step 0: The Bootstrap (Foundation)
Before the first deployment, you must build the "Foundation" where your State and Images live.

1. **Run Bootstrap Terraform**: Navigate to `infra/terraform/bootstrap` and run `terraform apply`. This creates:
   - The **Resource Group** for your base infrastructure.
   - The **Storage Account** for your `.tfstate` files.
   - The **Container Registry (ACR)** for your Docker images.
   - The **Managed Identity** with OIDC trust for GitHub Actions.
2. **Configure GitHub**: Add the outputted `client_id`, `tenant_id`, and `subscription_id` to your GitHub Repository Secrets.

---


## 🛡️ Step 1: Manual CIAM Configuration (One-Time)
Since CIAM exists in a separate directory, we configure it manually in the Azure Portal. Follow these exact steps:

### 1. Create the App Registration
1. Log into the [Azure Portal](https://portal.azure.com) and switch to your **CIAM Directory**.
2. Navigate to **Microsoft Entra ID** > **App registrations** > **New registration**.
3. **Name**: `Healthcheck-Dashboard-dev` (or `Healthcheck-Dashboard-pro` for production)
4. **Supported account types**: "Accounts in this organizational directory only".
5. **Redirect URI**: Select **SPA (Single-page application)** and enter `http://localhost:5173/`.
6. Click **Register**.

### 2. Configure Authentication
1. Inside the new app, go to **Authentication**.
2. Click **Add a platform** > **Web**.
3. Enter `https://localhost:3000/` as the Redirect URI.
4. Click **Configure**.

### 3. Expose the API (The "Anchor")
1. Go to **Expose an API**.
2. Next to **Application ID URI**, click **Add**. 
3. Leave the default GUID-based URI (`api://<client-id>`) and click **Save**. Do not enter a custom name to ensure MSAL scope matching works out-of-the-box.
4. Click **Add a scope**:
   - **Scope name**: `access_as_user`
   - **Who can consent?**: Admins and users.
   - **Admin consent display name**: `Access Healthcheck API`
   - **Admin consent description**: `Allows the application to access the Healthcheck API as the user.`
5. Click **Add scope**.

### 4. Set Token Version
1. Go to **Manifest** (left sidebar).
2. Find `"accessTokenAcceptedVersion": null,` and change it to **`2`**.
3. Click **Save**.

### 5. Capture the IDs
- **Application (client) ID**: Found on the **Overview** page.
- **Directory (tenant) ID**: Found on the **Overview** page.
- *Save these for Step 2.*

---

## 🤖 Step 2: Automated Infrastructure (GitHub Actions)
Once the CIAM registration is built, the rest of the world follows.

### Required GitHub Secrets
Ensure these are set in your GitHub repository:
- `AZURE_CLIENT_ID`: Your **Main Tenant** Bootstrap SP ID.
- `AZURE_TENANT_ID`: Your **Main Tenant** ID.
- `AZURE_SUBSCRIPTION_ID`: Your Azure Subscription ID.
- `AZURE_ACR_NAME`: Your Container Registry name.
- `ENTRA_CLIENT_ID`: The GUID from Step 1 above.

### The Deployment Logic
The environment is now "Identity-Driven."
1.  **GitHub** uses **OIDC** to assume the identity of the Bootstrap Managed Identity.
2.  **Terraform** creates the environment, including a **User-Assigned Managed Identity** for the app.
3.  **The App** uses its identity to request **AAD Tokens** for PostgreSQL and Key Vault.

**Zero secrets are stored in GitHub, Terraform state, or environment variables.**

### Production Environments & Manual Approvals

To ensure safe deployment to production (`pro`), both CI/CD pipelines target the `pro` environment which acts as an approval gate:
- **GitHub Actions**: The `deploy-pro` job references the `pro` environment. To configure this:
  1. Go to your GitHub Repository > **Settings** > **Environments**.
  2. Click **New environment** and name it `pro`.
  3. Check **Required reviewers** under **Environment protection rules** and select the designated approvers.
- **Azure DevOps**: The `DeployPro` stage utilizes a deployment job targeting the `pro` environment. To configure this:
  1. Navigate to your Azure DevOps Project > **Pipelines** > **Environments**.
  2. Click **New environment** and name it `pro` (select **None** for resource type).
  3. Click the three dots next to the created environment and choose **Approvals and checks**.
  4. Click **+** to add **Approvals**, then specify the authorized users/groups.

---

---

## 🛢️ Step 3: Database Migrations
Rather than running database schema setup automatically when the API starts up (which can cause locks and race conditions under multi-replica environments), migrations are managed as an independent step.

### 1. Local Development (Docker Compose)
When you run `docker-compose up --build`:
* The `migrate` container builds using [Dockerfile.migrate](file:///mnt/d/Dev/Projects/Healthcheck/Dockerfile.migrate), connecting to PostgreSQL using local credentials.
* The `api` and `worker` containers use the `service_completed_successfully` condition to block startup until the `migrate` container completes all `up` migration scripts.

### 2. Cloud Environments (GitHub Actions & Azure DevOps)
In the dev/production pipelines:
* **Build Phase**: The pipeline compiles and builds the `migrate` image alongside `api`, `web`, and `worker`, pushing it to ACR.
* **Deploy Phase**:
  1. The pipeline updates/provisions a dedicated **Azure Container App Job** named `caj-healthcheck-migrate-[dev/pro]`.
  2. The pipeline triggers the job to run. Since it runs as a container within our VNet subnet and inherits the same User-Assigned Managed Identity, it authenticates passwordlessly to PostgreSQL using Azure Entra ID.
  3. The pipeline polls the job status every 10 seconds.
  4. If the migration job fails (exits non-zero), the deployment halts immediately. This prevents a bad schema modification or database connection failure from affecting the live API/Web Container Apps.

---

## ☢️ The "Fresh Start" Procedure
If you have manually deleted your Resource Group and State, follow these steps to rebuild:

1. **Re-Initialize Locally (One-time)**:
   Navigate to the desired environment directory:
   ```powershell
   cd infra/terraform/environments/dev # or cd infra/terraform/environments/pro
   terraform init -reconfigure -backend-config="storage_account_name=<STORAGE_NAME>"
   ```
2. **Push to Main**: Simply push your code to GitHub. The updated `infra.yml` will detect the empty state and rebuild the entire environment.

---

## 🔍 State Drift Management

To ensure that live Azure resources do not deviate from their defined Infrastructure as Code (IaC) configuration (e.g. from manual changes in the Azure Portal), we have automated state drift detection.

### 1. Automation
Drift checks are run automatically on a scheduled basis (every night at midnight UTC) and can also be triggered manually:
* **GitHub Actions**: The [.github/workflows/drift.yml](file:///mnt/d/Dev/Projects/Healthcheck/.github/workflows/drift.yml) workflow runs speculative plans against the `dev` and `pro` environments.
* **Azure DevOps**: The [.azure-pipelines/drift.yml](file:///mnt/d/Dev/Projects/Healthcheck/.azure-pipelines/drift.yml) pipeline executes the same drift detection checks.

Both pipelines run Terraform plan with the `-detailed-exitcode` flag.
* If there is no drift, the step succeeds (exit code `0`).
* If drift is detected (exit code `2`), the step logs the drift diffs, raises a build/run failure, and alerts the engineering team.

### 2. Manual Verification
You can manually trigger the drift check at any time by running the [.github/workflows/drift.yml](file:///mnt/d/Dev/Projects/Healthcheck/.github/workflows/drift.yml) workflow via **GitHub Actions** (Workflow Dispatch) or by running [.azure-pipelines/drift.yml](file:///mnt/d/Dev/Projects/Healthcheck/.azure-pipelines/drift.yml) from the **Azure DevOps Pipelines** dashboard.

### 3. Remediation
When a drift alert occurs, remediate it using one of the following methods depending on whether the manual changes are desired:

* **Option A: Revert the Drift (Overwrite Manual Changes)**
  If the manual change was a mistake or temporary, you can overwrite it and restore the defined IaC state:
  * In GitHub, manually trigger [.github/workflows/infra.yml](file:///mnt/d/Dev/Projects/Healthcheck/.github/workflows/infra.yml) with `confirm_apply="yes"`.
  * In Azure DevOps, manually trigger [.azure-pipelines/infra.yml](file:///mnt/d/Dev/Projects/Healthcheck/.azure-pipelines/infra.yml) with `confirm_apply` parameter set to `yes`.

* **Option B: Adopt the Drift (Update Terraform Code)**
  If the manual change is correct and should be kept permanently:
  1. Update the local Terraform code under [infra/terraform](file:///mnt/d/Dev/Projects/Healthcheck/infra/terraform) to match the manual changes.
  2. Create a Pull Request (PR). The PR plan check will verify if there are any remaining differences.
  3. Once merged to `main`, the next automated drift detection run will pass cleanly.

---

## ✅ Final Review Verification
- **Frontend**: Uses Entra External ID (CIAM) for secure user login.
- **Backend**: Uses **Managed Identity** to log into Postgres (No DB password exists in the config).
- **Network**: **VNet Injection** ensures the database is invisible to the public internet.
- **Security**: **Checkov** scans every PR to ensure infrastructure compliance.

**Your architecture is now a production-ready, zero-secret masterpiece. 🟢🚀🛡️**

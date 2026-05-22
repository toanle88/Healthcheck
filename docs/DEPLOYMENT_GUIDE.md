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

1. **Run Bootstrap Terraform**: Navigate to `infra/bootstrap` and run `terraform apply`. This creates:
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

## ☢️ The "Fresh Start" Procedure
If you have manually deleted your Resource Group and State, follow these steps to rebuild:

1. **Re-Initialize Locally (One-time)**:
   Navigate to the desired environment directory:
   ```powershell
   cd infra/envs/dev # or cd infra/envs/pro
   terraform init -reconfigure -backend-config="storage_account_name=<STORAGE_NAME>"
   ```
2. **Push to Main**: Simply push your code to GitHub. The updated `infra.yml` will detect the empty state and rebuild the entire environment.

---

## ✅ Final Review Verification
- **Frontend**: Uses Entra External ID (CIAM) for secure user login.
- **Backend**: Uses **Managed Identity** to log into Postgres (No DB password exists in the config).
- **Network**: **VNet Injection** ensures the database is invisible to the public internet.
- **Security**: **Checkov** scans every PR to ensure infrastructure compliance.

**Your architecture is now a production-ready, zero-secret masterpiece. 🟢🚀🛡️**

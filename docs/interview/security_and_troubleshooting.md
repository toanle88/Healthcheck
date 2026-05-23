# SRE Security & Advanced Troubleshooting Scenarios

This guide focuses specifically on advanced **Security Operations (SecOps)**, **Pipeline Supply Chain Security**, and **Real-World Live Troubleshooting** scenarios. SRE and Security teams use these questions to verify if you can handle high-pressure production incidents.

---

## 🔒 Category 1: Advanced Security & Incident Response

### Scenario 1: Leakage of Secrets in Production Logs
**Question:**
> *"During a debugging session, a developer temporarily adds verbose logging that prints the database connection string—including the password—to standard output. This stdout is captured by Azure Container Apps and shipped to Azure Log Analytics. The debug code is now deleted, but the password remains visible in historical logs to anyone with read access. 
> 
> What are your immediate incident response steps to remediate this leak, rotate the credentials safely without downtime, and scrub the logs?"*

**Answer Key:**
1.  **Isolate & Triage:** Revoke read access to the compromised Log Analytics Workspace temporarily if possible, or limit access to the security team while remediation is active.
2.  **Rotate Credentials (Zero-Downtime):**
    *   Since we are using **Entra ID Managed Identities** for the main database connection, we are safe from database credential leaks! However, if this was a legacy service using a password:
    *   **Double-Credentials Pattern:** Postgres allows you to keep multiple active administrator/user credentials.
        1. Create a *new* password for the DB user.
        2. Update the secret value in Azure Key Vault with the new password.
        3. Trigger a rolling restart of the Container Apps so they pull the new secret from Key Vault.
        4. Once all active containers are confirmed running on the new password, delete the *old* password from the Postgres database.
3.  **Purge Logs:** Log Analytics Workspaces support data purging via the Azure REST API (`Purge` command). We must submit a purge request targeting the specific time window and table (e.g. `ContainerAppConsoleLogs_CL`) filtering for the leaked password string.

### Scenario 2: Supply Chain Attack & Malicious Package Injection
**Question:**
> *"A developer imports a popular Go library or npm package for a new frontend component. Unknown to them, the maintainer's account was compromised, and the new version contains a malicious post-install script that tries to exfiltrate environment variables. 
> 
> How does your current CI/CD pipeline protect the production environment from this 'Supply Chain' attack?"*

**Answer Key:**
*   **Static Vulnerability Scanning (Trivy):** In our CI pipeline ([cicd.yml](file:///mnt/d/Dev/Projects/Healthcheck/.github/workflows/cicd.yml)), we run **Trivy** on the container filesystem. Trivy scans the Go and npm lockfiles (`go.sum`, `package-lock.json`) against global CVE databases to detect known compromised packages before the container is deployed.
*   **Dependency Lockfiles:** Enforcing lockfiles prevents the pipeline from dynamically pulling "latest" versions that might include unpinned, compromised code.
*   **Network Ingress/Egress Isolation:** Even if a malicious package runs, it cannot exfiltrate data if the container has no route to the public internet. Our background worker runs inside a private subnet with restricted NAT Gateway egress rules, preventing outbound HTTP calls to unapproved destination IPs.
*   **Least Privilege Run-Time:** Our containers run as `nonroot` users, meaning the malicious script cannot install system-level packages, inspect other root processes, or modify the container filesystem.

---

## 🛠️ Category 2: Live Troubleshooting & Diagnostics

### Scenario 3: Container Denied Access to Key Vault (403 Forbidden)
**Question:**
> *"Your Go API container starts up but immediately crashes. The error logs show: `Failed to fetch secret: Key Vault access policy denied (HTTP 403)`. 
> 
> We are using Azure RBAC (not Key Vault Access Policies) and User-Assigned Managed Identity. How do you troubleshoot and fix this error step-by-step?"*

**Answer Key:**
1.  **Check Key Vault Auth model:** Verify that the Key Vault is configured to use **Azure role-based access control (RBAC)** instead of "Vault access policies" in the Azure portal. (Managed in Terraform via `enable_rbac_authorization = true`).
2.  **Verify IAM Role Assignment:** Confirm that the User-Assigned Managed Identity is assigned the **Key Vault Secrets User** role at the Key Vault scope (or resource group scope).
3.  **Check Container App Identity Binding:** Verify that the Container App's YAML configuration binds the User-Assigned Identity to the container, and that `AZURE_CLIENT_ID` is set in the container's environment variables to point to the client ID of the User-Assigned Identity (so the Azure Go SDK knows *which* identity to use when requesting a token).
4.  **Wait for Propagation:** Azure AD role assignments can take 1-2 minutes to propagate. If this is a fresh Terraform deployment, verify that a `time_sleep` resource was included to wait for RBAC propagation before starting the Container App.

### Scenario 4: DNS Resolution Failure inside a Container
**Question:**
> *"Your Go Worker container is running, but it fails to ping one of the target URLs. The log shows: `dial tcp: lookup api.github.com: no such host`. The same URL works fine from your laptop. 
> 
> How do you debug DNS resolution issues inside a running Docker container or Azure Container App?"*

**Answer Key:**
1.  **Isolate container vs host:** Run a shell inside the container (or use Container App Console debug tools) and test DNS resolution using `nslookup` or `dig`:
    ```bash
    nslookup api.github.com
    ```
2.  **Inspect resolv.conf:** Check the DNS configuration of the container:
    ```bash
    cat /etc/resolv.conf
    ```
    *Verify if the nameserver points to the expected internal DNS resolver (like `168.63.129.16` for Azure DNS, or CoreDNS for Kubernetes).*
3.  **Verify VNet DNS settings:** In Azure, if the VNet is configured with "Custom DNS" servers (e.g. self-hosted Active Directory DNS) but those custom servers don't have public forwarders configured, the containers will fail to resolve public domains.
4.  **Network Security Group (NSG) Block:** Check if NSG rules or Azure Firewall are blocking outbound UDP/TCP port 53 (DNS traffic) from the container's subnet.

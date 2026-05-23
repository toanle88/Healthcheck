# DevOps & SRE Interview Preparation Guide (Healthcheck Project)

This guide contains realistic, tough technical interview questions and strategic answers based on the architecture, security, and CI/CD patterns implemented in the **Healthcheck Dashboard** project.

Use this guide to prepare for transitioning from a Backend Software Engineer to a DevOps/SRE/Platform Engineering role.

---

## 🔑 Category 1: Cloud Identity & Security

### Q1: You mentioned your project has a "zero-secret" architecture. How did you implement this, and why is it better than traditional API keys or connection strings?
*   **Context:** Testing your knowledge of modern cloud security best practices (least privilege & identity-based access control).
*   **Answer:**
    > "Traditionally, applications connect to databases or retrieve secrets from vaults using hardcoded passwords or API keys stored in configuration files. If those config files are leaked or checked into Git, the entire system is compromised.
    >
    > In this project, I implemented **Azure User-Assigned Managed Identities** for the container workloads. Instead of storing a database connection string with a password or an Azure Key Vault API key, the Azure Container App runs under the identity of a specific Azure AD (Entra ID) Principal. 
    > 
    > When the Go API starts, it uses the official Azure SDK to request an Entra ID token automatically. It then uses this token to authenticate directly with **PostgreSQL Flexible Server** and **Azure Key Vault** using Azure RBAC (Role-Based Access Control) with least privilege (e.g. assigning the *'Key Vault Secrets User'* role to the identity). There are zero long-lived credentials stored in our code, environment variables, or databases."

### Q2: How did you secure your CI/CD pipelines connecting to Azure? Did you store Azure Service Principal client secrets in GitHub?
*   **Context:** Evaluates pipeline credential management and awareness of OIDC federation.
*   **Answer:**
    > "No, I did not store any Azure passwords or client secrets in GitHub or Azure DevOps. Instead, I configured **OpenID Connect (OIDC) Federated Credentials**. 
    >
    > In Azure, I created a User-Assigned Managed Identity and established a trust relationship with GitHub Actions and Azure DevOps. When the pipeline runs, GitHub issues a short-lived OIDC JSON Web Token (JWT). The runner presents this token to Azure Active Directory, which validates the signature and exchanges it for a short-lived Azure access token (valid for 1 hour). 
    > 
    > This zero-secret handshake prevents credential theft from the CI/CD platform and removes the operational overhead of rotating service principal keys."

---

## 🌐 Category 2: Networking & Network Isolation

### Q3: What is the difference between how you configured networking in your Development environment versus your Production environment?
*   **Context:** Probes your understanding of cost-vs-security tradeoffs and enterprise network security.
*   **Answer:**
    > "In the **Development** environment, we prioritize cost and simplicity. While resources are placed inside a VNet, we allow public network access to the Key Vault, and the Network Security Group (NSG) allows both HTTP (80) and HTTPS (443) traffic from the public internet.
    >
    > In **Production**, we enforce a strict 'Default Deny' network security model:
    > 1. We disabled public network access to the **Azure Key Vault** and exposed it privately inside our virtual network using a **Private Endpoint (Private Link)** in a dedicated subnet (`snet-endpoints`).
    > 2. We disabled all public access to the **PostgreSQL Flexible Server** and delegated its subnet (`snet-db`) directly to the PostgreSQL resource, using Private DNS Zones for resolution.
    > 3. The production Network Security Group (NSG) strictly **denies all plain HTTP (80) traffic**, accepting only encrypted HTTPS (443) traffic.
    > 
    > This ensures that in production, sensitive data and secrets are never exposed on the public internet, and components only communicate over secure, private channels."

### Q4: Why did you use Private Endpoints for the Key Vault instead of just Service Endpoints?
*   **Context:** A classic Azure networking question checking if you understand the security difference.
*   **Answer:**
    > "While Service Endpoints keep traffic on the Microsoft backbone network, the resource still retains a public IP address, and firewall rules are required to restrict access to specific subnets. Furthermore, Service Endpoints do not protect against data exfiltration, as an attacker could theoretically route traffic to a different Key Vault resource under their control.
    > 
    > **Private Endpoints**, on the other hand, assign a private IP address directly from our VNet subnet to the Key Vault. The public DNS name resolves to this private IP using an Azure Private DNS Zone. This disables public network routing entirely, forces all traffic through our internal virtual network, and prevents data exfiltration by binding the endpoint to a specific, unique resource instance."

---

## 📦 Category 3: Containerization & Architecture

### Q5: I see you used Go for the backend. How did you structure your Dockerfiles to ensure security and performance?
*   **Context:** Tests container security compliance, multi-stage builds, and size optimization.
*   **Answer:**
    > "I used **multi-stage Docker builds** combined with **distroless** base images.
    > 
    > 1. In the first builder stage, I use the full Go SDK image to compile the application binary. I ensure security flags are set and dependencies are cached.
    > 2. In the second deployment stage, I copy *only* the compiled static binary and necessary certificates into a `gcr.io/distroless/static-debian12` base image.
    > 3. I explicitly configure the container to run under a non-root user (`USER nonroot:nonroot`).
    > 
    > This achieves two major benefits:
    > * **Minimal attack surface:** Distroless images contain no package managers (like `apt`), shell utilities (`bash`/`sh`), or extra tools. If an attacker gains access to the container, they have no utilities to run exploits or scan the network.
    > * **Small Footprint:** The final image size is only about 12MB, which dramatically speeds up pipeline build/push times and container startup scaling in production."

### Q6: Why did you choose Azure Container Apps (ACA) instead of Azure Kubernetes Service (AKS) for hosting this application?
*   **Context:** Evaluates system design judgment and cost/complexity management.
*   **Answer:**
    > "Choosing the right level of abstraction is a key DevOps skill. Azure Kubernetes Service (AKS) is extremely powerful, but it carries high operational complexity (managing node pools, ingress controllers, upgrades) and high baseline costs (paying for control plane nodes and system VMs).
    > 
    > For a small business or a small microservice stack like this, **Azure Container Apps (ACA)** is a much better fit. ACA is a serverless container platform built on top of AKS, KEDA (Kubernetes Event-driven Autoscaling), and Envoy. It gives us:
    > * Out-of-the-box scaling to zero (saving 100% compute cost when idle).
    > * Simple ingress and DNS management.
    > * Built-in blue-green deployments (traffic splits).
    > * Zero cluster management overhead, allowing us to spend 90% of our time delivering application features rather than managing Kubernetes infrastructure."

---

## 🔄 Category 4: CI/CD & Deployments

### Q7: If a deployment to Production fails its health checks, how does your pipeline handle the situation?
*   **Context:** Tests release safety patterns and automated rollback experience.
*   **Answer:**
    > "The release pipeline utilizes a **smoke-testing and rollback strategy** built directly into the CD workflows.
    > 
    > When a new container image is deployed to Azure Container Apps, the pipeline runs an automated smoke-testing script that repeatedly pings the app's health endpoint (`/health`).
    > 
    > If the smoke test fails or times out (indicating a crash loop or configuration error):
    > 1. The pipeline script detects the non-zero exit code.
    > 2. It immediately executes a rollback command to revert the Container App's traffic allocation (100% traffic split) back to the last known healthy revision.
    > 3. It alerts the engineering team via webhook and blocks the pipeline promotion, ensuring that users never experience downtime due to broken releases."

### Q8: How do you enforce quality gates in your pipeline before code reaches production?
*   **Context:** Evaluates pipeline design, testing integrations, and security auditing.
*   **Answer:**
    > "We run a strict, automated **Audit and Test stage** on every Pull Request:
    > 1. **Linting & Formatting:** We run Go linter/format checkers to ensure clean code standards.
    > 2. **Security Scans:** We run **Trivy** to scan container filesystems and dependencies for known CVEs, and **Checkov** to run static analysis on our Terraform/Bicep templates.
    > 3. **Unit & Integration Testing:** We run the full test suites.
    > 4. **Coverage Gates:** The CI pipeline parses the test coverage output (`coverage.out`). If the code coverage falls below our target threshold (e.g. 80%), the pipeline fails and blocks the PR from being merged.
    > 5. **Manual Approvals:** Deployment to the production environment requires a manual approval sign-off in the pipeline UI (GitHub Environments or Azure DevOps Release approvals) after passing the dev deployment smoke tests."

---

## 📊 Category 5: Observability & Tracing

### Q9: Explain how W3C distributed tracing works in your Go services, and why we need it.
*   **Context:** Tests microservices debugging, OpenTelemetry, and observability expertise.
*   **Answer:**
    > "In a distributed system, a single user request can touch several microservices. If that request fails or is slow, it is very difficult to find the culprit by looking at isolated logs. Distributed tracing links these requests together.
    > 
    > In this project, I used the **OpenTelemetry Go SDK**. When the React frontend makes an API call, or the API triggers the background Worker, OpenTelemetry injects a standard W3C header called `traceparent` (containing a unique Trace ID) into the HTTP headers or message metadata.
    > 
    > The receiving service extracts this header and starts its own tracing 'spans' under the same Trace ID. When these spans are sent to our APM (Jaeger locally, or Azure Application Insights in production), they are stitched together into a single timeline chart. This lets us visualize the exact latency breakdown of the request across all microservice boundaries."

### Q10: How did you configure OpenTelemetry to export to Jaeger locally but Application Insights in the cloud without modifying the application code?
*   **Context:** Evaluates clean code principles and architecture portability.
*   **Answer:**
    > "I designed the telemetry initialization code to be **environment-driven**. 
    > 
    > The Go application uses the vendor-neutral OpenTelemetry SDK. In our telemetry startup script, the code checks for the presence of the `APPLICATIONINSIGHTS_CONNECTION_STRING` environment variable.
    > 
    > * If it is present (in Cloud Dev/Pro), the code configures the OpenTelemetry OTLP HTTP trace and metric exporters to ship data to the Azure Monitor OTLP ingestion endpoints (`/v2.1/otlp/v1/traces`).
    > * If it's missing or the environment is local, it appends options to point to the local Jaeger collector endpoint (`jaeger:4318`) over HTTP and exposes a Prometheus scrape endpoint.
    > 
    > This keeps our application binary fully portable. We can swap monitoring backends (e.g., migrating from Azure App Insights to Datadog or AWS X-Ray) simply by changing environment variables, without modifying a single line of Go source code."

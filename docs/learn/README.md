# Healthcheck Technology Learning Path 📚

Welcome to the **Healthcheck** repository's interactive technology learning path! This project serves as a hands-on sandbox for modern, cloud-native DevOps engineering on Microsoft Azure.

Rather than just reading theoretical docs, this path guides you through the concrete implementation details of the application, infrastructure, and automation layers.

## 🧭 The Learning Roadmap

The lessons are divided into two main categories:

### 🏗️ Foundation & Cloud Architecture
These lessons cover the system-level design, code, and hosting setup.
*   **[01: Architecture Overview](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/01-architecture-overview.md)**
    *   Learn the high-level system components, identity structures, security boundaries, and data flows visualized with interactive Mermaid diagrams.
*   **[02: Go Microservices & SSE](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/02-go-microservices.md)**
    *   Understand the Go codebase layout, passwordless PostgreSQL connections using JWT tokens, secure Auth middleware, real-time Server-Sent Events (SSE) streaming, and embedded Scalar API documentation.
*   **[03: Infrastructure as Code (Terraform & Bicep)](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/03-infrastructure-as-code.md)**
    *   Explore how to provision resources safely. Learn module structure, CIDR subnetting, security groups (NSGs), and OIDC-based Azure credentials.
*   **[04: Cloud Hosting with Azure Container Apps](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/04-azure-container-apps.md)**
    *   Deep dive into Azure Container Apps (ACA), private Virtual Networks, internal DNS resolution, and executing one-time or recurring Container Jobs.

### 🔒 Hardening, Observability, and Automation
These lessons dive into advanced security, distributed tracing, and quality gates.
*   **[05: CI/CD & Image Hardening](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/05-cicd-and-security.md)**
    *   Learn about passwordless GitHub pipelines (OIDC), Checkov static IaC checks, Distroless Docker images, Trivy vulnerability scanning, and SSRF prevention.
*   **[06: W3C Distributed Tracing](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/06-w3c-distributed-tracing.md)**
    *   Understand how context propagation works end-to-end under the W3C spec, tracing a background worker's ping request all the way through the API's middleware and database operations.
*   **[07: Entra ID Passwordless Security](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/07-entra-id-passwordless.md)**
    *   Deep dive into authentication without credentials, detailing Azure User-Assigned Managed Identity, automatic token refreshing in `pgx`, and API role-based access control (RBAC).
*   **[08: Network Isolation & Private Endpoints](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/08-network-isolation-security.md)**
    *   Explore how we enforce complete network isolation in production using delegated subnets, private endpoints for Key Vault/Postgres, and restrictive NSGs.
*   **[09: CI/CD Quality Gates & Automated Rollbacks](file:///mnt/d/Dev/Projects/Healthcheck/docs/learn/09-cicd-quality-gates.md)**
    *   Learn how quality gates (e.g., coverage minimums) block bad builds, and how release pipelines perform active health checks to trigger container rollbacks.

---

## 🚀 How to Use This Path
1.  **Read and Trace**: As you read each guide, follow the links directly to the files in the codebase to see the implementation in action.
2.  **Run Locally**: Run the services locally using Docker Compose to see metrics and logs propagate.
3.  **Inspect Telemetry**: Open the Grafana and Jaeger dashboards locally to witness W3C tracing and Prometheus metrics live.

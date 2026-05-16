# Healthcheck Dashboard — DevOps Playground (Azure + Go)

A tiny, production-like app built specifically to learn **CI/CD, Terraform, Docker, monitoring, and security on Azure using Go**. The app pings 3 public APIs every minute and shows green/red status, so you spend 90% of your time on infra, not features.

See `PROJECT.md` for full specs and `ROADMAP.md` for the 14-day plan.

## 🎯 Learning Goals

- **Go**: idiomatic HTTP servers, context cancellation, structured logging
- **Docker**: multi-stage builds with distroless, ~12MB images, non-root user
- **Terraform**: AzureRM for VNet, Container Apps, PostgreSQL Flexible Server, Key Vault, Managed Identity, ACR
- **CI/CD**: GitHub Actions with Azure OIDC
- **Observability**: OpenTelemetry Go SDK → Azure Monitor / Application Insights
- **Security**: Key Vault, RBAC least privilege, Trivy scanning, Defender for Cloud

## 🏗️ Architecture (Azure)

```
Browser → Azure Container Apps (Web) → Entra External ID (CIAM)
                          ↓
                   Azure Container Apps (API) → Go API
                          ↓
                   Azure Container Apps Job → Go Worker → Azure Database for PostgreSQL
```

This project uses a **"Clean Split"** architecture: Core infrastructure is automated via Terraform, while Customer Identity (CIAM) is managed as a curated one-time setup for maximum stability.

## 🛠️ Development & Quality

### Code Formatting
This project strictly enforces Go standards. To fix any formatting issues before pushing to GitHub, run:
```powershell
go fmt ./...
```

### Local Testing
To run the full suite of unit tests:
```powershell
go test -v -race ./...
```

### CI/CD Pipelines
*   **🛡️ CI**: Every Pull Request triggers an automated audit of code quality (linting), unit tests, and security scans (Trivy).
*   **🚀 CD**: Every merge to `main` builds new Docker images, pushes them to Azure Container Registry (ACR), and updates the infrastructure via Terraform.
```mermaid
graph TD
    subgraph Azure ["Azure Cloud"]
        subgraph VNet ["Virtual Network (vnet-healthcheck)"]
            subgraph AppSubnet ["App Subnet (snet-apps)"]
                API["API (Container App)"]
                Worker["Worker (Container Job)"]
            end
            subgraph DBSubnet ["DB Subnet (snet-db)"]
                DB[("PostgreSQL Flexible Server")]
            end
        end
        ACR["Container Registry (ACR)"]
        KV["Key Vault (Secrets)"]
        ID["Managed Identity"]
    end

    GitHub["GitHub Actions"] -- OIDC --> ID
    ID -- Deploy --> API
    API -- AAD Token --> DB
    Worker -- AAD Token --> DB
    API -- Managed ID --> KV
```

### Infrastructure as Code (Terraform)
We use a modular Terraform structure for maximum maintainability:

| Module | Purpose |
| :--- | :--- |
| `modules/identity` | OIDC federation and User-Assigned Managed Identity. |
| `modules/network` | VNet, Subnets, and Private DNS for network isolation. |
| `modules/postgres` | Flexible Server with Azure AD Authentication (Passwordless). |
| `modules/containerapp` | API, Web, and Worker Job with Scale-to-Zero and Blue-Green. |
| `modules/keyvault` | Secure secret storage for app configuration. |
| `modules/monitor` | Log Analytics and App Insights for full observability. |

---

## 🛠️ Tech Stack & Security

- **OIDC Authentication**: Passwordless GitHub login to Azure using Workload Identity Federation.
- **Identity (CIAM)**: Entra External ID for customer-facing authentication.
- **Backend API**: Go 1.23, Gin, pgx/v5 (Structured logging with `slog`)
- **Infrastructure**: Terraform ≥1.7 (Modular setup)

## 🚀 Deployment Guide

For a full "Clean Split" deployment walkthrough, see the **[DEPLOYMENT GUIDE](./docs/DEPLOYMENT_GUIDE.md)**.

### 🔐 Environment Files

**`.env.azure`** — Azure credentials for local Terraform (NEVER commit)
```bash
export ARM_SUBSCRIPTION_ID="your-id"
export ARM_TENANT_ID="your-main-tenant-id"
export ARM_CLIENT_ID="your-id-github-actions-bootstrap"
export ARM_USE_OIDC=true

# CIAM Configuration
export TF_VAR_entra_client_id="your-ciam-app-id"
```

Usage:
```bash
source .env          # for go run
source .env.azure    # for terraform
```


- **Backend API**: Go 1.26, Gin, pgx/v5 (Structured logging with `slog`)
- **Worker**: Go 1.26, robfig/cron, shared Postgres store
- **Frontend**: React 19 + Vite + TypeScript + Tailwind CSS 4 + React Query (TanStack) + Axios
- **Database**: PostgreSQL 18 (Local) / Azure Database for PostgreSQL Flexible Server (Cloud)
- **Testing**: Vitest + MSW (FE Unit/Integration), Playwright (E2E), Go Testing (BE Unit/Integration)
- **Containers**: Docker Compose (Local), Azure Container Apps (Cloud)
- **Infra**: Terraform ≥1.7
- **CI/CD**: GitHub Actions with Azure OIDC

## 📁 Repository Structure

```
.
├── cmd/
│   ├── api/          # HTTP server
│   └── worker/       # Cron worker
├── internal/
│   ├── config/       # env + Key Vault loading
│   ├── handler/      # HTTP handlers
│   ├── store/        # postgres queries
│   └── monitor/      # otel setup
├── web/              # React frontend (Modular architecture: hooks, components, pages, services)
├── infra/
│   ├── modules/
│   │   ├── network/
│   │   ├── containerapp/
│   │   ├── postgres/
│   │   └── keyvault/
│   └── envs/dev/
├── .github/workflows/
│   ├── ci.yml
│   └── cd.yml
├── Dockerfile.api
├── Dockerfile.worker
├── docker-compose.yml
├── go.mod
└── PROJECT.md
```

## 🎓 Learning Center

If you want to understand how this project works, follow our **Masterclass Curriculum**:

1. [Lesson 01: Architecture Overview](./docs/lessons/01-architecture-overview.md) — The "Big Picture."
2. [Lesson 02: Go Microservices](./docs/lessons/02-go-microservices.md) — Identity-aware Go code.
3. [Lesson 03: Infrastructure as Code](./docs/lessons/03-infrastructure-as-code.md) — The Terraform blueprint.
4. [Lesson 04: Azure Container Apps](./docs/lessons/04-azure-container-apps.md) — Scaling & Resilience.
5. [Lesson 05: CI/CD & Security](./docs/lessons/05-cicd-and-security.md) — Automating the "Castle."

---

## 🚀 Quick Start (Local Development)

### 1. Launch the Full Stack
This project is fully containerized. You can start the API, Worker, Database, and Frontend with a single command:

```bash
docker-compose up --build
```

- **Dashboard**: [http://localhost:5173](http://localhost:5173)
- **API Health**: [http://localhost:8080/health](http://localhost:8080/health)
- **API Status**: [http://localhost:8080/api/status](http://localhost:8080/api/status)

### 📊 Observability Dashboards
This project includes a full-stack observability suite:

- **Traces (Jaeger)**: [http://localhost:16686](http://localhost:16686)
  - View the "journey" of every request and background ping.
- **Metrics (Prometheus)**: [http://localhost:9090](http://localhost:9090)
  - **API Metrics**: [http://localhost:8080/metrics](http://localhost:8080/metrics)
  - **Worker Metrics**: [http://localhost:8081/metrics](http://localhost:8081/metrics)
  - Try querying: `healthcheck_status_total` or `healthcheck_latency_seconds_bucket`.
  - **P95 Latency Query**: `histogram_quantile(0.95, sum(rate(healthcheck_latency_seconds_bucket[5m])) by (le, target))`

### 2. Verify your Environment
Run the validation script to ensure linting and tests are passing:

```powershell
# Windows
./check.ps1

# Linux/macOS
chmod +x check.sh
./check.sh
```

### 3. Manual Frontend Development
If you want to run the frontend outside of Docker with Hot Module Replacement (HMR):
```bash
cd web
pnpm install
pnpm run dev
```

## ☁️ Quick Start (Azure)

1. **Prereqs**: Azure CLI, Terraform ≥1.8, Go 1.23

2. **OIDC Bootstrap**:
   - Run Terraform in `infra/bootstrap` to create the ACR and the OIDC Service Principal.
   - Configure your GitHub repository with the `AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, and `AZURE_SUBSCRIPTION_ID`.

3. **Deploy Infrastructure**:
   ```bash
   cd infra/envs/dev
   terraform init
   terraform plan
   terraform apply
   ```

4. **Push to main**: CI/CD handles the rest via OIDC.

5. **Push to main**: CI builds images, pushes to ACR, updates Container Apps

> Do not commit `.env` or `.env.azure`. Add both to `.gitignore`.

## 🔄 CI/CD Flow

**ci.yml (PR)**:
- go vet, go test -race, golangci-lint
- trivy fs --severity HIGH,CRITICAL
- terraform fmt -check && terraform validate

**cd.yml (main)**:
1. Setup Go & Audit (gofmt, vet, test, trivy)
2. generate short Git SHA for image tags
3. azure/login@v2 with Service Principal secrets
4. docker build and push to ACR
5. az containerapp update (API & Web) with the new SHA tag

## 📊 Observability

- Logs: `log/slog` with JSON handler → stdout → Log Analytics
- Traces: OpenTelemetry Go SDK → Application Insights
- Metrics: runtime metrics + custom `api_ping_duration_seconds`
- Alerts: P95 latency > 500ms, error rate >1%, worker job failed

## 🛡️ Security Checklist

- [x] Dockerfile uses distroless, USER nonroot
- [x] Checkov security auditing (Infra-as-Code compliance)
- [x] Zero-Secret Runtime: Managed Identity for Postgres and Key Vault
- [x] Network Isolation: Postgres VNet Injection + Private DNS
- [x] Hardened Ingress: HTTPS-only, CORS restricted, Port 22 blocked (NSG)
- [x] Cost Optimization: Scale-to-Zero for API and Web
- [x] Resilience: Blue-Green deployments with automatic rollback
- [x] OIDC Authentication: Secretless GitHub Actions deployment

## 📅 14-Day Roadmap

This project is designed as a 2-week playground. See `ROADMAP.md` for daily tasks.

**Week 1 – Local Go + Docker** ✅
- [x] API skeleton, worker, React frontend
- [x] Hardened Dockerfiles, slog JSON, OpenTelemetry

**Week 2 – Azure + Terraform + CI/CD** ✅
- [x] Terraform for network, ACR, Key Vault, Postgres, Container Apps
- [x] GitHub Actions CI/CD with OIDC (Passwordless)
- [x] Application Insights, alerts, and dashboards
- [x] Chaos engineering (Poison Pill) and automatic rollback
- [x] Stretch goals: Blue-Green deploys, Auto-Scale to Zero

Stretch goals: blue-green deploys, auto-scale to zero, multi-region with Front Door.

## 🧹 Cleanup

Estimated cost if left running: $5-12/month in dev.

```bash
source .env.azure
cd infra/envs/dev
terraform destroy -auto-approve
```

## 🤖 Using with Meta AI

When asking for help, upload:
1. The file you're editing (e.g., `internal/config/config.go`)
2. `PROJECT.md`
3. One related example

Example: "Based on Dockerfile.api, generate Dockerfile.worker for ./cmd/worker with same distroless hardening."

---
Built to learn DevOps with Go on Azure, by doing, not watching.

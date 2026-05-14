# Healthcheck Dashboard — DevOps Playground (Azure + Go)

A tiny, production-like app built specifically to learn **CI/CD, Terraform, Docker, monitoring, and security on Azure using Go**. The app pings 3 public APIs every minute and shows green/red status — so you spend 90% of your time on infra, not features.

## 🎯 Learning Goals
- Go: idiomatic HTTP servers, context cancellation, structured logging
- Docker: multi-stage builds with distroless, ~12MB images, non-root user
- Terraform: AzureRM for VNet, Container Apps, PostgreSQL Flexible Server, Key Vault, Managed Identity, ACR
- CI/CD: GitHub Actions with Azure OIDC (no long-lived secrets)
- Observability: OpenTelemetry Go SDK → Azure Monitor / Application Insights
- Security: Key Vault, RBAC least privilege, Trivy scanning, Defender for Cloud

## 🏗️ Architecture (Azure)
```
Browser → Azure Front Door + Azure Static Web Apps (React/Vite)
                           ↓
                    Azure Container Apps → Go API
                           ↓
                    Azure Container Apps Job → Go Worker → Azure Database for PostgreSQL Flexible Server
```
All secrets retrieved from Azure Key Vault using Managed Identity.

## 📦 Tech Stack
- **Backend API**: Go 1.26, Gin, pgx/v5, slog
- **Worker**: Go 1.26, robfig/cron, shared postgres store
- **Frontend**: React 19 + Vite + TypeScript + Tailwind CSS 4 + React Query + Axios
- **Database**: PostgreSQL 18
- **Testing**: Vitest + MSW (Frontend), Playwright (E2E), Go Testing (Backend)
- **Infra**: Terraform ≥1.7
- **CI/CD**: GitHub Actions
- **Containerization**: Docker Compose (Dev), Distroless (Prod)
- **Monitoring**: Application Insights, Log Analytics, Azure Monitor
- **Security**: Azure Key Vault, Microsoft Defender for Cloud, Trivy

## 📁 Repo Structure
```
.
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── config/      # env + Key Vault loading
│   ├── handler/     # HTTP handlers
│   ├── store/       # postgres queries
│   └── monitor/     # otel setup
├── web/             # React frontend (Hooks, Components, Pages, Services)
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
├── go.sum
└── PROJECT.md
```

## 🚀 Quick Start (Local)
```bash
git clone <repo>
cd healthcheck
docker-compose up --build
# API: http://localhost:8080/health
# Web: http://localhost:5173
```

docker-compose.yml runs Postgres + api + worker locally.

## 🐹 Go Dockerfiles

**Dockerfile.api**
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/api /api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
```

Same pattern for worker.

## 🔄 CI/CD Flow
**ci.yml (PR)**
- go vet ./...
- go test ./... -race
- golangci-lint
- trivy fs --severity HIGH,CRITICAL .
- terraform fmt -check && terraform validate

**cd.yml (main)**
1. Setup Go & Audit (gofmt, vet, test, trivy)
2. azure/login@v2 with OIDC
3. docker build & push for API, Worker, and Web with $GITHUB_SHA tags
4. az containerapp update (API & Web) with the new SHA tag
5. az containerapp job update (Worker) with the new SHA tag

## 🛡️ Security Checklist
- [ ] Dockerfile uses distroless, USER nonroot
- [ ] Trivy scan passes in CI
- [ ] No secrets in repo — use Managed Identity + Key Vault
- [ ] PostgreSQL: private endpoint only, SSL enforced
- [ ] Container App identity has only "Key Vault Secrets User" role
- [ ] Defender for Cloud enabled on ACR and Container Apps
- [ ] WAF enabled on Front Door

## 📊 Observability (Go)
```go
import (
  "go.opentelemetry.io/otel"
  "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

tracer := otel.Tracer("healthcheck-api")
handler := otelhttp.NewHandler(router, "api")
```
- Logs: slog with JSON handler → stdout → Log Analytics
- Metrics: otel runtime metrics + custom `api_ping_duration_seconds`
- Alerts in Azure Monitor: latency P95 > 500ms, error rate >1%

## 🧪 Terraform Layout
```hcl
# infra/envs/dev/main.tf
provider "azurerm" { features {} }

module "network" {
  source = "../../modules/network"
}

module "acr" {
  source = "../../modules/acr"
}

module "keyvault" {
  source = "../../modules/keyvault"
}

module "postgres" {
  source              = "../../modules/postgres"
  private_vnet_id     = module.network.vnet_id
}

module "containerapp_api" {
  source          = "../../modules/containerapp"
  image           = "${module.acr.login_server}/api:latest"
  keyvault_id     = module.keyvault.id
  postgres_fqdn   = module.postgres.fqdn
}
```

## 📚 Learning Path (2 weeks)
**Week 1 – Local Go + Docker**
- Day 1-2: `go mod init`, write /health handler, pgx connection
- Day 3-4: Write worker that pings APIs, multi-stage Dockerfiles
- Day 5-7: Add OpenTelemetry, structured slog, docker-compose

**Week 2 – Azure + Terraform + CI/CD**
- Day 8-9: Terraform for network, ACR, Key Vault
- Day 10-11: GitHub Actions with azure/login OIDC, build & push
- Day 12-13: Deploy to Container Apps, wire Managed Identity to Key Vault
- Day 14: Add Application Insights, create alerts, break DB and test rollback

## 🤖 Using Meta AI Web
Upload when prompting:
- `internal/config/config.go`
- `Dockerfile.api`
- One Terraform module

Prompt: "Based on our Go Dockerfile, generate Dockerfile.worker for the worker binary at ./cmd/worker, same distroless hardening."

## 🧹 Cleanup
```bash
terraform destroy -auto-approve
```
Estimated cost: $5-12/month if left running in dev.


---
Built to learn DevOps with Go on Azure — by doing, not watching.

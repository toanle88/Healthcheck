# ROADMAP.md — Healthcheck Dashboard (Go + Azure)

This roadmap breaks the 2-week DevOps playground into daily, shippable milestones. Each day ends with something you can run, test, or deploy. Follow in order — don't skip security steps.

## Phase 0 — Setup (Day 0)
**Goal:** Clean repo, tooling ready
- [x] `git init`, create repository structure
- [x] Install: Go 1.26, Docker, Terraform ≥1.7, Azure CLI, golangci-lint, Trivy
- [x] `az login` and create dev subscription/resource group
- [x] Create GitHub repo, enable Actions, add OIDC federated credential for Azure

**Deliverable:** Empty repo pushes successfully

---

## Phase 1 — Local Go App (Days 1-3)
**Goal:** Working app locally, no cloud yet

### Day 1 — API skeleton
- [x] `go mod init github.com/toanle88/healthcheck`
- [x] `cmd/api/main.go`: Gin server, `/health`, `/api/status`, `/api/history`
- [x] `internal/store/postgres.go`: pgx pool with context timeout
- [x] `docker-compose.yml`: postgres:18 only
- [x] Test: `go run ./cmd/api` → curl localhost:8080/health

### Day 2 — Worker
- [x] `cmd/worker/main.go`: cron every 60s, pings 3 APIs (httpbin, github, azure status)
- [x] Store results in Postgres table `checks(id, target, status, latency_ms, checked_at)`
- [x] Add `internal/config` to load from env

### Day 3 — Frontend
- [x] `web/`: Vite + React, fetch `/api/status`, show green/red cards
- [x] Update docker-compose to include api + worker + web
- [x] Deliverable: `docker-compose up` shows dashboard

---

## Phase 2 — Docker Hardening (Days 4-5)
**Goal:** Production-grade containers

### Day 4
- [x] Write `Dockerfile.api` and `Dockerfile.worker` (multi-stage, distroless, nonroot)
- [x] Image size < 20MB
- [x] Add HEALTHCHECK in Dockerfile
- [x] Run Trivy locally: `trivy image healthcheck-api`

### Day 5
- [x] Add structured logging with `log/slog` JSON
- [x] Add OpenTelemetry SDK, export to stdout initially
- [x] Add `/metrics` endpoint with otel runtime metrics
- [x] Deliverable: `docker images` shows hardened images, logs are JSON

---

## Phase 3 — Terraform Foundation (Days 6-8)
**Goal:** Infra as Code for Azure

### Day 6 — Network + ACR
- [x] `infra/modules/network`: VNet, 2 subnets (private, containerapps)
- [x] `infra/modules/acr`: Azure Container Registry with admin disabled
- [x] `terraform apply` in dev (Modular refactor complete)

### Day 7 — Data + Secrets
- [x] `infra/modules/postgres`: PostgreSQL Flexible Server, private endpoint, SSL enforced
- [x] `infra/modules/keyvault`: Key Vault with RBAC, store DB connection string
- [x] Enable Managed Identity for future Container Apps (Root module prepared)

### Day 8 — Container Apps
- [x] `infra/modules/containerapp`: Container Apps Environment, API app, Worker Job
- [x] Wire Managed Identity → Key Vault access
- [x] Output URLs
- [x] Deliverable: `terraform apply` creates empty infra

---

## Phase 4 — CI/CD (Days 9-11)
**Goal:** Push to main = deploy

### Day 9 — CI pipeline
- [x] `.github/workflows/ci.yml`: on PR
- [x] go vet, go test -race, golangci-lint (Tests configured)
- [x] trivy fs (Security scanning active)
- [x] terraform fmt, validate, plan (Infra checks active)

### Day 10 — CD pipeline
- [x] Configure Azure OIDC in GitHub (Identity built in Day 6)
- [x] `.github/workflows/cd.yml`: on main
- [x] docker buildx build & push to ACR (Multi-stage build ready)
- [x] terraform apply -auto-approve (Automated infra updates)
- [x] az containerapp update (Automated code deployment)

### Day 11 — First deploy
- [x] Merge to main, watch Actions
- [x] Verify API in Azure, check logs in Log Analytics
- [x] Deliverable: Live URL works

---

## Phase 5 — Observability (Day 12)
**Goal:** Know when it breaks
- [x] Instrument Go with otel → Application Insights
- [x] Create Log Analytics workspace via Terraform
- [x] Add Azure Monitor alerts:
  - P95 latency > 500ms (5 min)
  - Error rate > 1%
  - Worker job failed
- [x] Create dashboard in Azure Portal
- [x] Deliverable: Trigger 500 error, alert fires

---

## Phase 6 — Security Hardening (Day 13)
**Goal:** Production-ready security
- [x] Rotate to Managed Identity for Postgres (no password in Key Vault)
- [x] Run Checkov: `checkov -d infra/`


---

## Phase 7 — Chaos & Rollback (Day 14)
**Goal:** Prove reliability
- [x] Push bad image (panic on start), verify automatic rollback
- [x] Test `terraform destroy` and rebuild from scratch in <15 min


---

## Phase 8 — Stretch Goals (After Day 14)
Pick one based on interest:
- [x] **Blue-Green:** Two Container App revisions, traffic split via Terraform
- [x] **Cost:** Auto-scale to zero, schedule worker to stop at night


---

## Phase 9 — Production Enhancements (All 4 Options) ✅
Implement full-stack improvements to make the application highly interactive and SRE-ready:
- [x] **Option 1: Dynamic Healthcheck Targets (CRUD)**
  - [x] Database schema extension (add `targets` table)
  - [x] REST API endpoints (`GET /api/targets`, `POST /api/targets`, `DELETE /api/targets/:id`)
  - [x] Cron worker integration (pull ping targets dynamically from DB)
  - [x] React CRUD UI to manage targets dynamically
- [x] **Option 2: Uptime & Latency History Visualization**
  - [x] Implement `GET /api/history` with time-series DB queries
  - [x] Optimize Postgres with index on `(target, checked_at)`
  - [x] Render interactive latency graphs and state grids (Recharts / SVG) on Dashboard
- [x] **Option 3: State-Transition Alerting**
  - [x] Add Slack or Discord Webhook integration via environment config
  - [x] Detect state changes (`up` -> `down`, `down` -> `up`) in cron worker
  - [x] Send webhook notification payloads with incident/recovery details
- [x] **Option 4: 24h SLA & Uptime Percentages**
  - [x] Write SQL calculation logic for 24h/7d uptime percentages
  - [x] Display SLA progress meters/percentage badges on the React frontend
- [x] **Option 5: Hardened Synthetic Monitoring Features**
  - [x] Database schema extension (method, headers, expected_status, response_contains)
  - [x] API validator and creator support in backend handlers
  - [x] Worker ping engine supporting dynamic methods, HTTP header injection, expected status validation, and body substring matching
  - [x] Expanded React dashboard creation form UI and current targets list configuration view
- [x] **Option 6: Worker Robustness & Alert De-noising**
  - [x] Database schema extension for consecutive failure thresholds and alert state tracking
  - [x] Support customizing failure thresholds in React dashboard creation form and list view
  - [x] Introduce randomized worker ping jitter (0–15s) in batch scheduling
  - [x] Implement consecutive failure threshold-based webhook alerting to eliminate false positives from transient network flapping
- [x] **Option 7: Security & RBAC Hardening**
  - [x] Configure custom `CheckRedirect` on the worker's HTTP client to prevent SSRF by blocking redirects to internal/private IPs and limiting redirect hops (max 3).
  - [x] Enhance the API auth middleware to propagate JWT claims via the Gin context and implement a new `RequireRoleOrScope` role/scope validation middleware.
  - [x] Secure `POST /api/targets` and `DELETE /api/targets/:id` by requiring the `Healthcheck.Admin` role.

---

## Daily Workflow with AI Assistant
For each task, upload:
1. Current file you're editing
2. One example or schema file from repo

Example prompt for Day 4:
> "Based on Dockerfile.api in this project, create Dockerfile.worker for cmd/worker/main.go using same distroless hardening. Output only Dockerfile."

---

## Success Criteria
By end of roadmap you can:
1. Explain every line of Terraform
2. Deploy with `git push` only
3. Show Application Insights traces for a request
4. Demonstrate secret rotation without redeploy
5. Roll back a bad deploy in <2 minutes

This is the exact skillset for junior DevOps/SRE roles.

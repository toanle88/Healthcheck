# ROADMAP.md — Healthcheck Dashboard (Go + Azure)

This roadmap breaks the 2-week DevOps playground into daily, shippable milestones. Each day ends with something you can run, test, or deploy. Follow in order — don't skip security steps.

## Phase 0 — Setup (Day 0)
**Goal:** Clean repo, tooling ready
- [x] `git init`, create repo structure from PROJECT.md
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
- [ ] Test: `go run ./cmd/api` → curl localhost:8080/health

### Day 2 — Worker
- [ ] `cmd/worker/main.go`: cron every 60s, pings 3 APIs (httpbin, github, azure status)
- [ ] Store results in Postgres table `checks(id, target, status, latency_ms, checked_at)`
- [ ] Add `internal/config` to load from env

### Day 3 — Frontend
- [ ] `web/`: Vite + React, fetch `/api/status`, show green/red cards
- [ ] Update docker-compose to include api + worker + web
- [ ] Deliverable: `docker-compose up` shows dashboard

---

## Phase 2 — Docker Hardening (Days 4-5)
**Goal:** Production-grade containers

### Day 4
- [ ] Write `Dockerfile.api` and `Dockerfile.worker` (multi-stage, distroless, nonroot)
- [ ] Image size < 20MB
- [ ] Add HEALTHCHECK in Dockerfile
- [ ] Run Trivy locally: `trivy image healthcheck-api`

### Day 5
- [ ] Add structured logging with `log/slog` JSON
- [ ] Add OpenTelemetry SDK, export to stdout initially
- [ ] Add `/metrics` endpoint with otel runtime metrics
- [ ] Deliverable: `docker images` shows hardened images, logs are JSON

---

## Phase 3 — Terraform Foundation (Days 6-8)
**Goal:** Infra as Code for Azure

### Day 6 — Network + ACR
- [ ] `infra/modules/network`: VNet, 2 subnets (private, containerapps)
- [ ] `infra/modules/acr`: Azure Container Registry with admin disabled
- [ ] `terraform apply` in dev

### Day 7 — Data + Secrets
- [ ] `infra/modules/postgres`: PostgreSQL Flexible Server, private endpoint, SSL enforced
- [ ] `infra/modules/keyvault`: Key Vault with RBAC, store DB connection string
- [ ] Enable Managed Identity for future Container Apps

### Day 8 — Container Apps
- [ ] `infra/modules/containerapp`: Container Apps Environment, API app, Worker Job
- [ ] Wire Managed Identity → Key Vault access
- [ ] Output URLs
- [ ] Deliverable: `terraform apply` creates empty infra

---

## Phase 4 — CI/CD (Days 9-11)
**Goal:** Push to main = deploy

### Day 9 — CI pipeline
- [ ] `.github/workflows/ci.yml`: on PR
  - go vet, go test -race, golangci-lint
  - trivy fs
  - terraform fmt, validate, plan
- [ ] Test with dummy PR

### Day 10 — CD pipeline
- [ ] Configure Azure OIDC in GitHub (no secrets)
- [ ] `.github/workflows/cd.yml`: on main
  - azure/login@v2
  - docker buildx build & push to ACR with $GITHUB_SHA
  - terraform apply -auto-approve
  - az containerapp update

### Day 11 — First deploy
- [ ] Merge to main, watch Actions
- [ ] Verify API in Azure, check logs in Log Analytics
- [ ] Deliverable: Live URL works

---

## Phase 5 — Observability (Day 12)
**Goal:** Know when it breaks
- [ ] Instrument Go with otel → Application Insights
- [ ] Create Log Analytics workspace via Terraform
- [ ] Add Azure Monitor alerts:
  - P95 latency > 500ms (5 min)
  - Error rate > 1%
  - Worker job failed
- [ ] Create dashboard in Azure Portal
- [ ] Deliverable: Trigger 500 error, alert fires

---

## Phase 6 — Security Hardening (Day 13)
**Goal:** Production-ready security
- [ ] Enable Microsoft Defender for Cloud on ACR + Container Apps
- [ ] Add Front Door + WAF policy (block SQLi, rate limit)
- [ ] Rotate to Managed Identity for Postgres (no password in Key Vault)
- [ ] Add security headers in Go middleware
- [ ] Run Checkov: `checkov -d infra/`
- [ ] Deliverable: Security score > 80% in Defender

---

## Phase 7 — Chaos & Rollback (Day 14)
**Goal:** Prove reliability
- [ ] Simulate DB failover — does app reconnect?
- [ ] Push bad image (panic on start), verify automatic rollback
- [ ] Test `terraform destroy` and rebuild from scratch in <15 min
- [ ] Document RTO/RPO in README
- [ ] Deliverable: Recorded demo of break/fix

---

## Phase 8 — Stretch Goals (After Day 14)
Pick one based on interest:
- [ ] **Blue-Green:** Two Container App revisions, traffic split via Terraform
- [ ] **Cost:** Auto-scale to zero, schedule worker to stop at night
- [ ] **Testing:** Add integration tests with testcontainers-go
- [ ] **GitOps:** Replace GitHub Actions with FluxCD
- [ ] **Multi-region:** Deploy to East US + Southeast Asia, Front Door failover

---

## Daily Workflow with Meta AI Web
For each task, upload:
1. Current file you're editing
2. `PROJECT.md` for context
3. One example from repo

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

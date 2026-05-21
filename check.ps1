# Full Project Check Script for Windows (PowerShell)
# Runs a suite of verification checks to ensure project health before push or deploy.

param(
    [switch]$Quick,
    [switch]$Fix
)

Write-Host "🚀 Starting Full Project Check..." -ForegroundColor Blue

# --- 0. Generate & Sync OpenAPI Spec ---
Write-Host "`n--- [0/6] Generating OpenAPI Specification (swag) ---" -ForegroundColor Blue
if (-not (Get-Command swag -ErrorAction SilentlyContinue)) {
    Write-Host "swag not found. Installing via go install..." -ForegroundColor Blue
    go install github.com/swaggo/swag/cmd/swag@latest
    if ($LASTEXITCODE -ne 0) { Write-Host "❌ Failed to install swag" -ForegroundColor Red; exit $LASTEXITCODE }
}
swag init -g cmd/api/main.go --output docs --outputTypes json --quiet
if ($LASTEXITCODE -ne 0) { Write-Host "❌ swag generation failed" -ForegroundColor Red; exit $LASTEXITCODE }
Remove-Item docs/swagger.json -ErrorAction SilentlyContinue
Copy-Item docs/openapi.json internal/handler/openapi.json -Force
Write-Host "✓ Generated docs/openapi.json (OpenAPI 3.1) and synced to internal/handler/openapi.json" -ForegroundColor Green

# --- 1. Backend Format Check ---
Write-Host "`n--- [1/6] Checking Backend Code Formatting (gofmt) ---" -ForegroundColor Blue
if ($Fix) {
    Write-Host "Running 'go fmt ./...' to format code..." -ForegroundColor Blue
    go fmt ./...
} else {
    $unformatted = gofmt -l .
    if ($unformatted) {
        Write-Host "❌ Go files are not formatted correctly. Please run './check.ps1 -Fix' or 'go fmt ./...' to fix:" -ForegroundColor Red
        $unformatted | Write-Host -ForegroundColor Red
        exit 1
    }
    Write-Host "✓ Go code is formatted correctly." -ForegroundColor Green
}

# --- 2. Backend Tests ---
Write-Host "`n--- [2/6] Running Backend Tests (Go) ---" -ForegroundColor Blue
go test -v ./...
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Backend tests failed" -ForegroundColor Red; exit $LASTEXITCODE }

# --- 3. Backend Compilation Check ---
Write-Host "`n--- [3/6] Verifying Backend Compilation (go build) ---" -ForegroundColor Blue
go build ./cmd/...
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Compilation failed" -ForegroundColor Red; exit $LASTEXITCODE }

# --- 4. Frontend Lint ---
Write-Host "`n--- [4/6] Running Frontend Lint (ESLint) ---" -ForegroundColor Blue
Set-Location web
if ($Fix) {
    Write-Host "Running eslint --fix..." -ForegroundColor Blue
    pnpm lint --fix
}
pnpm lint
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Lint failed" -ForegroundColor Red; exit $LASTEXITCODE }

# --- 5. Frontend Unit Tests ---
Write-Host "`n--- [5/6] Running Frontend Unit Tests (Vitest) ---" -ForegroundColor Blue
pnpm test:unit
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Unit tests failed" -ForegroundColor Red; exit $LASTEXITCODE }

# --- 6. Frontend Build ---
Write-Host "`n--- [6/6] Running Frontend Build (Vite) ---" -ForegroundColor Blue
if ($Quick) {
    Write-Host "✓ Skipping frontend production build (Quick Mode)." -ForegroundColor Green
} else {
    pnpm build
    if ($LASTEXITCODE -ne 0) { Write-Host "❌ Build failed" -ForegroundColor Red; exit $LASTEXITCODE }
}

# Go back to root directory (just to be clean)
Set-Location ..

Write-Host "`n✅ All checks passed! Project is ready for deployment." -ForegroundColor Green

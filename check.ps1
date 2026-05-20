# Full Project Check Script for Windows (PowerShell)
# Runs a suite of verification checks to ensure project health before push or deploy.

param(
    [switch]$Quick,
    [switch]$Fix
)

Write-Host "🚀 Starting Full Project Check..." -ForegroundColor Blue

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

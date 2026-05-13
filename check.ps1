# Full Project Check Script for Windows (PowerShell)

Write-Host "🚀 Starting Full Project Check..." -ForegroundColor Blue

# 1. Backend Checks
Write-Host "`n--- [1/4] Running Backend Tests (Go) ---" -ForegroundColor Blue
go test -v ./...
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Backend tests failed" -ForegroundColor Red; exit $LASTEXITCODE }

# 2. Frontend Lint
Write-Host "`n--- [2/4] Running Frontend Lint (ESLint) ---" -ForegroundColor Blue
Set-Location web
pnpm lint
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Lint failed" -ForegroundColor Red; exit $LASTEXITCODE }

# 3. Frontend Unit Tests
Write-Host "`n--- [3/4] Running Frontend Unit Tests (Vitest) ---" -ForegroundColor Blue
pnpm test:unit
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Unit tests failed" -ForegroundColor Red; exit $LASTEXITCODE }

# 4. Frontend Build
Write-Host "`n--- [4/4] Running Frontend Build (Vite) ---" -ForegroundColor Blue
pnpm build
if ($LASTEXITCODE -ne 0) { Write-Host "❌ Build failed" -ForegroundColor Red; exit $LASTEXITCODE }

Set-Location ..
Write-Host "`n✅ All checks passed! Project is ready for deployment." -ForegroundColor Green

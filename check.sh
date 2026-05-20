#!/bin/bash

# Full Project Check Script for Linux/macOS (Bash)
# Runs a suite of verification checks to ensure project health before push or deploy.

# Exit immediately if a command exits with a non-zero status.
set -e

# Default settings
QUICK_MODE=false
FIX_MODE=false

# Parse flags
for arg in "$@"; do
    case $arg in
        --quick) QUICK_MODE=true ;;
        --fix) FIX_MODE=true ;;
    esac
done

# Define colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting Full Project Check...${NC}"

# --- 1. Backend Format Check ---
echo -e "\n${BLUE}--- [1/6] Checking Backend Code Formatting (gofmt) ---${NC}"
if [ "$FIX_MODE" = true ]; then
    echo -e "${BLUE}Running 'go fmt ./...' to format code...${NC}"
    go fmt ./...
else
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        echo -e "${RED}❌ Go files are not formatted correctly. Please run './check.sh --fix' or 'go fmt ./...' to fix:${NC}"
        echo "$unformatted"
        exit 1
    fi
    echo -e "${GREEN}✓ Go code is formatted correctly.${NC}"
fi

# --- 2. Backend Tests ---
echo -e "\n${BLUE}--- [2/6] Running Backend Tests (Go) ---${NC}"
go test -v ./...

# --- 3. Backend Compilation Check ---
echo -e "\n${BLUE}--- [3/6] Verifying Backend Compilation (go build) ---${NC}"
go build ./cmd/...

# --- 4. Frontend Lint ---
echo -e "\n${BLUE}--- [4/6] Running Frontend Lint (ESLint) ---${NC}"
cd web
if [ "$FIX_MODE" = true ]; then
    echo -e "${BLUE}Running eslint --fix...${NC}"
    pnpm lint --fix || true
fi
pnpm lint

# --- 5. Frontend Unit Tests ---
echo -e "\n${BLUE}--- [5/6] Running Frontend Unit Tests (Vitest) ---${NC}"
pnpm test:unit

# --- 6. Frontend Build ---
echo -e "\n${BLUE}--- [6/6] Running Frontend Build (Vite) ---${NC}"
if [ "$QUICK_MODE" = true ]; then
    echo -e "${GREEN}✓ Skipping frontend production build (Quick Mode).${NC}"
else
    pnpm build
fi

# Go back to root directory (just to be clean)
cd ..

echo -e "\n${GREEN}✅ All checks passed! Project is ready for deployment.${NC}"

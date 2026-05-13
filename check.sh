#!/bin/bash

# This script runs a full suite of checks to ensure the project is healthy.
# It is designed to be used locally before pushing or in a CI/CD pipeline (AZ-400 style!).

# Exit immediately if a command exits with a non-zero status.
set -e

# Define colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting Full Project Check...${NC}"

# --- 1. BACKEND CHECKS ---
echo -e "\n${BLUE}--- [1/4] Running Backend Tests (Go) ---${NC}"
# Run all go tests. If you have the test DB running, it will run integration tests too.
go test -v ./...

# --- 2. FRONTEND LINT ---
echo -e "\n${BLUE}--- [2/4] Running Frontend Lint (ESLint) ---${NC}"
cd web
pnpm lint

# --- 3. FRONTEND UNIT TESTS ---
echo -e "\n${BLUE}--- [3/4] Running Frontend Unit Tests (Vitest) ---${NC}"
pnpm test:unit

# --- 4. FRONTEND BUILD ---
echo -e "\n${BLUE}--- [4/4] Running Frontend Build (Vite) ---${NC}"
pnpm build

echo -e "\n${GREEN}✅ All checks passed! Project is ready for deployment.${NC}"

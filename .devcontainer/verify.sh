#!/bin/bash
set -e

echo ">>> Verifying devcontainer tools..."

check_cmd() {
  if command -v "$1" &> /dev/null; then
    # Some tools use --version, others version, etc.
    printf "%-12s %s\n" "$1:" "✅ Installed"
  else
    printf "%-12s %s\n" "$1:" "❌ NOT FOUND"
    # We won't exit 1 here so you can at least get into the container
  fi
}

check_cmd go
check_cmd gh
check_cmd node
check_cmd npm

# Removed trivy and checkov unless you add them to devcontainer.json features
# Removed the Database Connectivity check because you removed the DB service (Docker Compose)

echo ""
echo "✅ Verification complete."
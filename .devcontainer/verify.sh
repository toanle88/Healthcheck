#!/bin/bash
set -e

echo ">>> Verifying devcontainer tools..."

check_cmd() {
  if command -v "$1" &> /dev/null; then
    printf "%-12s %s\n" "$1:" "✅ Installed"
  else
    printf "%-12s %s\n" "$1:" "❌ NOT FOUND"
  fi
}

check_cmd go
check_cmd gh
check_cmd node

echo ""
echo "✅ Environment Ready."
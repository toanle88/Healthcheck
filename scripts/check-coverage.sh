#!/bin/bash
set -e

GO_THRESHOLD=${GO_THRESHOLD:-15}
WEB_THRESHOLD=${WEB_THRESHOLD:-40}

echo "=== Running Quality Gate Checks ==="
echo "Go Threshold: ${GO_THRESHOLD}%"
echo "Web Threshold: ${WEB_THRESHOLD}%"

# Check Go coverage
if [ ! -f "coverage.out" ]; then
  echo "❌ Go coverage file (coverage.out) not found!"
  exit 1
fi
GO_COV=$(go tool cover -func=coverage.out | grep total | grep -Eo '[0-9]+\.[0-9]+')
echo "📊 Current Go Statement Coverage: ${GO_COV}%"

GO_PASS=$(awk -v cov="$GO_COV" -v thresh="$GO_THRESHOLD" 'BEGIN {print (cov >= thresh) ? "1" : "0"}')
if [ "$GO_PASS" -eq "0" ]; then
  echo "❌ Go coverage is below the threshold of ${GO_THRESHOLD}%!"
  exit 1
else
  echo "✅ Go coverage meets Quality Gate threshold."
fi

# Check Web coverage
if [ ! -f "web/coverage/coverage-summary.json" ]; then
  echo "❌ Web coverage summary file (web/coverage/coverage-summary.json) not found!"
  exit 1
fi
WEB_COV=$(node -e "console.log(require('./web/coverage/coverage-summary.json').total.statements.pct)")
echo "📊 Current Web Statement Coverage: ${WEB_COV}%"

WEB_PASS=$(awk -v cov="$WEB_COV" -v thresh="$WEB_THRESHOLD" 'BEGIN {print (cov >= thresh) ? "1" : "0"}')
if [ "$WEB_PASS" -eq "0" ]; then
  echo "❌ Web coverage is below the threshold of ${WEB_THRESHOLD}%!"
  exit 1
else
  echo "✅ Web coverage meets Quality Gate threshold."
fi

echo "🎉 All Quality Gate thresholds met successfully!"
exit 0

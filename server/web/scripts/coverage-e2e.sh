#!/bin/bash
# E2E Full Coverage Runner
#
# Runs:
# 1. Go backend with coverage instrumentation (via GOCOVERDIR)
# 2. Playwright UI tests with JS coverage (via coverage.startJSCoverage)
# 3. Combined coverage report
#
# Usage:
#   cd server && bash web/scripts/coverage-e2e.sh
#
# Requirements:
#   Go 1.23+ (for GOCOVERDIR support)
#   Playwright (npm install @playwright/test)

set -e

COVERDIR=$(mktemp -d /tmp/panda-coverage-XXXX)
SERVER_PORT=8080
SERVER_BIN=/tmp/panda_ai_cover

cd "$(dirname "$0")/../.."

# ── Backend: build with coverage ──
echo "=== [1/4] Building Go backend with coverage ==="
go build -cover -o "$SERVER_BIN" ./cmd/panda_ai/ 2>&1
echo "  binary: $SERVER_BIN"

# ── Start server ──
echo "=== [2/4] Starting server on :$SERVER_PORT ==="
GOCOVERDIR="$COVERDIR" "$SERVER_BIN" -c config/config.yaml &
SERVER_PID=$!

for i in $(seq 1 20); do
  if curl -s http://localhost:$SERVER_PORT/health > /dev/null 2>&1; then
    echo "  server ready (PID: $SERVER_PID)"
    break
  fi
  sleep 0.5
done

if ! kill -0 $SERVER_PID 2>/dev/null; then
  echo "  server failed to start"
  exit 1
fi

# ── Start Vite ──
echo "=== [3/4] Starting frontend dev server ==="
npx vite --port 5173 --host &
VITE_PID=$!
sleep 2

# ── Run Playwright tests ──
echo "=== Running Playwright E2E tests ==="
cd web
npx playwright test --reporter=list || echo "  some tests failed (see above)"

# ── Stop servers ──
kill $VITE_PID 2>/dev/null || true
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
wait $VITE_PID 2>/dev/null || true

# ── Backend: convert coverage ──
echo ""
echo "=== [4/4] Generating coverage reports ==="
mkdir -p coverage

if ls "$COVERDIR"/*.cov 1>/dev/null 2>&1; then
  go tool covdata textfmt -i="$COVERDIR" -o=coverage/backend-e2e.out 2>&1
  go tool cover -func=coverage/backend-e2e.out 2>&1 | grep "total:"
  cp coverage/backend-e2e.out coverage/backend-e2e.txt
  echo "  backend coverage: coverage/backend-e2e.out"
else
  echo "  backend coverage: no data (Go 1.23+ required)"
fi

# ── Frontend: generate JS coverage report ──
if ls coverage/frontend/*.json 1>/dev/null 2>&1; then
  node scripts/generate-coverage-report.js
  echo "  frontend coverage: coverage/frontend-report.json"
else
  echo "  frontend coverage: no JS coverage data"
  echo "  (tests use coverage fixture in e2e/fixtures/coverage.ts)"
fi

# ── Cleanup ──
rm -rf "$COVERDIR"
echo ""
echo "=== Coverage summary ==="
if [ -f coverage/backend-e2e.out ]; then
  TOTAL=$(go tool cover -func=coverage/backend-e2e.out 2>&1 | grep "total:" | awk '{print $NF}')
  echo "  Backend (Go):     $TOTAL"
fi
if [ -f coverage/frontend-report.json ]; then
  FRONTEND=$(node -e "console.log(JSON.parse(require('fs').readFileSync('coverage/frontend-report.json','utf-8')).averageFunctionCoverage + '%')" 2>/dev/null)
  echo "  Frontend (JS):    $FRONTEND"
fi
echo "  Done."

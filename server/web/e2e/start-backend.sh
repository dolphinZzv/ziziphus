#!/bin/bash
# Start the Go backend with coverage instrumentation for E2E tests.
# Called by Playwright's webServer config.
set -euo pipefail

# Resolve paths relative to script location
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "→ Building coverage-instrumented binary..."
SERVER_DIR="$ROOT/server"
cd "$SERVER_DIR"
go build -cover -o "$ROOT/bin/ziziphus-coverage" ./cmd/ziziphus/

mkdir -p "$ROOT/coverage/backend"
export GOCOVERDIR="$ROOT/coverage/backend"

echo "→ Starting server (coverage dir: $GOCOVERDIR)..."
cd "$ROOT"
exec bin/ziziphus-coverage -c server/config/config.yaml

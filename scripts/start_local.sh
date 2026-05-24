#!/usr/bin/env bash
set -euo pipefail

# Simple local start script for development (HTTP, no TLS)
# Usage: scripts/start_local.sh

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CFG="$ROOT/config/local.json"

export TEBLORUM_CONFIG="$CFG"
export TEBLORUM_BOOTSTRAP_EMAIL="${TEBLORUM_BOOTSTRAP_EMAIL:-admin@example.com}"
export TEBLORUM_BOOTSTRAP_PASSWORD="${TEBLORUM_BOOTSTRAP_PASSWORD:-changeme}"

mkdir -p "$ROOT/backups" "$ROOT/data" "$ROOT/bin"

echo "Building teblorum..."
cd "$ROOT"
go build -o "$ROOT/bin/teblorum" ./cmd/teblorum

echo "Starting teblorum (HTTP) with config: $CFG"
"$ROOT/bin/teblorum"

#!/usr/bin/env bash
set -euo pipefail

# Start teblorum in background (nohup). Logs are written to ./teblorum.log

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CFG="$ROOT/config/local.json"

export TEBLORUM_CONFIG="$CFG"
export TEBLORUM_BOOTSTRAP_EMAIL="${TEBLORUM_BOOTSTRAP_EMAIL:-admin@example.com}"
export TEBLORUM_BOOTSTRAP_PASSWORD="${TEBLORUM_BOOTSTRAP_PASSWORD:-changeme}"

mkdir -p "$ROOT/backups" "$ROOT/data" "$ROOT/bin"

cd "$ROOT"
go build -o "$ROOT/bin/teblorum" ./cmd/teblorum

LOG="$ROOT/teblorum.log"
echo "Starting teblorum in background; logs -> $LOG"
nohup "$ROOT/bin/teblorum" >> "$LOG" 2>&1 &
echo $! > "$ROOT/teblorum.pid"
echo "PID: $(cat $ROOT/teblorum.pid)"

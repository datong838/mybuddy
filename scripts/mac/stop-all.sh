#!/usr/bin/env bash
# Stop adapter + Dify on macOS
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
if [[ -f /tmp/aos-adapter.pid ]]; then
  kill "$(cat /tmp/aos-adapter.pid)" 2>/dev/null || true
  rm -f /tmp/aos-adapter.pid
fi
pkill -f "uvicorn app.main:app" 2>/dev/null || true
cd "$ROOT/dify/docker"
docker compose down || true
echo "OK: stopped"

#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
bash "$ROOT/scripts/linux/start-dify.sh"
export DIFY_BASE_URL="${DIFY_BASE_URL:-http://127.0.0.1}"
if [[ -z "${DIFY_API_KEY:-}" && -z "${BUDDY_DEMO_MODE:-}" ]]; then
  export BUDDY_DEMO_MODE=1
  echo "BUDDY_DEMO_MODE=1"
fi
cd "$ROOT/adapter"
nohup python3 -m uvicorn app.main:app --host 127.0.0.1 --port 8090 >/tmp/aos-adapter.log 2>&1 &
echo $! >/tmp/aos-adapter.pid
sleep 2
bash "$ROOT/scripts/linux/healthcheck.sh" || true
echo "Stop: bash scripts/linux/stop-all.sh"

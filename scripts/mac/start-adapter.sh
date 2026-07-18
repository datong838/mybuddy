#!/usr/bin/env bash
# macOS adapter
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT/adapter"
export DIFY_BASE_URL="${DIFY_BASE_URL:-http://127.0.0.1}"
python3 -m uvicorn app.main:app --host 127.0.0.1 --port 8090

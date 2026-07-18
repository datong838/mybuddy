#!/usr/bin/env bash
# Linux healthcheck
set -euo pipefail
fail=0
(sudo docker info || docker info) >/dev/null 2>&1 && echo "[OK] docker" || { echo "[MISS] docker"; fail=1; }
code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 http://127.0.0.1/install || echo 000)
[[ "$code" =~ ^(200|301|302|307|308)$ ]] && echo "[OK] dify $code" || { echo "[MISS] dify $code"; fail=1; }
curl -sf --max-time 3 http://127.0.0.1:8090/healthz >/dev/null && echo "[OK] adapter" || echo "[WARN] adapter down"
exit $fail

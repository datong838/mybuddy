#!/usr/bin/env bash
# Fix Dify app storage volume ownership (API runs as uid 1001).
set -euo pipefail
for v in docker_dify_wsl_app dify_wsl_app; do
  if ! docker volume inspect "$v" >/dev/null 2>&1; then
    continue
  fi
  docker run --rm -v "$v":/storage alpine sh -c "chown -R 1001:1001 /storage; chmod -R u+rwX /storage"
  echo "fixed ownership on volume $v"
done

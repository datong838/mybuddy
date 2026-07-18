#!/usr/bin/env bash
# macOS · start Dify (Docker Desktop required)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT/dify/docker"
[ -f .env ] || cp .env.example .env
docker compose up -d
docker compose ps
echo "Open http://127.0.0.1/install"

#!/usr/bin/env bash
# Linux · start Dify
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT/dify/docker"
[ -f .env ] || cp .env.example .env
sudo docker compose up -d
sudo docker compose ps
echo "Open http://127.0.0.1/install"

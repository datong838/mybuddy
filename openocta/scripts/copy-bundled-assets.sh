#!/usr/bin/env bash
# Copy deploy/peekabo and deploy/inner_skills into Wails desktop build output.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_PEEKABO="${ROOT}/deploy/peekabo"
SRC_SKILLS="${ROOT}/deploy/inner_skills"

copy_tree() {
  local dest="$1"
  mkdir -p "${dest}"
  if [[ -d "${SRC_PEEKABO}" ]]; then
    rm -rf "${dest}/peekabo"
    cp -R "${SRC_PEEKABO}" "${dest}/"
  fi
  if [[ -d "${SRC_SKILLS}" ]]; then
    rm -rf "${dest}/inner_skills"
    cp -R "${SRC_SKILLS}" "${dest}/"
  fi
}

APP="${ROOT}/src/build/bin/OpenOcta.app"
if [[ -d "${APP}" ]]; then
  DEST="${APP}/Contents/Resources/bundled"
  echo "==> Copy bundled assets to ${DEST}"
  copy_tree "${DEST}"
fi

BIN_DIR="${ROOT}/src/build/bin"
if [[ -f "${BIN_DIR}/OpenOcta.exe" ]]; then
  DEST="${BIN_DIR}/bundled"
  echo "==> Copy bundled assets to ${DEST}"
  copy_tree "${DEST}"
fi

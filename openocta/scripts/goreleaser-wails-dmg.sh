#!/usr/bin/env bash
# GoReleaser before 钩子：在 macOS 上构建 arm64 + amd64 的 Wails .dmg，供 release.extra_files 上传。
# 在非 macOS（如 Linux CI）上立即退出 0，不构建 DMG。
# 用法（本地验证）：GORELEASER_INCLUDE_DMG=1 goreleaser release --snapshot --clean --skip=publish

set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "goreleaser-wails-dmg: skip (not macOS)"
  exit 0
fi

if [[ "${GORELEASER_INCLUDE_DMG:-}" != "1" ]]; then
  echo "goreleaser-wails-dmg: skip (set GORELEASER_INCLUDE_DMG=1 to build DMGs on macOS)"
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

echo "goreleaser-wails-dmg: building embed + dual-arch DMGs..."
make embed
if [[ "${OPENOCTA_GON:-0}" = "1" ]]; then
  echo "goreleaser-wails-dmg: OPENOCTA_GON=1 -> use gon signing/notarization"
  make wails-dmg-all-signed
else
  make wails-dmg-all
fi

echo "goreleaser-wails-dmg: done:"
ls -la "${ROOT}/dist-mac/"OpenOcta*.dmg 2>/dev/null || true

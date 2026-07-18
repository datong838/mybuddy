#!/usr/bin/env bash
set -euo pipefail

# 使用 Wails 构建 macOS 桌面应用，并打包为 .dmg
# 用法: ./deploy/macos/build-app.sh [--no-dmg]
# --no-dmg: 仅构建 .app，不生成 .dmg
#
# 环境变量:
#   SKIP_MAKE_WAILS=1  — 不执行 make wails（已由 Makefile 按架构执行 wails build 时使用）
#   ARCH=arm64|amd64   — 写入 DMG 文件名后缀 -darwin-<ARCH>（便于区分双架构产物）
#   OPENOCTA_MAC_DIST  — macOS 产物目录，默认 <仓库根>/dist-mac（勿用 dist/：GoReleaser 的 before
#                        钩子跑完后会要求 dist 为空，否则会报错 dist is not empty）
#   OPENOCTA_GON=1     — 使用 gon 对 src/build/bin/OpenOcta.app 做签名+公证+staple（需已安装 gon）
#   GON_CONFIG         — gon 配置文件路径（默认 <仓库根>/gon-sign.json）
#   AC_USERNAME / AC_PASSWORD / AC_TEAM_ID — gon 配置中的 Apple ID 凭据（推荐使用 App-Specific Password）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MAC_DIST="${OPENOCTA_MAC_DIST:-${ROOT}/dist-mac}"
BUILD_DMG=1

[[ "${1:-}" = "--no-dmg" ]] && BUILD_DMG=0

cd "${ROOT}"
if [[ "${SKIP_MAKE_WAILS:-0}" != "1" ]]; then
  echo "==> Wails 构建 macOS 应用（默认当前机架构）..."
  make wails
else
  echo "==> 跳过 make wails（使用已有 OpenOcta.app）"
fi

APP="${ROOT}/src/build/bin/OpenOcta.app"
if [[ ! -d "${APP}" ]]; then
  echo "未找到 OpenOcta.app，构建可能失败" >&2
  exit 1
fi

if [[ "${OPENOCTA_GON:-0}" = "1" ]]; then
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "WARN: OPENOCTA_GON=1 但当前不是 macOS，跳过 gon" >&2
  elif ! command -v gon >/dev/null 2>&1; then
    echo "ERROR: OPENOCTA_GON=1 但未找到 gon。请先安装: brew install mitchellh/gon/gon" >&2
    exit 1
  else
    GON_CFG="${GON_CONFIG:-${ROOT}/gon-sign.json}"
    if [[ ! -f "${GON_CFG}" ]]; then
      echo "ERROR: 未找到 gon 配置: ${GON_CFG}" >&2
      exit 1
    fi
    echo "==> gon: 对 OpenOcta.app 进行签名/公证（配置: ${GON_CFG}）..."
    # gon 会读取配置内的 source；本仓库默认配置指向 ./src/build/bin/OpenOcta.app
    gon "${GON_CFG}"
    echo "==> gon: 完成"
  fi
fi

mkdir -p "${MAC_DIST}"
cp -R "${APP}" "${MAC_DIST}/"
echo "Built: ${MAC_DIST}/OpenOcta.app"

if [[ "${BUILD_DMG}" -eq 1 ]]; then
  VERSION=$(git describe --tags --always 2>/dev/null | sed 's/^v//' || echo "SNAPSHOT")
  ARCH_SUFFIX=""
  if [[ -n "${ARCH:-}" ]]; then
    ARCH_SUFFIX="-darwin-${ARCH}"
  fi
  DMG="${MAC_DIST}/OpenOcta-${VERSION}${ARCH_SUFFIX}.dmg"
  DMG_TMP="${MAC_DIST}/OpenOcta-dmg-tmp"
  rm -rf "${DMG_TMP}" "${DMG}"
  mkdir -p "${DMG_TMP}"
  cp -R "${APP}" "${DMG_TMP}/"
  # 右侧「应用程序」入口：用户可将 OpenOcta 拖入系统应用程序文件夹
  ln -sf /Applications "${DMG_TMP}/Applications"
  cat > "${DMG_TMP}/安装说明.txt" << 'EOF'
OpenOcta 安装说明
================

1. 将左侧「OpenOcta」图标拖入右侧「应用程序」文件夹；或
2. 双击打开 OpenOcta，在弹出对话框中选择「安装」以复制到「应用程序」（需输入本机密码）。

安装完成后可从「启动台」或「应用程序」打开；可将磁盘映像推出。
EOF
  # 卷名避免与「已安装应用」混淆；挂载后为 /Volumes/OpenOcta-Installer（只读映像，不是 /Applications）
  hdiutil create -volname "OpenOcta-Installer" -srcfolder "${DMG_TMP}" -ov -format UDZO "${DMG}"
  rm -rf "${DMG_TMP}"
  echo "Built: ${DMG}"
  echo ""
  echo "用户使用：双击 .dmg → Finder 打开安装窗口 → 拖入「应用程序」或双击应用并按提示安装"
fi

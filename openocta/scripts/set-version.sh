#!/usr/bin/env bash
# 从 git tag 或环境变量设置版本号，更新 src/.env 和 ui/package.json
# 用法: VERSION=v0.1.13 ./scripts/set-version.sh
# 或: ./scripts/set-version.sh  (自动从 git describe 获取)

set -e
cd "$(dirname "$0")/.."

VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  VERSION=$(git describe --tags --always 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
fi
# 确保格式一致（去掉可能的前导 v）
VERSION="${VERSION#v}"

echo "Setting version: $VERSION"

# 更新 src/.env 中的 OPENOCTA_BUNDLED_VERSION
if [[ -f src/.env ]]; then
  if grep -q "^OPENOCTA_BUNDLED_VERSION=" src/.env; then
    if [[ "$(uname)" = "Darwin" ]]; then
      sed -i '' "s|^OPENOCTA_BUNDLED_VERSION=.*|OPENOCTA_BUNDLED_VERSION=$VERSION|" src/.env
    else
      sed -i "s|^OPENOCTA_BUNDLED_VERSION=.*|OPENOCTA_BUNDLED_VERSION=$VERSION|" src/.env
    fi
  else
    echo "OPENOCTA_BUNDLED_VERSION=$VERSION" >> src/.env
  fi
fi

# 更新 ui/package.json 中的 version（用于构建元数据，前端展示仍从后端获取）
if [[ -f ui/package.json ]]; then
  if command -v node &>/dev/null; then
    VERSION="$VERSION" node -e "
      const fs = require('fs');
      const v = (process.env.VERSION || '').replace(/^v/, '');
      const p = 'ui/package.json';
      const j = JSON.parse(fs.readFileSync(p, 'utf8'));
      j.version = v ? 'v' + v : j.version;
      fs.writeFileSync(p, JSON.stringify(j, null, 2) + '\n');
    "
  else
    ver="v${VERSION#v}"
    if [[ "$(uname)" = "Darwin" ]]; then
      sed -i '' "s|\"version\": *\"[^\"]*\"|\"version\": \"$ver\"|" ui/package.json
    else
      sed -i "s|\"version\": *\"[^\"]*\"|\"version\": \"$ver\"|" ui/package.json
    fi
  fi
fi

echo "Done. OPENOCTA_BUNDLED_VERSION=$VERSION, ui/package.json version=v$VERSION"

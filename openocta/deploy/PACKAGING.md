# OpenOcta 打包说明

Wails 桌面版（macOS / Windows）与 GoReleaser（Linux 服务端）的构建入口与产物说明。用户安装指引见 [`deploy/dist-README.md`](dist-README.md)。

---

## 1. 快速开始

| 目标 | 命令 | 产物 |
|------|------|------|
| 当前平台桌面应用 | `make wails` | macOS: `src/build/bin/OpenOcta.app`；Windows: `OpenOcta.exe` |
| macOS 安装包 | `make wails-dmg` | `dist-mac/OpenOcta-<version>.dmg` |
| macOS 签名分发 | `make wails-dmg-signed` | 同上（需 Apple 凭据，见 §4） |
| macOS 双架构 | `make wails-dmg-all` | `dist-mac/OpenOcta-<version>-darwin-{arm64,amd64}.dmg` |
| Windows 安装器 | `./build.sh wails-nsis` | `src/build/bin/OpenOcta-amd64-installer.exe` |
| Linux 服务端 | `./build.sh snapshot` / `./build.sh release` | deb / rpm / tar.gz（见 `.goreleaser.yaml`） |

`./build.sh wails*` 与对应 `make wails*` 等价；`build.sh` 会额外复制产物到 `dist/` 或 `dist-mac/`。

**平台限制**：Wails **不能**在 macOS 上交叉编译 Windows，也不能在 Linux 上打 macOS 包；各平台需在对应环境或 CI 中构建。

**无需手动 `make embed`**：`wails.json` 的 preBuildHooks 会自动执行（前端 → 版本号 → 嵌入资源）。

---

## 2. 环境依赖

| 类别 | 要求 |
|------|------|
| 通用 | Go 1.21+、Node.js 18+、Git |
| Wails | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| macOS | Xcode Command Line Tools（`hdiutil` 系统自带） |
| Windows 安装器 | [NSIS](https://nsis.sourceforge.io/Download)（`makensis.exe` 需在 PATH；Git Bash 下推荐用 `./build.sh wails-nsis`） |
| 版本号 | `scripts/set-version.sh` 从 `git describe --tags` 或 `VERSION` 写入 `src/.env`、`ui/package.json` |

---

## 3. 按平台构建

### macOS

```bash
make wails              # 仅 .app
make wails-dmg          # .app + .dmg
make wails-dmg-signed   # 签名/公证 + .dmg
make wails-dmg-all      # arm64 + amd64 两个 .dmg
```

底层脚本：`./deploy/macos/build-app.sh`（`--no-dmg` 可只打 .app）。DMG 输出目录为 **`dist-mac/`**（与 GoReleaser 的 `dist/` 分离）。

**注意**：勿用 `sudo` 执行 `make wails*`；若 `ui/node_modules` 被 root 占用，用 `sudo chown -R "$(whoami)" .` 修复。

**用户安装**：打开 .dmg → 拖入「应用程序」→ 若 Gatekeeper 拦截：右键「打开」。

### Windows

```bash
./build.sh wails-nsis   # 推荐：NSIS 安装器 + WebView2 检测
make wails              # 仅便携 .exe
```

**CLI + Launcher 模式**（非 Wails 单文件）：先 `make build && make launcher`，再运行 `deploy/windows/build-installer.ps1` → `OpenOcta-Setup.exe`。

### Linux

GoReleaser 构建服务端二进制，**不涉及 Wails**。配置见 `.goreleaser.yaml`。

---

## 4. 签名与发布

### macOS（gon）

配置：`gon-sign.json`（凭据从环境变量读取）。设置 `OPENOCTA_GON=1` 或在 `make wails-dmg-signed` 流程中启用。

```bash
export AC_USERNAME="you@example.com"
export AC_PASSWORD="app-specific-password"
export AC_TEAM_ID="ABCDE12345"
make wails-dmg-signed
```

### GoReleaser 附带 DMG

仅在 **macOS CI** 上设置 `GORELEASER_INCLUDE_DMG=1`；Linux job 勿设（钩子会自动跳过）。

| 变量 | 作用 |
|------|------|
| `GORELEASER_INCLUDE_DMG=1` | before 钩子构建 DMG |
| `OPENOCTA_GON=1` | 使用签名版 `wails-dmg-all-signed` |

```bash
goreleaser release --clean -f .goreleaser.yaml
```

- DMG 通过 `release.extra_files` 上传（`./dist-mac/OpenOcta*.dmg`）
- **勿**叠加多个 `-f` 配置文件（GoReleaser 只认最后一个）
- 推荐：`macos-latest` job 打 DMG，Linux job 打 deb/rpm

---

## 5. 产物目录

| 平台 | 构建输出 | 分发目录 |
|------|----------|----------|
| macOS .app | `src/build/bin/OpenOcta.app` | `dist-mac/` |
| macOS .dmg | — | `dist-mac/OpenOcta-<version>*.dmg` |
| Windows .exe | `src/build/bin/OpenOcta.exe` | `dist/` |
| Windows 安装器 | `src/build/bin/OpenOcta-amd64-installer.exe` | `dist/` |

清理：`make clean` 或 `./build.sh clean`。

---

## 6. build.sh 命令对照

| 命令 | 说明 |
|------|------|
| `./build.sh build` | ui → embed → go，产出 Linux 二进制 `openocta` |
| `./build.sh wails` | Wails 桌面应用（当前平台） |
| `./build.sh wails-dmg` | macOS .dmg |
| `./build.sh wails-dmg-signed` | macOS .dmg（gon） |
| `./build.sh wails-nsis` | Windows NSIS 安装器 |
| `./build.sh snapshot` | GoReleaser 快照 |
| `./build.sh release` | GoReleaser 正式发布 |

---

## 7. 运行配置

- 配置：`~/.openocta/openocta.json`（Windows 见 `paths` 包）
- 默认：`http://127.0.0.1:18900`

---

## 8. 常见问题

**macOS 能编 Windows 安装器吗？**  
不能，需在 Windows 或 Windows CI 上构建。

**embed 相关报错？**  
确认 `src/embed/frontend` 存在；直接 `make wails` 会自动 embed。

**`-nsis` 但只有 .exe？**  
未找到 `makensis`。安装 NSIS 或使用 `./build.sh wails-nsis`。

**修改应用图标？**  
源图 `imgs/openocta_logo.png`；`make prepare-wails-icons` 生成 `src/build/appicon.png` / `.ico`（`wails.json` preBuildHooks 已包含）。Windows 图标缓存异常时可删 `%LocalAppData%\IconCache.db` 后重登。

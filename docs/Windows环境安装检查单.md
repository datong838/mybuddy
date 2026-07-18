# Windows 环境安装检查单（v0.1 AOS 本地工作站）

> **用途**：编码前 / 换机后按本表安装；装完只跑 **一条** PowerShell 复检。  
> **配套脚本**：`mybuddy-v01/scripts/selftest/Check-WindowsDevEnv.ps1`  
> **通过标准**：脚本末尾打印 `RESULT: PASS`，退出码 `0`。  
> **关联**：[`10b 开发计划`](../../docs/palantier/10_v01/10b-v0.1开发计划与自测.md) · Day1 闸门

---

## 0. 一条复检命令（装完必跑）

在 **PowerShell**（勿需管理员，除非脚本提示 UAC）：

```powershell
cd c:\work\projects\wchat\mybuddy-v01
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\selftest\Check-WindowsDevEnv.ps1
```

缺项时由脚本尝试修复（会起 WSL dockerd / 装 rustup 等）：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\selftest\Check-WindowsDevEnv.ps1 -Repair
```

---

## 1. 硬件底线

| # | 项 | 要求 | 自检 |
| --- | --- | --- | --- |
| H1 | RAM | 推荐 ≥16GB（本机已验证约 32GB 亦可） | 任务管理器 |
| H2 | 磁盘 | 系统盘可用 ≥30GB（Dify 镜像约占数 GB～十数 GB） | 资源管理器 |
| H3 | 虚拟化 | BIOS 开启；WSL2 可用 | `wsl -l -v` |

---

## 2. 工具安装清单

| # | 组件 | 安装方式 | 通过判定 |
| --- | --- | --- | --- |
| T1 | **Git** | [git-scm.com](https://git-scm.com/) 或 winget `Git.Git` | `git --version` |
| T2 | **Node.js LTS**（系统安装） | [nodejs.org](https://nodejs.org/) 或 winget `OpenJS.NodeJS.LTS` | `where.exe node` **第一行**为 `C:\Program Files\nodejs\node.exe`（不得仅 Cursor 自带 helper） |
| T3 | **Rust / rustup** | [rustup.rs](https://rustup.rs/) 或本仓 `_installers\rustup-init.exe -y`；网络差可用 USTC：`RUSTUP_DIST_SERVER=https://mirrors.ustc.edu.cn/rust-static` | `rustc -V` · `cargo -V` |
| T4 | **MSVC 生成工具** | VS Build Tools「使用 C++ 的桌面开发」 | Tauri 编译需要（本机已有则跳过） |
| T5 | **WebView2** | 通常随 Edge；否则装 [Evergreen Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) | 存在 EdgeWebView 或 Edge |
| T6 | **Docker（二选一）** | **A.** Docker Desktop（管理员 UAC）<br>**B. 推荐当前路径：** WSL2 Ubuntu + `docker.io` + `docker-compose-v2`；Windows 用 `scripts\win\docker.cmd` | `docker version` 含 **Server**；`docker compose version` |
| T7 | **Python 3**（adapter 单测） | python.org 或 Store | `python --version`（可选但建议） |

### T6 细节（WSL 路径 · 本仓已落地）

1. `wsl -l -v` 有 Ubuntu（VERSION 2）。  
2. WSL 内：`docker.io` · `docker-compose-v2`；`iptables-legacy`；`/etc/docker/daemon.json` 配镜像加速。  
3. Windows 包装：`scripts\win\docker.cmd` · `start-dockerd.ps1` · `start-dify.ps1`。  
4. Dify 必须加：`docker-compose.wsl-volumes.yaml`（避免 `/mnt/c` 上 Postgres chmod 失败）。

Docker Desktop 安装包（可选）：`scripts\_installers\DockerDesktopInstaller.exe`（需管理员 UAC）。

---

## 3. 工程目录闸门（P0）

在 `mybuddy-v01/` 下须存在：

| 路径 | 说明 |
| --- | --- |
| `dify/` · `openocta/` | 上游（已浅克隆） |
| `VERSIONS.md` | SHA 锁 |
| `desktop/` · `openocta-overlay/` · `adapter/` · `docs/` | 自有目录骨架 |
| `scripts\win\` · `scripts\selftest\` | 起服与复检 |

---

## 4. Day1 运行闸门（Dify）

| # | 动作 | 通过 |
| --- | --- | --- |
| D1 | `start-dockerd.ps1` | `docker version` Server 通 |
| D2 | `start-dify.ps1` | compose 服务 Up；`db_postgres` Healthy |
| D3 | 浏览器 / 脚本 | `http://127.0.0.1/install` → **2xx**（或根路径可达） |

> 「空聊成功」需控制台配置模型 Key → 属 10b **A2**，不在本检查单硬性 FAIL。

---

## 5. PATH 约定（避免 Cursor helper 抢先）

用户 PATH 靠前应含：

1. `C:\Program Files\nodejs`  
2. `%USERPROFILE%\.cargo\bin`  
3. `...\mybuddy-v01\scripts\win`（提供 `docker.cmd`）

改完后 **新开** PowerShell 再复检。

---

## 6. 复检结果解读

| 输出 | 含义 | 动作 |
| --- | --- | --- |
| `RESULT: PASS` | 可进入 10b 编码（Tauri / overlay） | — |
| `RESULT: FAIL (N)` | 列在 `[MISS]` | 按表补装后 `-Repair` 再跑 |
| `[WARN]` | 不挡 PASS，建议修 | 如 Desktop 未装但 WSL Docker 可用 |

---

## 7. 手工补救速查

| 症状 | 处理 |
| --- | --- |
| `docker` 找不到 Server | `.\scripts\win\start-dockerd.ps1` |
| Postgres chmod / unhealthy | 确认使用 `docker-compose.wsl-volumes.yaml` |
| Hub 拉取超时 | WSL `/etc/docker/daemon.json` 加 `registry-mirrors`，重启 dockerd |
| `rustc` 无默认工具链 | `$env:RUSTUP_DIST_SERVER=...; rustup default stable` |
| node 指向 Cursor helper | 把 `C:\Program Files\nodejs` 提到 User PATH 最前 |
| `/install` 连不上 | `start-dify.ps1`；确认 `.env` 中 `SECRET_KEY` 非空 |

---

*检查单 v1.0 · 2026-07-16 · 与 Check-WindowsDevEnv.ps1 同步*

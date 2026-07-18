# mybuddy-v01 · AOS 本地工作站

Local-First 桌面试用版（Tauri + adapter + Dify）。方案：`docs/palantier/10_v01/`（10～10d · 11）。

## 一键自测

```powershell
powershell -File .\scripts\selftest\Check-WindowsDevEnv.ps1
powershell -File .\scripts\selftest\run-unit.ps1
```

## 启动 / 关闭（命令行）

完整说明见 [docs/palantier/10_v01/10d](../docs/palantier/10_v01/10d-v0.1-Phase4收口说明.md)。

```powershell
cd c:\work\projects\wchat\mybuddy-v01

# 正式试用：编辑 .env（模板见 .env.example；勿提交 Git）
#   DIFY_API_KEY=app-...
#   BUDDY_DEMO_MODE=0

# 启动（自动加载 .env；无 Key 时 Demo）
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\start-all.ps1

# 桌面
.\dist\windows\AOS-Local-Workstation.exe
# 或：cd desktop; npm run tauri dev

# 关闭（adapter + Dify）
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\win\stop-all.ps1
```

macOS / Linux：`bash ./scripts/mac/start-all.sh` · `stop-all.sh`（linux 同理）。  
交付文档：`docs/客户前置检查单.md` · `试用说明.md` · `故障话术.md` · `三端打包抽测表.md`  
打包：本机 `cd desktop; npm run tauri build`；三端 CI：`.github/workflows/aos-desktop-build.yml`

## 目录

| 目录 | 角色 |
| --- | --- |
| `desktop/` | Tauri 三栏 UI |
| `adapter/` | `/v1/buddy/ask` |
| `openocta-overlay/` | SearchKnowledgeDoc |
| `docs/` | 金牌 / 语料 / 交付 |
| `dify/` · `openocta/` | 上游引擎（不改内核） |

## 开源参考仓克隆

```powershell
# 全量缺口仓（含历史 airbyte 主仓条目；大仓慎用）
.\clone_mybuddy_repos.ps1

# Airbyte 轻量参考（推荐）：Agent SDK / PyAirbyte / CDK-Python …
# 不拉 monorepo、不拉 Java CDK
.\clone_airbyte_refs.ps1              # P0+P1+P2
.\clone_airbyte_refs.ps1 -Tier P0,P1  # MVP

# AOS 缺口依赖 -> refs/（PaddleOCR / MinIO / Grafana / Redis / cosign / MCP…）
# 维护真源：docs/palantier/20_tech/22-AOS开源产品维护清单.md
.\clone_aos_deps.ps1 -Tier P0         # DocIntel + MediaSet 硬缺口
.\clone_aos_deps.ps1 -Tier P0,P1
.\clone_aos_deps.ps1 -Tier All
.\clone_aos_deps.ps1 -Tier P0 -IncludeOptional   # + Tesseract / SeaweedFS
```

> AGPL（MinIO 服务端 / Grafana / ToolJet…）只放 `refs/` 参考，**禁止捆进客户交付包**。  
> 军规：[docs/palantier/20_tech/23-AOS开源引用与交付军规.md](../docs/palantier/20_tech/23-AOS开源引用与交付军规.md)  
> 客户先装 SOP：[docs/palantier/20_tech/24-AOS客户侧前置组件安装SOP.md](../docs/palantier/20_tech/24-AOS客户侧前置组件安装SOP.md) · 交接示例 [`docs/examples/customer-prereq/`](../docs/examples/customer-prereq/)

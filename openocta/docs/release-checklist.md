# OpenOcta 发版 Checklist

每次发版前、发版中、发版后按本清单逐项确认。建议复制到 Issue / MR 描述中，勾选完成项。

**版本规范**：Git tag 格式 `vx.y.z`（如 `v1.0.1`）。版本号写入 `scripts/set-version.sh` 管理的 `src/.env`、`ui/package.json`，并随 GoReleaser / CI 发布。

**发版负责人**：________　**目标版本**：v________　**计划日期**：________

---

## 一、发版前准备

### 1.1 代码与分支

- [ ] 所有待发版 MR 已合并到发版分支（通常为 `main` / `master`）
- [ ] 无已知 P0/P1 阻塞缺陷；遗留问题已在 Issue 中标注「延后至 x.y.z+1」
- [ ] 依赖版本无已知安全漏洞（`go mod`、`ui/package.json`）
- [ ] 配置 schema 变更已同步：`src/config-schema.json` 与文档一致
- [ ] 嵌入资源示例配置已更新：`src/openocta.json.example`

### 1.2 自动化测试

```bash
# 后端单元测试
cd src && go test ./...

# 前端测试
cd ui && npm test
```

- [ ] 后端 `go test ./...` 全部通过
- [ ] 前端 `npm test`（Vitest）全部通过
- [ ] 本地完整构建通过：`make clean && make build`
- [ ] 本地 Gateway 可启动：`./openocta gateway run`，控制台可访问 `http://127.0.0.1:18900`

### 1.3 版本与 Changelog 草稿

- [ ] 确定 semver：PATCH（修复）/ MINOR（功能）/ MAJOR（破坏性变更）
- [ ] `CHANGELOG.md` 已撰写 **未发布** 条目（见第七节）
- [ ] 对比上一版本 tag，确认无遗漏 commit：`git log v<上一版本>..HEAD --oneline`

---

## 二、打包与构建

> 详细说明见 [`deploy/PACKAGING.md`](../deploy/PACKAGING.md)。

### 2.1 本地快照验证（正式发布前）

```bash
make clean
./build.sh snapshot    # GoReleaser 快照，不发布
```

- [ ] 快照构建无报错
- [ ] `dist/` 产物完整：Linux `tar.gz` / `deb` / `rpm`、`openocta-docs_*.tar.gz`
- [ ] 解压 tar.gz 后 `./openocta gateway run` 可正常启动
- [ ] 归档内 `README.md`（来自 `deploy/dist-README.md`）与 `docs/` 目录存在

### 2.2 Linux 服务端（GoReleaser / CI）

- [ ] `.goreleaser.yaml` 与 `deploy/scripts/*.sh` 行尾为 LF（CI 会自动处理）
- [ ] GitLab CI Variables 已配置：`GITLAB_TOKEN`（api scope，Protected + Masked）
- [ ] 推送 tag 后 CI `release` job 成功
- [ ] GitLab Release 页面附件齐全（deb / rpm / tar.gz / docs）
- [ ] Docker 镜像已推送：`$CI_REGISTRY_IMAGE:<version>` 与 `:latest`

### 2.3 macOS 桌面版

```bash
# 不签名（内测）
./build.sh wails-dmg

# 签名 + 公证（对外分发）
./build.sh wails-dmg-signed
# 需 AC_USERNAME / AC_PASSWORD / AC_TEAM_ID
```

- [ ] `OpenOcta.app` 可双击启动，内嵌前端加载正常
- [ ] `.dmg` 可挂载、拖入「应用程序」后正常运行
- [ ] （对外版）签名与公证通过，Gatekeeper 无拦截
- [ ] （CI 集成）`GORELEASER_INCLUDE_DMG=1` 时 `dist-mac/OpenOcta*.dmg` 已附加到 Release

### 2.4 Windows 桌面版

> 须在 **Windows** 环境构建（Wails 无法跨平台产出安装器）。

```powershell
./build.sh wails-nsis
```

- [ ] `OpenOcta.exe` 可启动
- [ ] `OpenOcta-amd64-installer.exe`（NSIS）安装流程正常
- [ ] 未安装 WebView2 时安装器引导正常
- [ ] 安装后开机自启、开始菜单 / 桌面快捷方式正确

### 2.5 产物一致性

- [ ] 各平台「关于 / 版本号」显示为本次 tag 版本
- [ ] 安装包文件名含正确版本号
- [ ] `deploy/dist-README.md` 内容与当前安装方式一致（随 tar.gz 分发）

---

## 三、升级测试

从 **上一正式版** 升级至本次候选包，验证数据与配置兼容。

### 3.1 升级路径

| 平台 | 升级方式 | 确认 |
|------|----------|------|
| Linux deb/rpm | 直接安装新包覆盖 | [ ] |
| Linux tar.gz | 替换二进制，保留 `~/.openocta/` | [ ] |
| macOS .dmg | 覆盖安装到 `/Applications` | [ ] |
| Windows 安装器 | 运行新 Setup 覆盖 | [ ] |
| Docker | 拉取新 tag 镜像，挂载原配置卷 | [ ] |

### 3.2 配置与数据迁移

- [ ] 旧版 `~/.openocta/openocta.json` 可被新版正常读取，无需手工改配置
- [ ] 会话历史（transcript / jsonl）仍可加载
- [ ] 已安装的 Skill / MCP / 数字员工配置保留
- [ ] API Key、渠道凭据、环境变量未丢失
- [ ] `wizard.setup` 状态正确：老用户不重复弹出安装引导（除非刻意重置）
- [ ] systemd 服务升级后自动重启：`systemctl status openocta`

### 3.3 回滚预案

- [ ] 保留上一版本安装包 / 镜像 tag
- [ ] 记录回滚步骤（配置目录无需删除即可降级测试）

---

## 四、重点功能测试

按模块勾选。测试环境需配置有效模型 API Key（如 `ANTHROPIC_API_KEY` 或已接入的国产模型）。

### 4.1 Gateway 与控制台

- [ ] Gateway 默认端口 `18900` 监听正常
- [ ] 控制台页面加载：消息、员工市场、知识库、定时任务、各配置入口
- [ ] 主题切换：浅色 / 深色 / 跟随系统
- [ ] 安装引导 wizard（新装或重置后）：场景初始化流程可走通
- [ ] 概览页：模型 Token 消耗、工具调用次数统计正常

### 4.2 对话与 Agent

- [ ] 新建对话、多轮上下文、历史会话列表
- [ ] 工具调用与推理过程折叠面板（LIVE）展示正常
- [ ] 附件上传：普通文件（≤5MB）、视频（mp4/mov/avi，≤50MB）
- [ ] 交付物路径识别与预览（Markdown / CSV / JSON / 代码等）
- [ ] `@` 本地协同工具委派（Cursor、Codex 等，按本机实际安装情况测）
- [ ] Agent CLI：`./openocta agent -m "echo test"` 有正常回复

### 4.3 开放 API

- [ ] API Key 管理：创建、编辑、启用/停用、删除、重新生成
- [ ] 绑定数字员工 / Skill+MCP+模型 组合
- [ ] `ping` 连通性接口
- [ ] `completion` 对话接口（curl / Python 样例与文档一致）

### 4.4 数字员工、Skill、MCP、工具

- [ ] 员工市场：浏览、安装、启用内置员工
- [ ] 技能库 / 工具库 / 模型库：分类筛选、安装、编辑、删除
- [ ] MCP 服务器连接与工具调用
- [ ] 内置 browser 工具（若本版有变更需回归）
- [ ] 场景模板：`deploy/scenarios/` 至少抽测 1 个（如 `host-inspection`）CLI 或 wizard 初始化

### 4.5 知识库

- [ ] 知识库创建、文档上传、检索
- [ ] Agent 对话中引用知识库内容

### 4.6 定时任务（Cron）

- [ ] 创建任务：名称、描述、员工/Skill/MCP、调度规则、消息、模型
- [ ] 任务按时触发，运行历史可跳转 `cron/xxx` 会话
- [ ] 编辑 / 停用 / 删除任务

### 4.7 IM 渠道（按实际接入渠道勾选）

- [ ] 飞书：收消息 → Agent 回复 → 出站消息（文本 / 卡片 / 媒体）
- [ ] 钉钉：Stream 模式收发
- [ ] 企业微信：Webhook 收发
- [ ] Telegram / QQ / 其他已启用渠道
- [ ] **回归重点**：工具调用后最终结果能回传到 IM（历史已知问题）

### 4.8 Webhooks

- [ ] `/hooks/wake`、`/hooks/agent`、`/hooks/alert` 请求格式与文档一致
- [ ] 鉴权（若启用 token）行为正确

### 4.9 安全与权限

- [ ] Gateway Token 认证开启/关闭行为
- [ ] 工具执行策略 / 命令白名单（见 `docs/security*.md`）
- [ ] 敏感配置不在日志中明文输出

### 4.10 桌面端特有

- [ ] macOS：`.dmg` 安装后首次运行弹窗、菜单栏 / Dock 行为
- [ ] Windows：安装目录、卸载程序、WebView2 依赖
- [ ] 托盘 / launcher（`openocta-launcher`）若随包分发需单独验证
- [ ] 自动更新：每日检查、跳过版本、手动「检查更新」、安装重启（见 [app-update.md](./app-update.md)）

---

## 五、文档

### 5.1 必更新文档（有变更则改）

- [ ] `CHANGELOG.md` — 见第七节
- [ ] `README.md` / `README.en.md` — 新功能、命令、环境要求
- [ ] `deploy/PACKAGING.md` — 构建流程或产物变更
- [ ] `deploy/dist-README.md` — 用户安装说明（随包分发）
- [ ] `docs/configuration.md` — 配置项增删改
- [ ] `docs/environment-variables.md` — 新增环境变量
- [ ] 功能专项文档（按需）：`docs/channels*.md`、`docs/skills.md`、`docs/tools*.md`、`docs/scenarios.md`、`docs/mcp-configuration.md`、`docs/app-update.md`

### 5.2 文档质量

- [ ] 中英文入口互相可发现（README 语言映射）
- [ ] 文档中的版本号、路径、端口号与代码一致
- [ ] 新配置项已写入 `config-schema.json` 并在文档中说明
- [ ] API 样例（curl / Python）可复制运行

### 5.3 发版说明（对外）

- [ ] GitLab / GitHub Release Notes（可与 CHANGELOG 亮点段复用）
- [ ] 讨论群 / 社区公告（重大版本）
- [ ] 升级注意事项单独列出（破坏性变更、需手动迁移项）

---

## 六、Changelog 整理

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，参考现有 `CHANGELOG.md` 结构。

### 6.1 条目结构

```markdown
## [x.y.z] - YYYY-MM-DD

上一版本：[v<上一版>](<release-url>)

### 亮点
（3～5 条用户可感知的能力，面向使用者而非开发者）

### Added
### Changed
### Fixed
### Deprecated / Removed（如有）
### Security（如有）
```

### 6.2 撰写检查

- [ ] 每条说明 **用户能做什么**，避免仅写「重构 xxx 包」
- [ ] 破坏性变更在 `Changed` 或 `Removed` 中 **加粗提醒**
- [ ] 关联 Issue / MR 编号（可选）
- [ ] 与 git log 对照，无遗漏的重要变更
- [ ] 发布时将日期改为实际发版日，并补上 Release 链接

### 6.3 版本亮点（对外一页纸，可选）

- [ ] 从 CHANGELOG「亮点」提炼 5 条以内，用于 Release Notes / 社群通知

---

## 七、正式发布流程

### 7.1 打 tag 并触发 CI

```bash
git checkout main
git pull
# 确认 CHANGELOG、版本相关文件已提交
git tag vx.y.z
git push origin vx.y.z
```

- [ ] tag 指向待发版 commit
- [ ] GitLab CI `release` + `docker` pipeline 绿色
- [ ] GitLab Release 创建成功，附件可下载
- [ ] 容器镜像 `:<version>` 可拉取并启动

### 7.2 桌面版单独发布（若未走 CI DMG）

- [ ] macOS `.dmg` 上传至 Release 或内部分发渠道
- [ ] Windows 安装包上传至 Release 或内部分发渠道
- [ ] 下载链接有效（内网镜像需同步）

### 7.3 发版后验证

- [ ] 从 Release 页面下载 **全新安装** 包，按第四节抽测 smoke test
- [ ] 从 Release 页面下载包做 **第三节升级测试**
- [ ] 生产 / 预发环境已部署新版本
- [ ] 监控与日志无异常尖刺（错误率、启动失败）

### 7.4 收尾

- [ ] 关闭本版本 Milestone / 关联 Issue
- [ ] 创建 `vx.y.z+1` 或下一 MINOR 开发分支规划（如有）
- [ ] 归档本 Checklist（勾选结果附在 Release MR 或 Wiki）

---

## 八、快速命令参考

| 用途 | 命令 |
|------|------|
| 完整构建 | `make build` 或 `./build.sh build` |
| 清理 | `make clean` |
| 快照包 | `./build.sh snapshot` |
| 正式发布 | `./build.sh release`（本地）或 push tag 走 CI |
| Docker 本地 | `./build.sh docker` |
| macOS DMG | `./build.sh wails-dmg` |
| macOS 签名 DMG | `./build.sh wails-dmg-signed` |
| Windows 安装器 | `./build.sh wails-nsis`（Windows） |
| 设置版本 | `VERSION=x.y.z ./scripts/set-version.sh` |
| 后端测试 | `cd src && go test ./...` |
| 前端测试 | `cd ui && npm test` |

---

## 九、发版签字（可选）

| 角色 | 姓名 | 日期 | 签字 |
|------|------|------|------|
| 开发负责人 | | | |
| 测试负责人 | | | |
| 运维 / 发布 | | | |
| 产品 / 文档 | | | |

---

**文档维护**：发版流程或 CI 变更时，同步更新本文件与 `deploy/PACKAGING.md`。

# Changelog

OpenOcta 版本更新记录。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)。

## [1.2.0] - 2026-07-08

上一版本：[v1.0.5](https://github.com/openocta/openocta/releases/tag/v1.0.5)

### 亮点

v1.2.0 在 v1.0.5 基础上聚焦 **本地模型** 与 **发布运维**，核心围绕四件事：

1. **内嵌模型与模型广场** — 基于 yzma + GGUF 的本地 Chat / Embedding 推理，CanIRun 风格目录、下载、启停与 Gateway 代理
2. **手动导入 GGUF** — 用户自行下载权重放入指定目录，刷新扫描后即可识别、启动
3. **应用自动更新** — 桌面端与服务端共用版本检查、跳过记录与一键/手动安装流程
4. **对话与模型路由稳定性** — 内嵌模型 Gateway 代理路由、会话模型解析等修复

### Added

#### 内嵌模型与模型广场

- 新增 **模型广场**（模型库 → 模型广场）：集成 CanIRun.ai 风格目录（约 77 个模型条目），本地 S–F 推荐分级与硬件适配说明
- 支持 **内嵌下载** 内置 GGUF 模型（如 Qwen3-0.6B、Qwen3-Embedding、Qwen2.5-VL-3B、SmolLM2-135M 等），带进度条与取消
- 本地 **Chat / Embedding** 推理：权重存于 `~/.openocta/embedded-models/`，多模型可并行运行，经 Gateway `/api/embedded-models/v1` 统一代理
- 启动后自动合并 `openocta.json` 的 `openocta-embedded-chat` / `openocta-embedded-embedding` provider；Gateway 重启后按 manifest 自动恢复运行中模型
- 模型广场 **测试对话** 弹窗，可验证单模型推理路由
- 新增 **手动导入**：扫描 `embedded-models/<模型ID>/` 下用户放置的 GGUF，刷新后出现在列表（自定义 ID 标记「手动导入」）
- 模型广场 **「手动导入」** 说明弹框，与 [embedded-models-manual-import.md](./docs/embedded-models-manual-import.md) 内容一致
- 详见 [embedded-models.md](./docs/embedded-models.md)、[embedded-models-manual-import.md](./docs/embedded-models-manual-import.md)

#### 应用自动更新

- 桌面（Wails）与服务端（systemd）共用 **版本检查** API 与 UI 弹框
- 支持每日自动检查、手动「检查更新」、跳过指定版本、deb/rpm/dmg/exe 自动安装及手动命令指引
- 详见 [app-update.md](./docs/app-update.md)

#### 对话与控制台

- 对话页支持 **按会话选择模型**（覆盖 Agent 默认配置）
- 新增 **快捷建议提示**（chat suggestions）组件
- 知识库页新增 **图谱视图**（vault graph），可视化笔记关联
- 推理内容支持 `reasoning` / thinking 标签解析与展示优化

#### Agent 运行时

- 内嵌模型专用 `createEmbeddedChatModelFactory`，强制走 Gateway OpenAI 兼容代理

#### 文档与脚本

- 新增/更新：`embedded-models.md`、`embedded-models-manual-import.md`、`app-update.md`、`release-checklist.md`、`desktop-app-design.md` 等
- 新增 `ui/scripts/generate-plaza-catalog.mjs`、`resolve-plaza-gguf-links.mjs` 用于维护广场目录

### Changed

- **模型库**：侧栏增加「模型广场」分类；已安装内嵌模型在本地分区展示
- **内嵌模型 provider**：`baseUrl` 统一指向 Gateway 代理（`http://127.0.0.1:{port}/api/embedded-models/v1`），由请求体 `model` 字段路由到具体实例
- **对话布局**与 **知识库** 样式微调，与模型广场视觉统一
- **sessions** 模型解析：Agent 配置优先于 defaults，支持 `{ primary: "..." }` 对象格式

### Fixed

- **内嵌模型对话**：修复选用 `openocta-embedded-chat/...` 时仍请求 `api.anthropic.com` 的问题；未知 provider 不再静默回退 Anthropic
- **会话模型展示**：修复 `resolveSessionModelRef` 中 defaults 覆盖 Agent 专属模型配置的问题

### 升级建议

1. 首次使用内嵌模型：打开 **模型 → 模型广场**，下载或手动导入 GGUF 后点击 **刷新**，再 **启动** 并在对话中选择 `openocta-embedded-chat/<modelId>`
2. 若曾手动编辑 `openocta.json` 中内嵌 provider 的 `baseUrl` 为旧版直连端口（如 `18902`），请重新启动内嵌模型或打开模型广场以同步为 Gateway 代理地址
3. 自行下载 GGUF 时，目录名即为模型 ID；完整步骤见模型广场 **「手动导入」** 弹框
4. 桌面用户可在顶部栏使用 **检查更新**；Linux 服务端自动安装需配置无密码 `sudo`
5. 从 v1.0.5 升级后若多轮 tool 对话仍异常，请确认 Gateway 版本不低于 v1.0.5（含工具历史水合修复）

### 获取方式

```bash
git clone https://github.com/openocta/openocta.git
cd openocta
git checkout v1.2.0
make build
./openocta gateway run
```

- GitHub Release：[v1.2.0](https://github.com/openocta/openocta/releases/tag/v1.2.0)
- 上一版本说明：见下方 [v1.0.0](#100---2026-06-25)

## [1.0.0] - 2026-06-25

上一版本：[v0.3.0](https://github.com/openocta/openocta/releases/tag/v0.3.0)

### 亮点

v1.0.0 是 OpenOcta 的首个正式大版本，在 v0.3.0 基础上完成控制台改版、开放接入与协同能力升级，核心围绕五件事：

1. **控制台全新视觉** — 更偏技术感的界面风格，支持明暗主题切换
2. **开放 API 接入** — API Key 管理与第三方 HTTP 对话接口，可绑定员工与资源组合
3. **本地智能体协同** — 自动探测 Cursor、Codex 等本机 CLI 工具，对话中 `@` 委派任务
4. **定时任务易用化** — 配置项大幅精简，与数字员工 / Skill / 模型深度绑定
5. **对话与渠道体验修复** — 多模态视频上传、IM 渠道回复、工具结果展示等多项问题修复

### Added

#### 控制台界面

- 整体视觉风格重新设计，布局保持不变，交互与动效更现代、更偏技术风格
- 新增主题切换：浅色 / 深色 / 跟随系统
- 顶部主导航精简为：**消息**、**员工市场**、**知识库**、**定时任务**；原配置区二级菜单平铺为一级入口（技能库、工具库、模型库、通道、安全策略、环境变量、API Key 等）
- 技能库、工具库、模型库的分类筛选移至页面内容区右侧，不再占用左侧功能栏

#### API Key 与开放接入

- 新增 **API Key 管理** 页面：创建、编辑、启用/停用、删除
- 创建时可绑定 **数字员工**，或自定义 **Skill + MCP** 组合，并指定 **模型**
- 支持创建后再次查看密钥、重新生成密钥
- 提供 **请求样例** 弹窗（参数说明、curl / Python 示例）
- 开放 HTTP 接口供第三方调用：`ping` 连通性检测、对话 `completion`（与控制台对话、IM 渠道共用底层流程）

#### 本地智能体协同

- 安装后自动探测本机是否安装：**OpenClaw**、**Hermes Agent**、**Cursor**、**Codex**、**OpenCode**、**Trae**（支持 Windows / macOS / Linux）
- **概览页** 展示各工具探测状态（未检测到为灰色）、探测方式与调用说明
- **对话页** 支持 `@工具名` 委派任务，一句话可拆分给多个协同工具执行
- 输入 `@` 时提供可用工具提示；协同工具入口收纳为下拉菜单，减少输入区占用
- 内置 **local-agents-collab** Skill 手册，引导大模型先理解意图再委派，而非直接执行 shell 命令

#### 对话能力

- 支持上传 **视频文件**（mp4 / mov / avi，单文件上限 50MB），由多模态模型理解
- 普通附件大小上限由原先限制调整为 **5MB**
- 概览页新增 **模型 Token 消耗** 与 **工具调用次数** 统计（轻量汇总，无需开启完整 Trace）
- 对话运行中展示 **推理过程** 折叠面板（LIVE 标签），与工具调用 / 生成回复状态指示联动
- 文件预览按类型渲染：Markdown、CSV 表格、JSON / YAML、代码片段等；对话内联预览与弹窗预览样式统一

#### 定时任务

- 配置页面大幅简化：仅需填写 **名称**、**描述**、**数字员工 / Skill / MCP**、**调度规则**、**消息内容**
- 支持为定时任务指定 **模型**；会话 ID 由系统自动生成，无需手动填写
- 运行历史跳转对话时，统一使用 `cron/xxx` 会话格式

#### 内置资源

- 场景模板目录迁移至 `deploy/scenarios/`
- 新增多份内置 Skill 包（含本地协同、桌面控制、新闻摘要、自进化等）

#### Agent 运行时

- 对话请求可从会话 **transcript（jsonl）** 水合历史消息，多轮上下文更完整；可通过 `session.sessionHistory` 配置启用、角色过滤与条数上限

### Changed

- **对话页**：取消输入框下方的 **浏览器预览** 功能，不再在对话框内嵌展示 Agent 打开的网页；Agent 仍可通过内置 browser 工具操控浏览器，但预览面板与相关接口已移除
- **定时任务**：移除 agentId、负载、投递、超时等高级配置项，降低使用门槛
- **飞书渠道**：升级官方 SDK，适配文本、Markdown、卡片、媒体等多种出站消息与流式回复
- **Agent 对话**：优化上下文与工具加载策略，长对话 Token 消耗更可控；TurnLoop 统一承载多轮对话，transcript 与 ADK checkpoint 协同 Hydrate 模型输入
- **配置页结构**：移除配置 Tab 下的嵌套侧边栏与「关于我们」入口，相关功能并入平铺一级菜单
- **模型库**：支持在线编辑模型 ID 与显示名称；重命名时同步默认模型引用与 `modelEnv` 键；内置厂商在已写入配置后可删除
- **交付物识别**：助手文本中引用的绝对路径、`~` 路径及 A2UI 展示文本内的文件路径均可识别为可预览附件

### Fixed

- **IM 渠道回复**：修复微信、飞书、钉钉等渠道用户消息能收到、但 Agent 最终结果无法回传的问题
- **IM 工具调用场景**：修复对话中使用工具后，最终回复无法投递到 IM 渠道的问题
- **API Key 渠道**：修复通过开放接口调用时，助手回复未正确回传给调用方的问题
- **定时任务**：修复配置了投递渠道但 `deliverTo` 等字段为空导致推送失败的问题；修复运行历史跳转对话链接格式不一致、无法打开会话的问题
- **对话展示**：修复 bash 工具返回的文件预览按钮空白、下载按钮过大，以及图片/文件展示尺寸异常的问题
- **协同工具 UI**：修复 `@` 提示面板遮挡输入内容的问题
- **API Key 页面**：修复「新建 API Key」按钮无响应的问题
- **运行时启动**：修复 browser 工具重复注册导致 Agent 无法创建的问题
- **多轮对话**：修复 transcript 路径解析不一致导致历史未注入模型输入的问题；修复 `tool_use` 停止过早结束事件流、工具链执行中断的问题
- **工具调用参数**：修复部分模型返回空字符串、双重 JSON 编码等异常 `arguments` 导致工具执行失败的问题
- **Skill 加载**：修复 preempt / abort 时 Skill 元数据读取被上下文取消、工具列表构建失败的问题
- **Agent 中断**：修复 panic 或用户中断时已生成内容丢失；`send on closed channel` 不再重复写入 transcript 错误
- **WebSocket**：修复向已关闭 Send 通道写入导致的 panic；响应队列满时优雅降级而非崩溃
- **文件预览**：修复 Data URL 文本附件未按 charset 解码；`write_file` / `edit_file` 输出路径无法生成交付物预览块的问题
- **模型库**：修复删除厂商时未同步清理默认模型与环境变量；安装引导内模型弹层样式层级问题

### Removed

- **教程** 页面及对应前后端代码
- **在线文档** 独立入口
- **会话管理** 独立页面（Sessions）
- **LLM Trace** 调试页面及相关配置（用量改由概览页轻量统计替代）
- **对话框浏览器预览** — 移除对话输入区的预览开关、侧边预览面板及对应后端接口（内置 browser 工具能力保留）
- **配置 Tab 嵌套侧边栏** 与 **关于我们** 侧边入口
- **远程市场** 相关前端模块

### 升级建议

1. 从 v0.3.0 升级后，建议先浏览新版顶部导航，熟悉 API Key、技能库等入口位置变化
2. 若需第三方系统接入，在 **API Key** 页创建密钥并参考页面内请求样例
3. 若使用 Cursor、Codex 等本机工具，可在概览页确认探测结果，并在对话中用 `@` 体验协同委派
4. 原依赖 **LLM Trace** 做调试的用户，可改用概览页 Token / 工具调用统计；深度排查仍可使用日志与 Debug 页
5. 定时任务若曾手动填写会话 ID 或复杂投递配置，升级后请按新表单重新保存任务
6. 若多轮对话上下文异常，检查 `session.sessionHistory` 配置并确认会话 transcript 文件路径可访问

### 获取方式

```bash
git clone https://github.com/openocta/openocta.git
cd openocta
git checkout v1.0.0
make build
./openocta gateway run
```

- GitHub Release：[v1.0.0](https://github.com/openocta/openocta/releases/tag/v1.0.0)
- 上一版本说明：见下方 [v0.3.0](#030---2026-06-22)

## [0.3.0] - 2026-06-22

上一版本：[v0.2.8](https://github.com/openocta/openocta/releases/tag/v0.2.8)

### 亮点

v0.3.0 是一次体验与能力并重的版本更新，核心围绕四件事：

1. **首次安装引导** — 新用户可按步骤完成环境、模型、技能/MCP/员工、场景初始化
2. **内置浏览器** — 无需额外 MCP，Agent 可直接操控 Chromium 完成网页自动化
3. **知识库（Knowledge Vault）** — Obsidian 兼容笔记 + 全文/向量检索，可视化拓扑浏览
4. **对话与交付体验升级** — A2UI 交互、文件/交付物预览、会话运行状态、从对话提取 Skill

### Added

#### 安装引导向导（Setup Wizard）

- 首次启动或新版本首次打开时，弹出多步骤安装引导
- 引导步骤：环境检查 → 模型配置 → 资源安装（技能 / 数字员工 / MCP）→ 场景初始化 → 完成摘要
- 支持在引导中安装内置 Chromium 浏览器
- 内置场景模板机制，当前提供「主机巡检」等场景（`deploy/scenarios/host-inspection`）
- 支持在引导流程中配置 IM 渠道（企业微信、微信等）

#### 内置浏览器

- 新增 `browser` 内置工具，支持 `navigate`、`snapshot`、`screenshot`、`act` 等操作，无需配置浏览器 MCP
- 内置 Chromium 下载与安装管理（引导页 / 控制台均可触发）
- 对话窗口新增浏览器预览面板，可实时查看 Agent 打开的页面
- 新增浏览器相关 HTTP API 与 Gateway 处理器

#### 知识库（Knowledge Vault）

- 全新知识库能力，基于 Obsidian 兼容 Vault + Bleve 全文索引；配置 Embedding API Key 后可启用向量语义检索
- Agent 新增检索工具：`memory_search`（Vault 笔记）、`session_search`（当前会话历史）
- 控制台新增知识库页面，支持笔记浏览与知识拓扑图可视化
- 支持手动「同步索引」，更新笔记后即时生效
- 详见 [knowledge-vault.md](./docs/knowledge-vault.md)

#### 自主进化（Evolution）

- Agent 运行时支持 L4 自主进化：维护 curated MEMORY / USER / SOUL / PROMPT，配合 memory 工具实现长期偏好与行为沉淀

#### Agent 工具与 Skill 能力

- `web_tools` — Web 相关辅助工具集
- `deliverable_files` / `deliverable_read` — 交付物文件生成与读取，对话中可预览 HTML 等交付物
- 从对话提取 Skill — 基于当前会话一键生成 Skill 草稿并下载
- Skill 创意中心 & AI 分析 — 上传 ZIP 后 AI 自动分析元信息；支持多轮对话从零生成 `SKILL.md`
- Skill 组合 API — 新增 `skill_analyze`、`skill_compose` 等 Gateway 接口
- 详见 [skill-create-guide.md](./docs/skill-create-guide.md)

#### A2UI 交互

- 新增 A2UI Bridge / Repair 机制，修复 Markdown 中 A2UI 组件的渲染与文本损坏问题
- 对话中支持 A2UI 交互面板，可响应 Agent 生成的结构化 UI
- 优化文件块（file blocks）展示，支持附件缓存与本地图片预览

#### 配置与文档

- 内置完整配置参考模板 `openocta.json.example` 与 `CONFIG.md` 配置指南（`src/prompt/prompt-zh/`）
- 新增 [knowledge-vault.md](./docs/knowledge-vault.md)、[skill-create-guide.md](./docs/skill-create-guide.md)
- 内置 AMC 企业版对比 Skill，便于在社区版与企业版能力之间做准确引导
- 控制台顶部新增企业版入口链接

### Changed

#### 对话体验

- 重构对话分组渲染，工具调用内联展示，层级更清晰
- 新增会话运行状态指示，长任务执行过程更可感知
- 支持交付物附件、文件预览、聊天资源侧边展示
- 优化对话布局、滚动行为与工具输出面板稳定性（含 Windows 适配）

#### 资源库界面

- 工具库、技能库、员工库统一列表组件，交互与视觉一致
- 概览页 Usage 统计组件重构，信息展示更紧凑

#### Agent 运行时

- 优化 Agent 连接池（pool）与超时策略
- 增强 bash 兼容层与命令策略
- 优化 LLM 调用追踪（llm trace）能力
- 定时任务超时配置增加诊断日志

#### 打包与依赖

- 修正 Goreleaser 文档路径（`src/docs` → `docs`）
- 补全 `agentsdk-go` 依赖条目

### Fixed

- **钉钉渠道**：修复定时任务发送失败（统一 `chatID` 大小写处理）
- **Windows**：修复命令行工具后台执行与窗口闪烁问题
- **A2UI**：修复部分场景下组件渲染异常
- **对话**：修复会话侧边栏与对话内容展示问题
- **文件**：修复交付物 / 附件在对话中的展示问题
- 修复部分运行时 bug 与 UI 细节问题
- 修复 Goreleaser 打包失败问题

### Removed

- **旧版 Memory 模块**：原 SQLite 嵌入式记忆索引（`pkg/memory`）已删除，请迁移至 Knowledge Vault
- **CustomBashTool**：Windows 本地 bash 命令工具已移除，命令执行统一走兼容层与 `WindowsCmdTool`
- **部分场景模板**：内置场景保留 `host-inspection`；`database-ops`、`k8s-incident`、`browser-office` 等模板已移除，可按 [deploy/scenarios/SPEC.md](./deploy/scenarios/SPEC.md) 自行扩展
- **独立 Usage 页面**：已合并至概览页组件

### 升级建议

1. 若曾依赖旧版 Memory，请在 `<workspace>/vault/` 或配置的 Vault 路径中维护 Markdown 笔记，并在知识库页点击「同步索引」
2. 首次打开 v0.3.0 建议完成安装引导，一次性配置模型、浏览器与常用资源
3. 配置文件可参考新版 `openocta.json.example` 与 `CONFIG.md` 补充 `agents.defaults.knowledge` 等字段

### 获取方式

```bash
git clone https://github.com/openocta/openocta.git
cd openocta
git checkout v0.3.0
make build
./openocta gateway run
```

- GitHub Release：[v0.3.0](https://github.com/openocta/openocta/releases/tag/v0.3.0)
- 文档：[README](./README.md) · [知识库说明](./docs/knowledge-vault.md) · [Skill 创建指南](./docs/skill-create-guide.md)

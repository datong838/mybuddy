---
name: local-agents-collab
description: Use when the user @mentions or asks to delegate work to local CLI agents (OpenClaw, Hermes, Cursor, Codex, OpenCode, Trae). Guides task formulation and local_agent tool usage — never invoke CLIs directly from shell.
---

# 本地协同智能体（Local Agents Collaboration）

## 概述

OpenOcta 可探测并委派任务到本机已安装的 CLI 智能体。用户可通过 `@cursor`、`@codex` 等方式提及工具。

**强制工作流：**

1. **理解意图** — 结合对话上下文，把模糊需求扩展为完整、可独立执行的任务描述。
2. **调用 `local_agent` 工具** — 由 OpenOcta 代为执行 CLI，**禁止**在 bash/shell 中直接运行下列命令。
3. **汇总结果** — 将工具输出整理成用户可读的回答。

## 工具 API

```json
// 探测已安装工具
{ "action": "status" }

// 单个委派
{ "action": "run", "agent": "cursor", "task": "完整任务描述…" }

// 并行委派
{ "action": "run_many", "tasks": [
  { "agent": "cursor", "task": "…" },
  { "agent": "codex", "task": "…" }
]}
```

`agent` 可用 id 或别名：`openclaw`/`claw`, `hermes`, `cursor`, `codex`, `opencode`, `trae`/`trae-cli`。

## 任务撰写要点

好的 `task` 应包含：

- **目标**：要完成什么（功能、修复、文档、测试等）
- **上下文**：相关文件路径、技术栈、现有约束
- **验收标准**：怎样算完成（可选但推荐）
- **边界**：不要改什么、用什么模式（只读分析 vs 可改代码）

示例（差 → 好）：

| 用户输入 | 不应直接传递 | 应整理后传递 |
|---------|-------------|-------------|
| `@cursor 登录` | `登录` | `在当前工作区实现用户登录：邮箱+密码表单、JWT session、/api/login 路由；参考 src/auth/` |
| `@codex 写测试` | `写测试` | `为 pkg/localagents/parse.go 的 ParseMessage 函数编写 table-driven 单元测试，覆盖多 @ 提及与空任务场景` |

## 各工具说明

### OpenClaw (`@openclaw` / `@claw`)

- **适用**：OpenClaw 生态内的本地 agent 任务、与 OpenOcta 同体系的工作流。
- **CLI 形态**：`openclaw agent --local --agent main --message "<task>" --json`
- **任务建议**：说明 agent 名称、是否需 JSON 输出、工作区路径。

### Hermes Agent (`@hermes`)

- **适用**：Hermes CLI 支持的自动化与 coding 任务。
- **CLI 形态**：`hermes -z "<task>"`
- **任务建议**：单条清晰指令；复杂任务拆步骤写进 task 正文。

### Cursor (`@cursor`)

- **适用**：代码编辑、重构、项目内多文件改动、IDE 级 coding agent。
- **CLI 形态**：`agent -p --output-format text "<task>"`
- **任务建议**：给出仓库路径、目标文件、框架版本；说明是否允许创建新文件。

### Codex (`@codex`)

- **适用**：OpenAI Codex CLI 的 exec 模式任务（脚本、补丁、终端导向工作）。
- **CLI 形态**：`codex exec --output-format text "<task>"`
- **任务建议**：明确输出格式期望；涉及 secrets 时提醒用户自行配置环境变量。

### OpenCode (`@opencode`)

- **适用**：OpenCode CLI 的 run 流程。
- **CLI 形态**：`opencode run "<task>"`
- **任务建议**：与 Cursor 类似，强调项目根目录与语言/构建工具。

### Trae (`@trae` / `@trae-cli`)

- **适用**：Trae CLI 的 run 任务。
- **CLI 形态**：`trae-cli run "<task>"`（或 `trae run`）
- **任务建议**：保持任务自包含；说明运行环境（Node/Go 等）。

## 多工具协作

用户可能在一句话中 @ 多个工具，例如：

```
@cursor 实现 API  @codex 为 API 写集成测试
```

处理步骤：

1. 分别理解每段意图，各自整理完整 task。
2. 若任务有依赖（先实现再测试），**顺序**调用两次 `local_agent` run。
3. 若完全独立，使用 `run_many` 并行执行。
4. 合并输出，标注各工具负责的部分。

## 错误处理

- `local_agent status` 显示未安装 → 告知用户安装对应 CLI，或建议换用已安装工具。
- 工具返回 Error → 提取关键错误信息，给出下一步（重试、缩小任务范围、检查 PATH）。
- 用户仅 `@cursor` 无任务 → **先追问**，不要猜测执行。

## 反模式（禁止）

- ❌ 在 shell 中直接运行 `cursor`、`codex exec`、`openclaw agent` 等
- ❌ 把用户原始短句不加分析就传给 CLI
- ❌ 跳过 local_agent 工具自行 subprocess
- ✅ 分析 → local_agent → 汇总

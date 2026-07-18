# SearchKnowledgeDoc

> OpenOcta Skill 模板：检索企业知识库（调本机 Dify 或 adapter）。

## 输入

- `question`（必填）
- `object_id`（可选，来自桌面左栏选中）

## 动作

1. `compose_query(question, object_id)`
2. HTTP `POST {DIFY_BASE_URL}/v1/chat-messages`（或 `BUDDY_ASK_URL`）
3. `map_dify_to_contract` → 返回 JSON（含 `trace_id` / `citations` / `status`）

## 环境

见仓库 `config.example.yaml` 与 10a §3。

## 可测模块

`logic.py`（pytest：`openocta-overlay/skills/SearchKnowledgeDoc/tests`）

# OpenOcta 本机起服（Phase 2）

> v0.1 **主问答路径**可用 adapter 直连桌面，不阻塞交付。OpenOcta 为可选 Agent 壳。

## 获取二进制

1. 查看上游 `mybuddy-v01/openocta/README.md` 发布页 / 官方文档下载对应 OS 构建。  
2. 或本机编译（需 Go 环境，耗时长，非 Day1 必选）。

## 加载 overlay

将 `openocta-overlay/` 中 Skill 与配置按官方「外部 skills 目录」方式挂载（见上游 skills 文档）。  
核心可测逻辑在：

`openocta-overlay/skills/SearchKnowledgeDoc/logic.py`

HTTP 封装脚本（不依赖 OpenOcta 进程即可测）：

```powershell
cd mybuddy-v01
$env:BUDDY_DEMO_MODE=1
# 另窗起 adapter
python openocta-overlay/skills/SearchKnowledgeDoc/skill_ask.py "OBJ-1001 的冷运行参数是什么？" --object OBJ-1001
```

## 健康探测

默认 `OPENOCTA_BASE_URL=http://127.0.0.1:18789`（以实际为准）。桌面顶栏 OpenOcta 状态可后续接探活。

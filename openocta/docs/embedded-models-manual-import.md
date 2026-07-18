# 内嵌模型手动导入

本文说明如何**自行下载 GGUF 权重**，放入 OpenOcta 指定目录，在**模型广场**点击「刷新」后识别、启动、停止与删除。

> 适用场景：HuggingFace / ModelScope / 其他渠道下载的 GGUF，或广场目录中暂不支持「内嵌下载」的模型（若该模型 ID 已在广场列表中，也可用手动放置文件代替在线下载）。在 **模型广场** 点击 **「手动导入」** 可查看完整说明。

---

## 一、目录位置

默认状态目录为 `~/.openocta`（Windows：`%APPDATA%\openocta`）。内嵌模型权重统一放在：

```text
~/.openocta/embedded-models/
├── manifest.json                 # 安装与运行状态（自动维护，勿手动改）
├── qwen3-0.6b/                   # 示例：与广场 catalog 相同的 ID
│   └── Qwen3-0.6B-Q4_K_M.gguf
└── my-local-qwen/                # 示例：自定义 ID（不在广场目录中）
    └── My-Local-Qwen-Q4_K_M.gguf
```

可通过环境变量 **`OPENOCTA_STATE_DIR`** 修改状态根目录；内嵌模型子目录始终为 `{OPENOCTA_STATE_DIR}/embedded-models/`。

---

## 二、操作步骤

### 1. 准备模型 ID

**模型 ID = 子目录名**，规则：

- 仅允许字母、数字、`-`、`_`、`.`，且以字母或数字开头
- 示例：`my-qwen-7b`、`bge-small-embed`

两种用法：

| 场景 | 目录名建议 |
|------|------------|
| 模型已在广场列表中（如 `qwen3-0.6b`） | **必须与列表中的 ID 完全一致** |
| 完全自定义、不在广场列表 | 自拟 ID，刷新后会出现在列表**顶部**（标记「手动导入」） |

### 2. 下载并放置 GGUF

1. 从 HuggingFace、ModelScope 等获取 **GGUF** 文件（非 Ollama blob、非 safetensors）。
2. 创建目录：`~/.openocta/embedded-models/<模型ID>/`
3. 将 `.gguf` 文件放入该目录（可直接放根目录，不必嵌套多层）。

**对话模型（Chat）**

- 至少一个**非** `mmproj` 的 `.gguf` 权重文件。
- 若目录内多个权重，优先选用文件名较长的一个作为主权重。

**向量模型（Embedding）**

- 目录名或文件名含 `embed`（不区分大小写）时，会识别为 **Embedding** 类型。
- 示例 ID：`bge-small-embed`、`my-embed-model`。

**多模态（VLM）**

- 除主权重外，将 `mmproj-*.gguf` 放在同一目录。
- 示例（广场模型 `qwen2.5-vl-3b`）：
  ```text
  ~/.openocta/embedded-models/qwen2.5-vl-3b/
  ├── Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf
  └── mmproj-Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf
  ```

### 3. 刷新扫描

1. 打开 **模型 → 模型广场**
2. 点击右上角 **「刷新」**
3. Gateway 会扫描 `embedded-models/` 下含 `.gguf` 的子目录并更新列表

识别成功后：

- 自定义模型：列表顶部显示，带 **「手动导入」** 标记
- 已有 catalog ID：对应行显示 **「已安装」**

### 4. 启动 / 停止 / 删除

与在线下载的模型相同：

1. **启动** — 加载 GGUF，分配本地端口，并写入 `openocta.json` 的 `openocta-embedded-chat` / `openocta-embedded-embedding` provider
2. **停止** — 释放该实例内存
3. **删除** — 停止服务并**删除整个模型目录**（不可恢复）

### 5. 在对话中使用

启动 Chat 模型后，在主聊天或模型配置中选择：

```text
openocta-embedded-chat/<模型ID>
```

例如：`openocta-embedded-chat/my-local-qwen`

---

## 三、平台路径对照

| 系统 | 状态目录 | 内嵌模型目录 |
|------|----------|--------------|
| macOS / Linux | `~/.openocta` | `~/.openocta/embedded-models/` |
| Windows | `%APPDATA%\openocta` | `%APPDATA%\openocta\embedded-models\` |

---

## 四、常见问题

**Q: 放入文件后刷新仍看不到？**

- 确认路径为 `embedded-models/<ID>/`，不是把 gguf 直接放在 `embedded-models/` 根下
- 确认目录内有 `.gguf` 文件
- 确认 `<ID>` 符合命名规则
- 若 ID 与广场 catalog 相同，会合并到对应行，不会 duplicate 一行

**Q: 能否用任意文件名？**

- 可以。扫描时会自动发现目录内 `.gguf`；多文件时选主权重，含 `mmproj` 的视为视觉投影。

**Q: 与 Ollama 下载的模型能否共用？**

- 不能。Ollama 使用 blob 存储，OpenOcta 内嵌推理需要**裸 GGUF 文件**。若已用 Ollama，请在「模型配置」中添加 Ollama Provider，无需手动导入。

**Q: Embedding 模型能否用于主聊天？**

- 不能。Embedding 仅用于向量接口；对话请选择 Chat 类型模型。

**Q: Gateway 重启后会不会丢？**

- 不会。`manifest.json` 会记录已安装与「运行中」状态；手动导入的模型在刷新后也会写入 manifest，支持自动恢复运行。

---

## 五、相关文档

- [内嵌模型总览](./embedded-models.md) — 下载、API、架构
- [模型厂商配置](./model-providers.md) — Ollama / 远程 API

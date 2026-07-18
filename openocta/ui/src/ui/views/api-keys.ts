import { html, nothing } from "lit";
import { icons } from "../icons.js";
import type { SkillStatusEntry } from "../types.ts";
import type { ApiKeyEntry, ApiKeyFormState, ApiKeySecretView } from "../controllers/api-keys.ts";
import { copyApiKeySecret } from "../controllers/api-keys.ts";
import { nativeConfirm } from "../native-dialog-bridge.ts";
import { t } from "../strings.js";

export type ApiKeyModelOption = { value: string; label: string };
export type ApiKeyMcpOption = { key: string; label: string };

export type ApiKeysProps = {
  loading: boolean;
  keys: ApiKeyEntry[];
  error: string | null;
  busy: boolean;
  connected: boolean;
  form: ApiKeyFormState;
  formModalOpen: boolean;
  formModalMode: "create" | "edit";
  viewSecret: ApiKeySecretView | null;
  createdSecret: string | null;
  examplesModalOpen: boolean;
  gatewayBaseUrl: string;
  modelOptions: ApiKeyModelOption[];
  skillOptions: SkillStatusEntry[];
  mcpOptions: ApiKeyMcpOption[];
  digitalEmployees: Array<{ id: string; name?: string; enabled?: boolean }>;
  digitalEmployeesLoading?: boolean;
  onRefresh: () => void;
  onOpenCreateModal: () => void;
  onOpenEditModal: (entry: ApiKeyEntry) => void;
  onCloseFormModal: () => void;
  onFormChange: (patch: Partial<ApiKeyFormState>) => void;
  onRestoreDefaultPaths: () => void;
  onCreate: () => void;
  onUpdate: () => void;
  onViewSecret: (entry: ApiKeyEntry) => void;
  onRegenerateSecret: (entry: ApiKeyEntry) => void;
  onCloseViewSecret: () => void;
  onRemove: (entry: ApiKeyEntry) => void;
  onToggleEnabled: (entry: ApiKeyEntry, enabled: boolean) => void;
  onDismissCreatedSecret: () => void;
  onOpenExamplesModal: () => void;
  onCloseExamplesModal: () => void;
};

function toggleListItem(list: string[], value: string, checked: boolean): string[] {
  const set = new Set(list);
  if (checked) {
    set.add(value);
  } else {
    set.delete(value);
  }
  return [...set];
}

function renderResourceCheckboxes(
  items: Array<{ key: string; label: string; description?: string }>,
  selected: string[],
  onToggle: (key: string, checked: boolean) => void,
  emptyText: string,
) {
  if (!items.length) {
    return html`<div class="cron-resource-box api-keys-modal__resource-box muted">${emptyText}</div>`;
  }
  return html`
    <div class="cron-resource-box api-keys-modal__resource-box">
      ${items.map(
        (item) => html`
          <label class="cron-resource-item">
            <input
              type="checkbox"
              .checked=${selected.includes(item.key)}
              @change=${(e: Event) => onToggle(item.key, (e.target as HTMLInputElement).checked)}
            />
            <span class="cron-resource-item__text">
              <span class="cron-resource-item__name">${item.label}</span>
              ${item.description ? html`<span class="muted cron-resource-item__desc">${item.description}</span>` : nothing}
            </span>
          </label>
        `,
      )}
    </div>
  `;
}

function renderPathsTable(props: ApiKeysProps) {
  const paths = props.form.allowedPaths;
  const updatePath = (index: number, value: string) => {
    const next = [...paths];
    next[index] = value;
    props.onFormChange({ allowedPaths: next });
  };
  const removePath = (index: number) => {
    props.onFormChange({ allowedPaths: paths.filter((_, i) => i !== index) });
  };
  const addPath = () => {
    props.onFormChange({ allowedPaths: [...paths, ""] });
  };

  return html`
    <table class="env-vars__table api-keys__paths-table">
      <thead>
        <tr>
          <th style="width: 48px;">#</th>
          <th>路径前缀（完整 URI 前缀，含 /openocta/open/v1）</th>
          <th style="width: 72px;">操作</th>
        </tr>
      </thead>
      <tbody>
        ${paths.map(
          (path, index) => html`
            <tr>
              <td>${index + 1}</td>
              <td>
                <span class="input"><input
                  type="text"
                  .value=${path}
                  placeholder="/openocta/open/v1/ping"
                  ?disabled=${!props.connected}
                  @input=${(e: Event) => updatePath(index, (e.target as HTMLInputElement).value)}
                /></span>
              </td>
              <td>
                <button
                  class="btn small btn--ghost"
                  type="button"
                  ?disabled=${!props.connected || paths.length <= 1}
                  @click=${() => removePath(index)}
                >
                  删除
                </button>
              </td>
            </tr>
          `,
        )}
      </tbody>
    </table>
    <div class="api-keys__path-actions">
      <button class="btn btn--bg-content" type="button" ?disabled=${!props.connected} @click=${addPath}>
        添加一行
      </button>
      <button
        class="btn btn--bg-content"
        type="button"
        ?disabled=${!props.connected}
        @click=${props.onRestoreDefaultPaths}
      >
        恢复默认
      </button>
    </div>
  `;
}

function renderFormBody(props: ApiKeysProps) {
  const enabledSkills = props.skillOptions.filter((s) => !s.disabled && s.eligible);
  const skillItems = enabledSkills.map((s) => ({
    key: s.skillKey,
    label: s.name || s.skillKey,
    description: s.description,
  }));
  const mcpItems = props.mcpOptions.map((m) => ({ key: m.key, label: m.label }));
  const employeeOptions = props.digitalEmployees
    .filter((e) => e.enabled !== false)
    .slice()
    .sort((a, b) => (a.name ?? a.id ?? "").localeCompare(b.name ?? b.id ?? "", undefined, { sensitivity: "base" }));
  const modelOptions = props.modelOptions.filter((m) => m.value);

  return html`
    ${props.error ? html`<div class="callout danger">${props.error}</div>` : nothing}

    <label class="field">
      <span>名称</span>
      <span class="input"><input
        type="text"
        placeholder="例如 生产网关"
        .value=${props.form.name}
        @input=${(e: Event) => props.onFormChange({ name: (e.target as HTMLInputElement).value })}
      /></span>
    </label>

    <div class="field">
      <span>允许访问的路径</span>
      ${renderPathsTable(props)}
    </div>

    <div class="field">
      <span>绑定方式</span>
      <div class="api-keys__binding-mode">
        <label class="cron-resource-item">
          <input
            type="radio"
            name="api-key-binding"
            .checked=${props.form.bindingMode === "resources"}
            @change=${() => props.onFormChange({ bindingMode: "resources" })}
          />
          <span>按模型 / Skill / MCP 组合</span>
        </label>
        <label class="cron-resource-item">
          <input
            type="radio"
            name="api-key-binding"
            .checked=${props.form.bindingMode === "employee"}
            @change=${() => props.onFormChange({ bindingMode: "employee" })}
          />
          <span>仅绑定数字员工</span>
        </label>
      </div>
    </div>

    <div class="field">
      <span>允许使用的模型</span>
      <div class="muted api-keys-modal__hint">
        不选表示不限制；选择后开放 completion 请求的 model 须匹配白名单（modelKey 或模型名称）。
      </div>
      ${renderResourceCheckboxes(
        modelOptions.map((m) => ({ key: m.value, label: m.label })),
        props.form.allowedModels,
        (key, checked) =>
          props.onFormChange({
            allowedModels: toggleListItem(props.form.allowedModels, key, checked),
          }),
        "暂无可用模型，请先在模型库中配置。",
      )}
    </div>

    ${
      props.form.bindingMode === "resources"
        ? html`
            <div class="api-keys-modal__resources-grid">
              <div class="field">
                <span>技能资源</span>
                <div class="muted api-keys-modal__hint">
                  绑定至该密钥，供后续开放能力与运行时使用（当前示例 completion 仅落库策略）。
                </div>
                ${renderResourceCheckboxes(
                  skillItems,
                  props.form.skillKeys,
                  (key, checked) =>
                    props.onFormChange({
                      skillKeys: toggleListItem(props.form.skillKeys, key, checked),
                    }),
                  "暂无可用技能。",
                )}
              </div>
              <div class="field">
                <span>MCP 工具</span>
                <div class="muted api-keys-modal__hint">绑定 MCP 服务供开放 API 调用时使用。</div>
                ${renderResourceCheckboxes(
                  mcpItems,
                  props.form.mcpServers,
                  (key, checked) =>
                    props.onFormChange({
                      mcpServers: toggleListItem(props.form.mcpServers, key, checked),
                    }),
                  "暂无已启用的 MCP 服务。",
                )}
              </div>
            </div>
          `
        : html`
            <div class="field">
              <span>数字员工</span>
              <span class="select"><select
                .value=${props.form.digitalEmployeeId}
                @change=${(e: Event) =>
                  props.onFormChange({ digitalEmployeeId: (e.target as HTMLSelectElement).value })}
              >
                <option value="">请选择数字员工</option>
                ${employeeOptions.map(
                  (e) => html`
                    <option value=${e.id} ?selected=${props.form.digitalEmployeeId === e.id}>
                      ${e.name ?? e.id}
                    </option>
                  `,
                )}
              </select></span>
              ${
                props.digitalEmployeesLoading
                  ? html`<div class="muted" style="margin-top: 6px;">加载数字员工…</div>`
                  : nothing
              }
            </div>
          `
    }

    <label class="field">
      <span>月度 Token 限额（可选）</span>
      <span class="input"><input
        type="number"
        min="0"
        placeholder="留空表示不限制"
        .value=${props.form.monthlyTokenLimit}
        @input=${(e: Event) =>
          props.onFormChange({ monthlyTokenLimit: (e.target as HTMLInputElement).value })}
      /></span>
    </label>
  `;
}

export function renderApiKeysFormModal(props: ApiKeysProps) {
  if (!props.formModalOpen) {
    return nothing;
  }
  const isEdit = props.formModalMode === "edit";
  const title = isEdit ? "编辑 API Key" : "新建 API Key";
  const subtitle = isEdit
    ? "修改名称、路径、模型与资源绑定；Key 本身可通过「查看 Key」再次查看。"
    : "填写名称、以表格维护允许路径，并配置模型 / 资源绑定与可选的月度 Token 限额。";

  return html`
    <div
      class="modal-overlay modal-overlay--api-keys"
      role="dialog"
      aria-modal="true"
      aria-labelledby="api-keys-modal-title"
      @click=${props.onCloseFormModal}
    >
      <div class="modal card emp-detail-modal api-keys-modal" @click=${(e: Event) => e.stopPropagation()}>
        <div class="emp-detail-modal__header api-keys-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 id="api-keys-modal-title" class="emp-detail-title" style="margin: 0;">${title}</h1>
            <div class="emp-detail-summary api-keys-modal__sub">${subtitle}</div>
          </div>
          <button class="emp-detail-modal__close" type="button" aria-label="关闭" @click=${props.onCloseFormModal}>
            ${icons.x}
          </button>
        </div>
        <div class="api-keys-modal__body">${renderFormBody(props)}</div>
        <div class="modal__actions api-keys-modal__actions">
          <button class="btn" type="button" @click=${props.onCloseFormModal}>取消</button>
          <button
            class="btn primary"
            type="button"
            ?disabled=${props.busy || !props.form.name.trim()}
            @click=${isEdit ? props.onUpdate : props.onCreate}
          >
            ${props.busy ? t("commonSaving") : isEdit ? "保存修改" : "创建"}
          </button>
        </div>
      </div>
    </div>
  `;
}

export function renderApiKeysSecretModal(props: ApiKeysProps) {
  if (!props.viewSecret) {
    return nothing;
  }
  const entry = props.keys.find((k) => k.id === props.viewSecret?.id);
  return html`
    <div
      class="modal-overlay modal-overlay--api-keys"
      role="dialog"
      aria-modal="true"
      aria-labelledby="api-keys-secret-title"
      @click=${props.onCloseViewSecret}
    >
      <div class="modal card emp-detail-modal api-keys-modal api-keys-secret-modal" @click=${(e: Event) => e.stopPropagation()}>
        <div class="emp-detail-modal__header api-keys-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 id="api-keys-secret-title" class="emp-detail-title" style="margin: 0;">查看 API Key</h1>
            <div class="emp-detail-summary api-keys-modal__sub">${props.viewSecret.name}</div>
          </div>
          <button class="emp-detail-modal__close" type="button" aria-label="关闭" @click=${props.onCloseViewSecret}>
            ${icons.x}
          </button>
        </div>
        <div class="api-keys-modal__body">
          <div class="callout success">
            <div><strong>完整 API Key</strong></div>
            <code class="api-keys__secret">${props.viewSecret.secret}</code>
          </div>
          <div class="api-keys__secret-actions">
            <button
              class="btn primary"
              type="button"
              @click=${() => void copyApiKeySecret(props.viewSecret!.secret)}
            >
              复制 Key
            </button>
            ${
              entry
                ? html`
                    <button
                      class="btn"
                      type="button"
                      ?disabled=${props.busy}
                      @click=${async () => {
                        const ok = await nativeConfirm(
                          `重新生成后旧 Key「${entry.keyPrefix}…」将立即失效，确定继续？`,
                        );
                        if (ok) props.onRegenerateSecret(entry);
                      }}
                    >
                      重新生成 Key
                    </button>
                  `
                : nothing
            }
          </div>
        </div>
        <div class="modal__actions api-keys-modal__actions">
          <button class="btn" type="button" @click=${props.onCloseViewSecret}>关闭</button>
        </div>
      </div>
    </div>
  `;
}

/** @deprecated Use renderApiKeysFormModal */
export const renderApiKeysCreateModal = renderApiKeysFormModal;

function renderCodeSample(label: string, code: string) {
  return html`
    <div class="api-keys-examples__sample">
      <div class="api-keys-examples__sample-head">
        <span class="api-keys-examples__sample-label">${label}</span>
        <button class="btn small" type="button" @click=${() => void copyApiKeySecret(code)}>复制</button>
      </div>
      <pre class="api-keys-examples__pre"><code>${code}</code></pre>
    </div>
  `;
}

function exampleModelKey(props: ApiKeysProps): string {
  return props.modelOptions[0]?.value || "your-model-key";
}

function buildPingCurl(base: string, apiKey: string): string {
  return `curl -X POST "${base}/openocta/open/v1/ping" \\
  -H "Authorization: Bearer ${apiKey}" \\
  -H "Content-Type: application/json"`;
}

function buildPingPython(base: string, apiKey: string): string {
  return `import requests

url = "${base}/openocta/open/v1/ping"
headers = {"Authorization": f"Bearer ${apiKey}"}

resp = requests.post(url, headers=headers, timeout=30)
resp.raise_for_status()
print(resp.json())`;
}

function buildCompletionCurl(base: string, apiKey: string, model: string): string {
  return `curl -X POST "${base}/openocta/open/v1/completion" \\
  -H "Authorization: Bearer ${apiKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${model}",
    "messages": [
      {"role": "user", "content": "你好，请简单介绍一下自己。"}
    ],
    "stream": false
  }'`;
}

function buildCompletionPython(base: string, apiKey: string, model: string): string {
  return `import requests

url = "${base}/openocta/open/v1/completion"
headers = {
    "Authorization": f"Bearer ${apiKey}",
    "Content-Type": "application/json",
}
payload = {
    "model": "${model}",
    "messages": [
        {"role": "user", "content": "你好，请简单介绍一下自己。"}
    ],
    "stream": False,
}

resp = requests.post(url, headers=headers, json=payload, timeout=60)
resp.raise_for_status()
print(resp.json())`;
}

export function renderApiKeysExamplesModal(props: ApiKeysProps) {
  if (!props.examplesModalOpen) {
    return nothing;
  }
  const base = props.gatewayBaseUrl.replace(/\/$/, "");
  const apiKey = "YOUR_API_KEY";
  const model = exampleModelKey(props);
  const completionRequest = `{
  "model": "${model}",
  "messages": [
    { "role": "user", "content": "你好，请简单介绍一下自己。" }
  ],
  "conversation_id": "my-thread-001",
  "stream": false
}`;
  const pingResponse = `{
  "ok": true,
  "ts": 1710000000000,
  "keyId": "key_abc123",
  "name": "生产网关",
  "keyPrefix": "oo_xxxx",
  "message": "pong"
}`;
  const completionResponse = `{
  "id": "open-key_abc123",
  "object": "chat.completion",
  "created": 1710000000,
  "model": "${model}",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "..."
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 0
  }
}`;

  return html`
    <div
      class="modal-overlay modal-overlay--api-keys"
      role="dialog"
      aria-modal="true"
      aria-labelledby="api-keys-examples-title"
      @click=${props.onCloseExamplesModal}
    >
      <div
        class="modal card emp-detail-modal api-keys-modal api-keys-examples-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header api-keys-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 id="api-keys-examples-title" class="emp-detail-title" style="margin: 0;">开放 API 请求样例</h1>
            <div class="emp-detail-summary api-keys-modal__sub">
              Gateway 基址：<code>${base}</code>；请将 <code>YOUR_API_KEY</code> 替换为实际密钥。
            </div>
          </div>
          <button class="emp-detail-modal__close" type="button" aria-label="关闭" @click=${props.onCloseExamplesModal}>
            ${icons.x}
          </button>
        </div>
        <div class="api-keys-modal__body api-keys-examples__body">
          <section class="api-keys-examples__section">
            <h2 class="api-keys-examples__title">鉴权</h2>
            <p class="muted api-keys-examples__desc">
              所有开放接口均需携带 API Key，任选其一：
            </p>
            <ul class="api-keys-examples__list">
              <li><code>Authorization: Bearer &lt;api-key&gt;</code></li>
              <li><code>X-OpenOcta-Api-Key: &lt;api-key&gt;</code></li>
            </ul>
            <p class="muted api-keys-examples__desc">
              每个 Key 可配置允许访问的路径前缀；请求路径不在白名单内将返回 <code>403</code>。
            </p>
          </section>

          <section class="api-keys-examples__section">
            <h2 class="api-keys-examples__title">POST /openocta/open/v1/ping</h2>
            <p class="muted api-keys-examples__desc">连通性测试，无需请求体。别名：<code>POST /ping</code>。</p>
            <table class="env-vars__table api-keys-examples__params">
              <thead>
                <tr><th>参数</th><th>位置</th><th>必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr>
                  <td><code>Authorization</code> 或 <code>X-OpenOcta-Api-Key</code></td>
                  <td>Header</td>
                  <td>是</td>
                  <td>API Key</td>
                </tr>
              </tbody>
            </table>
            ${renderCodeSample("响应示例", pingResponse)}
            ${renderCodeSample("cURL", buildPingCurl(base, apiKey))}
            ${renderCodeSample("Python (requests)", buildPingPython(base, apiKey))}
          </section>

          <section class="api-keys-examples__section">
            <h2 class="api-keys-examples__title">POST /openocta/open/v1/completion</h2>
            <p class="muted api-keys-examples__desc">
              Chat 补全接口，内部走与 Web 聊天 / IM 相同的 <code>chat.send</code> 流程。若 Key 配置了模型白名单，<code>model</code> 须匹配允许项；Skill/MCP/数字员工绑定在创建 Key 时配置。
            </p>
            <table class="env-vars__table api-keys-examples__params">
              <thead>
                <tr><th>参数</th><th>位置</th><th>必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr>
                  <td><code>Authorization</code> 或 <code>X-OpenOcta-Api-Key</code></td>
                  <td>Header</td>
                  <td>是</td>
                  <td>API Key</td>
                </tr>
                <tr>
                  <td><code>model</code></td>
                  <td>Body</td>
                  <td>是</td>
                  <td>模型标识（modelKey 或名称）</td>
                </tr>
                <tr>
                  <td><code>messages</code></td>
                  <td>Body</td>
                  <td>否</td>
                  <td>对话消息数组，元素含 <code>role</code>、<code>content</code></td>
                </tr>
                <tr>
                  <td><code>conversation_id</code></td>
                  <td>Body</td>
                  <td>否</td>
                  <td>多轮会话 ID，相同 ID 共享上下文；默认 <code>default</code></td>
                </tr>
                <tr>
                  <td><code>stream</code></td>
                  <td>Body</td>
                  <td>否</td>
                  <td>是否流式，默认 <code>false</code></td>
                </tr>
              </tbody>
            </table>
            ${renderCodeSample("请求体示例", completionRequest)}
            ${renderCodeSample("响应示例", completionResponse)}
            ${renderCodeSample("cURL", buildCompletionCurl(base, apiKey, model))}
            ${renderCodeSample("Python (requests)", buildCompletionPython(base, apiKey, model))}
          </section>
        </div>
        <div class="modal__actions api-keys-modal__actions">
          <button class="btn" type="button" @click=${props.onCloseExamplesModal}>关闭</button>
        </div>
      </div>
    </div>
  `;
}

function bindingSummary(entry: ApiKeyEntry): string {
  if (entry.bindingMode === "employee") {
    return entry.digitalEmployeeId ? `数字员工: ${entry.digitalEmployeeId}` : "数字员工（未指定）";
  }
  const parts: string[] = [];
  if (entry.allowedModels?.length) parts.push(`模型 ${entry.allowedModels.length}`);
  if (entry.skillKeys?.length) parts.push(`技能 ${entry.skillKeys.length}`);
  if (entry.mcpServers?.length) parts.push(`MCP ${entry.mcpServers.length}`);
  return parts.length ? parts.join(" · ") : "模型/Skill/MCP 组合";
}

export function renderApiKeys(props: ApiKeysProps) {
  return html`
    <div class="api-keys">
      <div class="api-keys__toolbar">
        <button class="btn btn--bg-content" ?disabled=${props.loading || !props.connected} @click=${props.onRefresh}>
          ${props.loading ? "…" : t("overviewRefresh")}
        </button>
        <button class="btn primary" ?disabled=${!props.connected} @click=${props.onOpenCreateModal}>新建 API Key</button>
      </div>

      ${
        props.createdSecret
          ? html`
              <div class="callout success api-keys__secret-banner">
                <div><strong>API Key 已创建，可随时在列表中点击「查看 Key」再次查看：</strong></div>
                <code class="api-keys__secret">${props.createdSecret}</code>
                <div class="api-keys__secret-actions">
                  <button class="btn small primary" type="button" @click=${() => void copyApiKeySecret(props.createdSecret!)}>
                    复制 Key
                  </button>
                  <button class="btn small" type="button" @click=${props.onDismissCreatedSecret}>关闭</button>
                </div>
              </div>
            `
          : nothing
      }

      ${props.error && !props.formModalOpen && !props.viewSecret ? html`<div class="callout danger">${props.error}</div>` : nothing}

      <div class="card" style="margin-top: 16px;">
        <p class="muted api-keys__intro" style="font-size: 12px; margin-bottom: 12px;">
          第三方可通过
          <code>Authorization: Bearer &lt;api-key&gt;</code>
          或
          <code>X-OpenOcta-Api-Key</code>
          调用开放接口。连通性测试：
          <code>POST /openocta/open/v1/ping</code>
          或
          <code>POST /ping</code>。
          <button class="api-keys__doc-link" type="button" @click=${props.onOpenExamplesModal}>请求样例</button>
        </p>
        ${
          props.keys.length === 0
            ? html`<p class="muted">${props.loading ? "加载中…" : "暂无 API Key，点击「新建 API Key」创建。"}</p>`
            : html`
                <table class="env-vars__table">
                  <thead>
                    <tr>
                      <th>名称</th>
                      <th>Key 前缀</th>
                      <th>绑定</th>
                      <th>路径数</th>
                      <th>本月用量</th>
                      <th>状态</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    ${props.keys.map(
                      (entry) => html`
                        <tr>
                          <td>${entry.name}</td>
                          <td><code>${entry.keyPrefix}…</code></td>
                          <td>${bindingSummary(entry)}</td>
                          <td>${entry.allowedPaths?.length ?? 0}</td>
                          <td>
                            ${
                              entry.monthlyTokenLimit
                                ? `${entry.monthlyTokensUsed ?? 0} / ${entry.monthlyTokenLimit}`
                                : `${entry.monthlyTokensUsed ?? 0}`
                            }
                          </td>
                          <td>${entry.enabled ? "启用" : "停用"}</td>
                          <td class="api-keys__actions">
                            <button
                              class="btn small"
                              type="button"
                              ?disabled=${props.busy}
                              @click=${() => props.onViewSecret(entry)}
                            >
                              查看 Key
                            </button>
                            <button
                              class="btn small"
                              type="button"
                              ?disabled=${props.busy}
                              @click=${() => props.onOpenEditModal(entry)}
                            >
                              编辑
                            </button>
                            <button
                              class="btn small"
                              type="button"
                              ?disabled=${props.busy}
                              @click=${() => props.onToggleEnabled(entry, !entry.enabled)}
                            >
                              ${entry.enabled ? "停用" : "启用"}
                            </button>
                            <button
                              class="btn small btn--ghost"
                              type="button"
                              ?disabled=${props.busy}
                              @click=${async () => {
                                const ok = await nativeConfirm(`确定删除 API Key「${entry.name}」？`);
                                if (ok) props.onRemove(entry);
                              }}
                            >
                              删除
                            </button>
                          </td>
                        </tr>
                      `,
                    )}
                  </tbody>
                </table>
              `
        }
      </div>
    </div>
  `;
}

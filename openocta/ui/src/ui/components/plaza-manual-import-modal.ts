import { html, nothing } from "lit";
import { icons } from "../icons.js";

export type PlazaManualImportModalProps = {
  open: boolean;
  onClose: () => void;
};

function renderPre(text: string) {
  return html`<pre class="plaza-manual-import__pre">${text}</pre>`;
}

export function renderPlazaManualImportModal(props: PlazaManualImportModalProps) {
  if (!props.open) {
    return nothing;
  }

  return html`
    <div class="modal-overlay" @click=${props.onClose}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large plaza-guide-modal plaza-manual-import-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div>
            <h2>内嵌模型手动导入</h2>
            <p class="plaza-detail-sub">自行下载 GGUF，放入指定目录后点击「刷新」扫描识别</p>
          </div>
          <button class="emp-detail-modal__close" type="button" aria-label="关闭" @click=${props.onClose}>
            ${icons.x}
          </button>
        </div>
        <div class="emp-detail-modal__body plaza-guide-modal__body plaza-manual-import-modal__body">
          <p class="plaza-manual-import__lead muted">
            适用于 HuggingFace、ModelScope 等渠道下载的 GGUF，或广场中暂不支持「内嵌下载」的模型。若模型 ID
            已在广场列表中，也可用手动放置文件代替在线下载。
          </p>

          <section class="plaza-guide-section">
            <h3 class="plaza-guide-section__title">一、目录位置</h3>
            <p>默认状态目录为 <code>~/.openocta</code>（Windows：<code>%APPDATA%\\openocta</code>）。</p>
            ${renderPre(`~/.openocta/embedded-models/
├── manifest.json              # 自动维护，勿手动改
├── qwen3-0.6b/                # 与广场 ID 一致
│   └── Qwen3-0.6B-Q4_K_M.gguf
└── my-local-qwen/             # 自定义 ID
    └── My-Local-Qwen-Q4_K_M.gguf`)}
            <p class="muted">
              可通过环境变量 <code>OPENOCTA_STATE_DIR</code> 修改状态根目录；内嵌模型始终在
              <code>{OPENOCTA_STATE_DIR}/embedded-models/</code> 下。
            </p>
          </section>

          <section class="plaza-guide-section">
            <h3 class="plaza-guide-section__title">二、操作步骤</h3>
            <ol class="plaza-manual-import__steps">
              <li>
                <strong>准备模型 ID</strong>（= 子目录名）：字母/数字开头，可含 <code>-</code>、<code>_</code>、
                <code>.</code>。广场已有模型（如 <code>qwen3-0.6b</code>）须与列表 ID
                完全一致；自定义模型自拟 ID，刷新后出现在列表顶部并标记「手动导入」。
              </li>
              <li>
                <strong>下载并放置 GGUF</strong>：创建
                <code>~/.openocta/embedded-models/&lt;模型ID&gt;/</code>，放入
                <code>.gguf</code> 文件（非 Ollama blob、非 safetensors）。
                <ul>
                  <li><strong>Chat</strong>：至少一个非 <code>mmproj</code> 的权重文件。</li>
                  <li><strong>Embedding</strong>：目录名或文件名含 <code>embed</code> 时识别为向量模型。</li>
                  <li><strong>VLM</strong>：同一目录再放 <code>mmproj-*.gguf</code>。</li>
                </ul>
              </li>
              <li><strong>刷新扫描</strong>：本页点击「刷新」，Gateway 扫描含 GGUF 的子目录。</li>
              <li><strong>启动 / 停止 / 删除</strong>：与在线下载模型相同；删除会移除整个模型目录。</li>
              <li>
                <strong>在对话中使用</strong>：启动 Chat 模型后选择
                <code>openocta-embedded-chat/&lt;模型ID&gt;</code>，例如
                <code>openocta-embedded-chat/my-local-qwen</code>。
              </li>
            </ol>
          </section>

          <section class="plaza-guide-section">
            <h3 class="plaza-guide-section__title">三、平台路径</h3>
            <table class="plaza-manual-import__table">
              <thead>
                <tr>
                  <th>系统</th>
                  <th>内嵌模型目录</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td>macOS / Linux</td>
                  <td><code>~/.openocta/embedded-models/</code></td>
                </tr>
                <tr>
                  <td>Windows</td>
                  <td><code>%APPDATA%\\openocta\\embedded-models\\</code></td>
                </tr>
              </tbody>
            </table>
          </section>

          <section class="plaza-guide-section">
            <h3 class="plaza-guide-section__title">四、常见问题</h3>
            <dl class="plaza-manual-import__faq">
              <dt>放入文件后刷新仍看不到？</dt>
              <dd>
                确认路径为 <code>embedded-models/&lt;ID&gt;/</code>（不要放在根目录）；目录内有
                <code>.gguf</code>；ID 符合命名规则。
              </dd>
              <dt>能否用任意文件名？</dt>
              <dd>可以。扫描会自动发现 <code>.gguf</code>；含 <code>mmproj</code> 的视为视觉投影。</dd>
              <dt>与 Ollama 模型能否共用？</dt>
              <dd>
                不能。内嵌推理需要裸 GGUF。若已用 Ollama，请在「模型配置」中添加 Ollama Provider。
              </dd>
              <dt>Embedding 能否用于主聊天？</dt>
              <dd>不能。Embedding 仅用于向量；对话请选 Chat 类型。</dd>
              <dt>Gateway 重启后会丢失吗？</dt>
              <dd>不会。刷新后会写入 manifest，支持自动恢复运行中的模型。</dd>
            </dl>
          </section>
        </div>
      </div>
    </div>
  `;
}

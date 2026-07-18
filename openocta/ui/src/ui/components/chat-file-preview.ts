import { customElement, property, state } from "lit/decorators.js";
import { LitElement, css, html, nothing } from "lit";
import { icons } from "../icons.ts";
import { decodeFileText, type FilePreviewRequest } from "../chat/file-blocks.ts";
import { renderTextFilePreviewBody } from "../chat/file-preview-content.ts";

@customElement("chat-file-preview")
export class ChatFilePreview extends LitElement {
  @property({ attribute: false }) request: FilePreviewRequest | null = null;
  @property({ attribute: false }) onClose?: () => void;

  @state() private error: string | null = null;

  static styles = css`
    :host {
      position: fixed;
      inset: 0;
      z-index: 1200;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }
    .backdrop {
      position: absolute;
      inset: 0;
      background: rgba(0, 0, 0, 0.45);
      border: none;
      padding: 0;
      cursor: default;
    }
    .panel {
      position: relative;
      z-index: 1;
      width: min(920px, 100%);
      max-height: min(85vh, 900px);
      background: var(--bg-content, #fff);
      border-radius: 14px;
      border: 1px solid var(--border, rgba(127, 127, 127, 0.25));
      display: flex;
      flex-direction: column;
      overflow: hidden;
      box-shadow: 0 16px 48px rgba(0, 0, 0, 0.18);
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 12px 16px;
      border-bottom: 1px solid var(--border, rgba(127, 127, 127, 0.2));
    }
    .title {
      font-weight: 600;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .actions {
      display: flex;
      gap: 8px;
      align-items: center;
      flex-shrink: 0;
    }
    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 4px;
      height: 28px;
      padding: 0 10px;
      border: 1px solid var(--border, rgba(127, 127, 127, 0.35));
      border-radius: 6px;
      background: var(--bg, #f5f5f5);
      color: var(--text-primary, #111);
      font-size: 12px;
      line-height: 1;
      cursor: pointer;
      text-decoration: none;
      white-space: nowrap;
    }
    .btn:hover {
      border-color: var(--border-primary, rgba(127, 127, 127, 0.55));
    }
    .btn--icon {
      width: 28px;
      min-width: 28px;
      padding: 0;
    }
    .btn svg {
      width: 14px;
      height: 14px;
      stroke: currentColor;
      fill: none;
      stroke-width: 1.75px;
    }
    .body {
      overflow: auto;
      padding: 16px;
      flex: 1;
    }
    .image-preview {
      display: flex;
      justify-content: center;
      align-items: center;
    }
    .image-preview img {
      max-width: 100%;
      max-height: min(70vh, 720px);
      object-fit: contain;
      border-radius: 8px;
    }
    pre {
      margin: 0;
      white-space: pre-wrap;
      word-break: break-word;
      font-family: var(--mono, ui-monospace, monospace);
      font-size: 13px;
      line-height: 1.5;
    }
    iframe {
      width: 100%;
      min-height: 70vh;
      border: 0;
      border-radius: 8px;
      background: #fff;
    }
    .muted {
      color: var(--text-secondary, #666);
      font-size: 13px;
    }
    .callout.danger {
      color: var(--danger, #c62828);
      font-size: 13px;
    }
    .file-preview-table-wrap {
      overflow: auto;
      max-height: min(70vh, 720px);
      border: 1px solid var(--border, rgba(127, 127, 127, 0.2));
      border-radius: 8px;
    }
    .file-preview-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 13px;
      line-height: 1.4;
    }
    .file-preview-table th,
    .file-preview-table td {
      padding: 8px 10px;
      border-bottom: 1px solid var(--border, rgba(127, 127, 127, 0.18));
      text-align: left;
      vertical-align: top;
      word-break: break-word;
    }
    .file-preview-table th {
      position: sticky;
      top: 0;
      background: var(--bg, #f5f5f5);
      font-weight: 600;
      white-space: nowrap;
    }
    .file-preview-table tbody tr:hover {
      background: color-mix(in srgb, var(--accent, #4f8cff) 6%, transparent);
    }
    .file-preview-code,
    .file-preview-plain {
      margin: 0;
      padding: 12px 14px;
      border-radius: 8px;
      background: var(--bg, #f8f8f8);
      border: 1px solid var(--border, rgba(127, 127, 127, 0.18));
      white-space: pre-wrap;
      word-break: break-word;
      font-family: var(--mono, ui-monospace, monospace);
      font-size: 13px;
      line-height: 1.5;
      overflow: auto;
      max-height: min(70vh, 720px);
    }
    .file-preview-markdown {
      font-size: 14px;
      line-height: 1.6;
      overflow: auto;
      max-height: min(70vh, 720px);
    }
    .file-preview-table-more {
      padding: 6px 10px;
      font-size: 12px;
      border-top: 1px solid var(--border, rgba(127, 127, 127, 0.18));
    }
  `;

  protected updated(changed: Map<string, unknown>): void {
    if (changed.has("request")) {
      this.error = null;
    }
  }

  private close = () => {
    this.onClose?.();
  };

  private isImageRequest(req: FilePreviewRequest): boolean {
    const ext = req.filename.split(".").pop()?.toLowerCase() ?? "";
    if (["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg"].includes(ext)) {
      return true;
    }
    return req.mimeType.toLowerCase().startsWith("image/");
  }

  private renderBody() {
    const req = this.request;
    if (!req) {
      return nothing;
    }
    const ext = req.filename.split(".").pop()?.toLowerCase() ?? "";
    if (this.isImageRequest(req)) {
      return html`
        <div class="image-preview">
          <img src=${req.url} alt=${req.filename} />
        </div>
      `;
    }
    const isPdf = req.mimeType === "application/pdf" || ext === "pdf";
    const isHtml =
      ext === "html" ||
      ext === "htm" ||
      req.mimeType.toLowerCase().includes("html");
    if (isPdf || isHtml) {
      return html`<iframe title=${req.filename} src=${req.url}></iframe>`;
    }
    const text = decodeFileText(req.url);
    if (text == null && !req.url.startsWith("data:")) {
      return html`
        <div class="muted">无法预览远程文件，请使用下载。</div>
        <a class="btn" href=${req.url} download=${req.filename} target="_blank" rel="noopener">下载</a>
      `;
    }
    if (text == null) {
      return html`<div class="callout danger">无法解码文件内容</div>`;
    }
    return renderTextFilePreviewBody(text, req.filename, req.mimeType, "modal");
  }

  render() {
    if (!this.request) {
      return nothing;
    }
    return html`
      <button type="button" class="backdrop" aria-label="关闭预览" @click=${this.close}></button>
      <div class="panel" role="dialog" aria-modal="true" @click=${(e: Event) => e.stopPropagation()}>
        <div class="header">
          <div class="title">${this.request.filename}</div>
          <div class="actions">
            <a class="btn" href=${this.request.url} download=${this.request.filename} target="_blank" rel="noopener">
              下载
            </a>
            <button type="button" class="btn btn--icon" @click=${this.close} aria-label="关闭">
              ${icons.x}
            </button>
          </div>
        </div>
        <div class="body">
          ${this.error ? html`<div class="callout danger">${this.error}</div>` : this.renderBody()}
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "chat-file-preview": ChatFilePreview;
  }
}

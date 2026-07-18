import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import type { AppUpdateCheckResult, AppUpdateProgress } from "../controllers/app-update.ts";

export type AppUpdateModalProps = {
  open: boolean;
  info: AppUpdateCheckResult | null;
  manualHint: string | null;
  checking: boolean;
  installing: boolean;
  error: string | null;
  progress: AppUpdateProgress | null;
  onClose: () => void;
  onSkip: () => void | Promise<void>;
  onInstall: () => void | Promise<void>;
  onCopyManualHint: () => void | Promise<void>;
  onOpenDownload: () => void | Promise<void>;
};

export function renderAppUpdateModal(props: AppUpdateModalProps) {
  if (!props.open) {
    return nothing;
  }
  const current = props.info?.currentVersion ?? "---";
  const latest = props.info?.latestVersion ?? "---";
  const progressMsg = props.progress?.message ?? (props.installing ? "正在更新…" : "");
  const progressPct = props.progress?.percent;
  const manualHint = props.manualHint ?? props.info?.manualInstallHint ?? "";
  const canAutoInstall = Boolean(props.info?.autoInstallSupported);
  const canDownload = Boolean(props.info?.downloadSupported && props.info?.downloadUrl);

  return html`
    <div
      class="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="app-update-title"
      @click=${(e: Event) => {
        if ((e.target as HTMLElement).classList.contains("modal-overlay")) {
          props.onClose();
        }
      }}
    >
      <div class="modal card about-uninstall-modal" @click=${(e: Event) => e.stopPropagation()}>
        <h3 id="app-update-title" class="modal__title">发现新版本</h3>
        <p class="muted">
          当前版本 <strong>${current}</strong>，最新版本 <strong>${latest}</strong>。
          ${canAutoInstall ? "是否立即更新？" : "请使用自动安装或下方手动命令完成升级。"}
        </p>
        ${!props.info?.downloadSupported
          ? html`<p class="about-uninstall-api-error" role="alert">当前系统暂无可用安装包，请前往官网手动下载。</p>`
          : nothing}
        ${props.error
          ? html`<p class="about-uninstall-api-error" role="alert" style="white-space: pre-wrap;">${props.error}</p>`
          : nothing}
        ${progressMsg
          ? html`
              <p class="muted">${progressMsg}${typeof progressPct === "number" ? `（${progressPct}%）` : ""}</p>
              ${typeof progressPct === "number"
                ? html`<div
                    style="height: 6px; border-radius: 999px; background: var(--border, #e5e7eb); overflow: hidden; margin: 8px 0 12px;"
                  >
                    <div
                      style="height: 100%; width: ${Math.max(0, Math.min(100, progressPct))}%; background: var(--accent, #2563eb); transition: width 0.2s ease;"
                    ></div>
                  </div>`
                : nothing}
            `
          : nothing}
        ${manualHint
          ? html`
              <div class="field" style="margin: 12px 0;">
                <div class="card-sub" style="margin-bottom: 6px;">手动安装命令（自动安装失败时可使用）</div>
                <pre
                  style="margin: 0; padding: 10px 12px; border-radius: 8px; background: var(--bg-muted, #f3f4f6); overflow: auto; max-height: 220px; font-size: 12px; line-height: 1.5; white-space: pre-wrap;"
                >${manualHint}</pre>
              </div>
            `
          : nothing}
        <div class="modal__actions">
          <button type="button" class="btn" ?disabled=${props.installing} @click=${props.onClose}>稍后</button>
          <button
            type="button"
            class="btn btn--secondary"
            ?disabled=${props.installing || props.checking}
            @click=${props.onSkip}
          >
            跳过此版本
          </button>
          ${manualHint
            ? html`
                <button
                  type="button"
                  class="btn btn--secondary"
                  ?disabled=${props.installing}
                  @click=${props.onCopyManualHint}
                >
                  复制命令
                </button>
              `
            : nothing}
          ${canDownload && !canAutoInstall
            ? html`
                <button
                  type="button"
                  class="btn btn--secondary"
                  ?disabled=${props.installing}
                  @click=${props.onOpenDownload}
                >
                  打开下载
                </button>
              `
            : nothing}
          ${canAutoInstall
            ? html`
                <button
                  type="button"
                  class="btn btn--primary"
                  ?disabled=${props.installing || props.checking}
                  @click=${props.onInstall}
                >
                  <span class="btn__icon" aria-hidden="true">${icons.download}</span>
                  ${props.installing ? "正在更新…" : "立即更新"}
                </button>
              `
            : nothing}
        </div>
      </div>
    </div>
  `;
}

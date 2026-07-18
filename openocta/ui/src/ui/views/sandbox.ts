import { html, nothing } from "lit";
import { nativeConfirm } from "../native-dialog-bridge.ts";
import { t } from "../strings.js";
import type { SandboxConfigForm } from "../controllers/sandbox.ts";
import type { ApprovalEntry, ApprovalsListResult } from "../controllers/approvals.ts";
import type { Tab } from "../navigation.ts";

export type SandboxProps = {
  sandbox: SandboxConfigForm | null;
  saving: boolean;
  onToggleEnabled: () => void;
  onToggleValidatorEnabled?: () => void;
  onToggleApprovalEnabled?: () => void;
  onPatch: (path: string[], value: unknown) => void;
  onSave: () => void;
  // Approval queue (embedded in sandbox page)
  approvalsLoading: boolean;
  approvalsResult: ApprovalsListResult | null;
  approvalsError: string | null;
  onApprovalsRefresh: () => void;
  onApprove: (requestId: string) => void;
  onDeny: (requestId: string, reason?: string) => void;
  onWhitelistSession: (requestId: string, sessionId: string) => void;
  pathForTab: (tab: Tab) => string;
};

function ensureArray(arr: string[] | undefined): string[] {
  return Array.isArray(arr) ? arr : [];
}

function joinLines(arr: string[]): string {
  return ensureArray(arr).filter(Boolean).join("\n");
}

function splitLines(s: string): string[] {
  return (s || "")
    .split("\n")
    .map((x) => x.trim())
    .filter(Boolean);
}

function formatApprovalTime(ms: number | undefined): string {
  if (ms == null) return "—";
  const d = new Date(ms);
  return d.toLocaleString();
}

export function renderSandbox(props: SandboxProps) {
  const s = props.sandbox ?? {};
  // 仅当 explicitly true 时显示为启用，false 或 undefined 均显示为禁用
  const enabled = s.enabled === true;
  const allowedPaths = joinLines(ensureArray(s.allowedPaths));
  const networkAllow = joinLines(ensureArray(s.networkAllow));
  const validator = s.validator ?? {};
  const validatorEnabled = (validator as { enabled?: boolean }).enabled === true;
  const approvalQueue = (s as { approvalQueue?: { enabled?: boolean; timeoutSeconds?: number; blockOnApproval?: boolean; allow?: string[]; ask?: string[]; deny?: string[] } }).approvalQueue ?? {};
  const banCommands = joinLines(ensureArray(validator.banCommands));
  const banArguments = joinLines(ensureArray(validator.banArguments));
  const banFragments = joinLines(ensureArray(validator.banFragments));
  const resourceLimit = s.resourceLimit ?? {};
  const maxCpuPercent = resourceLimit.maxCpuPercent ?? "";
  const maxMemoryBytes = resourceLimit.maxMemoryBytes ?? "";
  const maxDiskBytes = resourceLimit.maxDiskBytes ?? "";
  const secretPatterns = joinLines(ensureArray(validator.secretPatterns));
  const approvalEnabled = approvalQueue.enabled === true;
  const approvalTimeoutSeconds = approvalQueue.timeoutSeconds ?? "";
  // Default unchecked when undefined; only checked when explicitly true
  let approvalBlockOnApproval = approvalQueue.blockOnApproval === true;
  const approvalAllow = joinLines(ensureArray(approvalQueue.allow));
  const approvalAsk = joinLines(ensureArray(approvalQueue.ask));
  const approvalDeny = joinLines(ensureArray(approvalQueue.deny));

  const entries = props.approvalsResult?.entries ?? [];
  const storePath = props.approvalsResult?.storePath ?? "";

  return html`
    <section class="card">
      <div class="row" style="justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px;">
        <div>
          <div class="card-title">${t("navTitleSandbox")}</div>
          <div class="card-sub">${t("subtitleSandbox")}</div>
        </div>
      </div>

      <div class="sandbox-sections" style="margin-top: 20px;">
        <details class="sandbox-details" open>
          <summary class="sandbox-summary">
            <span>${t("securitySectionSandbox")}</span>
            <span class="security-help" title=${t("securitySectionSandboxDesc")}>❕</span>
          </summary>
          <div class="sandbox-section-body" style="margin-top: 16px;">
            <div class="muted" style="font-size: 13px; margin-bottom: 12px;">${t("securitySectionSandboxDesc")}</div>
            <div class="row" style="align-items: center; gap: 12px; margin-bottom: 16px;">
              <button
                type="button"
                class="btn ${enabled ? "btn-ok" : ""}"
                ?disabled=${props.saving}
                @click=${props.onToggleEnabled}
              >
                ${enabled ? t("sandboxActionDisable") : t("sandboxActionEnable")}
              </button>
              <span class="muted" style="font-size: 13px;">
                ${enabled ? t("sandboxEnabled") : t("sandboxDisabled")}
              </span>
            </div>
            <div class="sandbox-form-center">
              <div class="field" style="width: 100%; margin-bottom: 16px;">
                <span>${t("sandboxAllowedPaths")}</span>
                <span class="textarea"><textarea
                  rows="3"
                  .value=${allowedPaths}
                  placeholder="/tmp&#10;./workspace"
                  @input=${(e: Event) => {
                    const v = (e.target as HTMLTextAreaElement).value;
                    props.onPatch(["allowedPaths"], splitLines(v));
                  }}
                ></textarea></span>
              </div>
              <div class="field" style="width: 100%; margin-bottom: 16px;">
                <span>${t("sandboxNetworkAllow")}</span>
                <span class="textarea"><textarea
                  rows="2"
                  .value=${networkAllow}
                  placeholder="localhost&#10;127.0.0.1&#10;*.anthropic.com"
                  @input=${(e: Event) => {
                    const v = (e.target as HTMLTextAreaElement).value;
                    props.onPatch(["networkAllow"], splitLines(v));
                  }}
                ></textarea></span>
              </div>

              <div style="margin: 24px 0;">
                <div class="card-sub" style="margin-bottom: 12px; font-size: 14px;">${t("sandboxResourceLimit")}</div>
                <div class="row" style="flex-wrap: wrap; gap: 12px;">
                  <div class="field" style="flex: 1 1 160px; min-width: 0;">
                    <span style="font-size: 14px;">${t("sandboxMaxCPUPercent")}</span>
                    <span class="input"><input
                      type="text"
                      min="0"
                      max="100"
                      step="1"
                      .value=${String(maxCpuPercent)}
                      placeholder="80"
                      @input=${(e: Event) => {
                        const raw = (e.target as HTMLInputElement).value.trim();
                        const v = raw === "" ? undefined : Number(raw);
                        props.onPatch(["resourceLimit", "maxCpuPercent"], Number.isNaN(v as number) ? undefined : v);
                      }}
                    /></span>
                  </div>
                  <div class="field" style="flex: 1 1 220px; min-width: 0;">
                    <span style="font-size: 14px;">${t("sandboxMaxMemoryBytes")}</span>
                    <span class="input"><input
                      type="text"
                      .value=${String(maxMemoryBytes)}
                      placeholder="1G, 512M, 1024"
                      @input=${(e: Event) => {
                        const raw = (e.target as HTMLInputElement).value.trim();
                        props.onPatch(["resourceLimit", "maxMemoryBytes"], raw === "" ? undefined : raw);
                      }}
                    /></span>
                  </div>
                  <div class="field" style="flex: 1 1 220px; min-width: 0;">
                    <span style="font-size: 14px;">${t("sandboxMaxDiskBytes")}</span>
                    <span class="input"><input
                      type="text"
                      .value=${String(maxDiskBytes)}
                      placeholder="10G, 100G, 10240"
                      @input=${(e: Event) => {
                        const raw = (e.target as HTMLInputElement).value.trim();
                        props.onPatch(["resourceLimit", "maxDiskBytes"], raw === "" ? undefined : raw);
                      }}
                    /></span>
                  </div>
                </div>
              </div>

              <div class="row" style="gap: 8px; margin-top: 16px;">
                <button type="button" class="btn primary" ?disabled=${props.saving} @click=${props.onSave}>
                  ${props.saving ? t("commonLoading") : t("commonSave")}
                </button>
              </div>
            </div>
          </div>
        </details>

        <details class="sandbox-details" style="margin-top: 16px;" >
          <summary class="sandbox-summary">
            <span>${t("securitySectionValidator")}</span>
            <span class="security-help" title=${t("securitySectionValidatorDesc")}>❕</span>
          </summary>
          <div class="sandbox-section-body" style="margin-top: 16px;">
            <div class="muted" style="font-size: 13px; margin-bottom: 12px;">${t("securitySectionValidatorDesc")}</div>
            <div class="sandbox-form-center">
              <div class="row" style="align-items: center; gap: 12px; margin-bottom: 16px;">
                <button
                  type="button"
                  class="btn ${validatorEnabled ? "btn-ok" : ""}"
                  ?disabled=${props.saving}
                  @click=${() => {
                    if (props.onToggleValidatorEnabled) {
                      props.onToggleValidatorEnabled();
                    } else {
                      props.onPatch(["validator", "enabled"], !validatorEnabled);
                    }
                  }}
                >
                  ${validatorEnabled ? t("sandboxActionDisable") : t("sandboxActionEnable")}
                </button>
                <span class="muted" style="font-size: 13px;">
                  ${validatorEnabled ? t("sandboxEnabled") : t("sandboxDisabled")}
                </span>
              </div>
              <div class="row" style="flex-direction: column; gap: 12px;">
                <div class="field" style="width: 100%;">
                  <span style="font-size: 14px;">${t("sandboxBanCommands")}</span>
                  <span class="textarea"><textarea
                    rows="2"
                    style="font-size: 14px;"
                    .value=${banCommands}
                    placeholder="dd&#10;mkfs&#10;sudo"
                    @input=${(e: Event) => {
                      const v = (e.target as HTMLTextAreaElement).value;
                      props.onPatch(["validator", "banCommands"], splitLines(v));
                    }}
                  ></textarea></span>
                </div>
                <div class="field" style="width: 100%;">
                  <span style="font-size: 14px;">${t("sandboxBanArguments")}</span>
                  <span class="textarea"><textarea
                    rows="2"
                    style="font-size: 14px;"
                    .value=${banArguments}
                    placeholder="--no-preserve-root&#10;/dev/"
                    @input=${(e: Event) => {
                      const v = (e.target as HTMLTextAreaElement).value;
                      props.onPatch(["validator", "banArguments"], splitLines(v));
                    }}
                  ></textarea></span>
                </div>
                <div class="field" style="width: 100%;">
                  <span style="font-size: 14px;">${t("sandboxBanFragments")}</span>
                  <span class="textarea"><textarea
                    rows="2"
                    style="font-size: 14px;"
                    .value=${banFragments}
                    placeholder="rm -rf&#10;rm -r"
                    @input=${(e: Event) => {
                      const v = (e.target as HTMLTextAreaElement).value;
                      props.onPatch(["validator", "banFragments"], splitLines(v));
                    }}
                  ></textarea></span>
                </div>
                <div class="field" style="width: 100%;">
                  <span style="font-size: 14px;">${t("sandboxSecretPatterns")}</span>
                  <span class="textarea"><textarea
                    rows="3"
                    style="font-size: 14px; font-family: var(--mono);"
                    .value=${secretPatterns}
                    placeholder="sk-[a-zA-Z0-9]{48}&#10;ghp_[a-zA-Z0-9]{36}"
                    @input=${(e: Event) => {
                      const v = (e.target as HTMLTextAreaElement).value;
                      props.onPatch(["validator", "secretPatterns"], splitLines(v));
                    }}
                  ></textarea></span>
                  <div class="muted" style="font-size: 12px; margin-top: 4px;">${t("sandboxSecretPatternsHint")}</div>
                </div>
              </div>

              <div class="row" style="gap: 8px; margin-top: 16px;">
                <button type="button" class="btn primary" ?disabled=${props.saving} @click=${props.onSave}>
                  ${props.saving ? t("commonLoading") : t("commonSave")}
                </button>
              </div>
            </div>
          </div>
        </details>

        <details class="sandbox-details" style="margin-top: 16px;">
          <summary class="sandbox-summary">
            <span>${t("securitySectionApprovalQueue")}</span>
            <span class="security-help" title=${t("securitySectionApprovalQueueDesc")}>❕</span>
          </summary>
          <div class="sandbox-section-body" style="margin-top: 16px;">
            <div class="muted" style="font-size: 13px; margin-bottom: 12px;">${t("securitySectionApprovalQueueDesc")}</div>
            <div class="sandbox-form-center" style="margin-bottom: 12px;">
              <div class="row" style="align-items: center; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; justify-content: flex-start;">
                <button
                  type="button"
                  class="btn ${approvalEnabled ? "btn-ok" : ""}"
                  ?disabled=${props.saving}
                  @click=${() => {
                    if (props.onToggleApprovalEnabled) {
                      props.onToggleApprovalEnabled();
                    } else {
                      props.onPatch(["approvalQueue", "enabled"], !approvalEnabled);
                    }
                  }}
                >
                  ${approvalEnabled ? t("sandboxActionDisable") : t("sandboxActionEnable")}
                </button>
                <span class="muted" style="font-size: 13px;">
                  ${approvalEnabled ? t("sandboxEnabled") : t("sandboxDisabled")}
                </span>
                ${approvalEnabled
                  ? html`
                      <div class="row" style="align-items: flex-start; gap: 8px; margin-left: 4px;">
                        <span class="checkbox"><input
                          type="checkbox"
                          id="blockOnApproval"
                          .checked=${approvalBlockOnApproval}
                          ?disabled=${props.saving}
                          @input=${(e: Event) => {
                            const checked = (e.target as HTMLInputElement).checked;
                            props.onPatch(["approvalQueue", "blockOnApproval"], checked);
                          }}
                        /></span>
                        <div style="flex: 1; min-width: 220px;">
                          <label for="blockOnApproval" style="font-size: 14px; cursor: pointer;">
                            ${t("securityApprovalBlockOnApproval")}
                            <span class="muted" style="font-size: 12px; margin-left: 8px;">
                              ${t("securityApprovalBlockOnApprovalHint")}
                            </span>
                          </label>
                        </div>
                      </div>
                    `
                  : nothing}
              </div>

              <div class="field" style="width: 100%; margin-top: 10px;">
                <span style="font-size: 14px;">${t("securityApprovalTimeoutSeconds")}</span>
                <span class="input"><input
                  type="text"
                  .value=${String(approvalTimeoutSeconds)}
                  placeholder="300"
                  @input=${(e: Event) => {
                    const raw = (e.target as HTMLInputElement).value.trim();
                    const v = raw === "" ? undefined : Number(raw);
                    props.onPatch(["approvalQueue", "timeoutSeconds"], Number.isNaN(v as number) ? undefined : v);
                  }}
                /></span>
                <div class="muted" style="font-size: 12px; margin-top: 4px;">${t("securityApprovalTimeoutSecondsHint")}</div>
              </div>

              <div class="field" style="width: 100%; margin-top: 16px;">
                <span style="font-size: 14px;">${t("securityApprovalAllow")}</span>
                <span class="textarea"><textarea
                  rows="2"
                  style="font-size: 14px;"
                  .value=${approvalAllow}
                  placeholder="ls&#10;pwd&#10;echo"
                  @input=${(e: Event) => {
                    const v = (e.target as HTMLTextAreaElement).value;
                    props.onPatch(["approvalQueue", "allow"], splitLines(v));
                  }}
                ></textarea></span>
                <div class="muted" style="font-size: 12px; margin-top: 4px;">${t("securityApprovalAllowHint")}</div>
              </div>

              <div class="field" style="width: 100%; margin-top: 16px;">
                <span style="font-size: 14px;">${t("securityApprovalAsk")}</span>
                <span class="textarea"><textarea
                  rows="2"
                  style="font-size: 14px;"
                  .value=${approvalAsk}
                  placeholder="rm&#10;mv&#10;cp"
                  @input=${(e: Event) => {
                    const v = (e.target as HTMLTextAreaElement).value;
                    props.onPatch(["approvalQueue", "ask"], splitLines(v));
                  }}
                ></textarea></span>
                <div class="muted" style="font-size: 12px; margin-top: 4px;">${t("securityApprovalAskHint")}</div>
              </div>

              <div class="field" style="width: 100%; margin-top: 16px;">
                <span style="font-size: 14px;">${t("securityApprovalDeny")}</span>
                <span class="textarea"><textarea
                  rows="2"
                  style="font-size: 14px;"
                  .value=${approvalDeny}
                  placeholder="sudo&#10;dd&#10;mkfs"
                  @input=${(e: Event) => {
                    const v = (e.target as HTMLTextAreaElement).value;
                    props.onPatch(["approvalQueue", "deny"], splitLines(v));
                  }}
                ></textarea></span>
                <div class="muted" style="font-size: 12px; margin-top: 4px;">${t("securityApprovalDenyHint")}</div>
              </div>

              <div class="row" style="gap: 8px; margin-top: 12px;">
                <button type="button" class="btn primary" ?disabled=${props.saving} @click=${props.onSave}>
                  ${props.saving ? t("commonLoading") : t("commonSave")}
                </button>
              </div>
            </div>

            <div class="row" style="justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 12px;">
              <div class="muted" style="font-size: 13px;"></div>
              <button class="btn primary" ?disabled=${props.approvalsLoading} @click=${props.onApprovalsRefresh}>
                ${props.approvalsLoading ? t("commonLoading") : t("commonRefresh")}
              </button>
            </div>
            ${props.approvalsError ? html`<div class="callout danger" style="margin-bottom: 12px;">${props.approvalsError}</div>` : nothing}
            <div class="mcp-table table sandbox-approvals-table">
              <div class="mcp-table-head table-head sandbox-approvals-head">
                <div>${t("approvalsId")}</div>
                <div>${t("approvalsSessionKey")}</div>
                <div>${t("approvalsSessionId")}</div>
                <div>${t("approvalsCommand")}</div>
                <div>${t("approvalsTimeout")}</div>
                <div>${t("approvalsTTL")}</div>
                <div>${t("approvalsStatus")}</div>
                <div>${t("mcpTableActions")}</div>
              </div>
              ${
                entries.length === 0
                  ? html`
                      <div class="muted" style="padding: 24px; text-align: center;">
                        ${props.approvalsLoading ? t("commonLoading") : t("approvalsNoEntries")}
                      </div>
                    `
                  : entries.map(
                      (e: ApprovalEntry) => {
                        const canAct = e.status === "pending";
                        const sessionPath = e.sessionKey
                          ? `${props.pathForTab("message")}?session=${encodeURIComponent(e.sessionKey)}`
                          : "";
                        return html`
                          <div class="mcp-table-row table-row">
                            <div class="mcp-table-cell mono">${e.id}</div>
                            <div class="mcp-table-cell mono muted" style="max-width: 160px; overflow: hidden; text-overflow: ellipsis;" title=${e.sessionKey ?? ""}>${e.sessionKey ?? "—"}</div>
                            <div class="mcp-table-cell mono muted">${e.sessionId}</div>
                            <div class="mcp-table-cell mono" style="max-width: 200px; overflow: hidden; text-overflow: ellipsis;" title=${e.command}>${e.command}</div>
                            <div class="mcp-table-cell muted">${formatApprovalTime(e.timeoutAt)}</div>
                            <div class="mcp-table-cell muted">${e.ttlSeconds ?? "—"}</div>
                            <div class="mcp-table-cell">${e.status === "expired" ? t("approvalsExpired") : e.status === "pending" ? t("approvalsPending") : e.status}</div>
                            <div class="mcp-table-cell row" style="gap: 6px; justify-content: flex-end;">
                              ${sessionPath ? html`<a class="btn btn--sm" href="${sessionPath}">${t("approvalsViewSession")}</a>` : nothing}
                              ${canAct
                                ? html`
                                    <button class="btn btn--sm btn-ok" @click=${() => props.onApprove(e.id)}>${t("approvalsApprove")}</button>
                                    <button class="btn btn--sm" @click=${() => props.onDeny(e.id)}>${t("approvalsDeny")}</button>
                                    <button
                                      class="btn btn--sm btn-ok"
                                      @click=${async () => {
                                        const confirmed = await nativeConfirm(
                                          `全部放行后该 sessionId (${e.sessionId}) 对应的会话执行的所有命令将默认通过，不再进入审批队列。确定要全部放行吗？`,
                                        );
                                        if (confirmed) {
                                          props.onWhitelistSession(e.id, e.sessionId);
                                        }
                                      }}
                                    >
                                      全部放行
                                    </button>
                                  `
                                : nothing}
                            </div>
                          </div>
                        `;
                      },
                    )
              }
            </div>
          </div>
        </details>
      </div>
    </section>
  `;
}

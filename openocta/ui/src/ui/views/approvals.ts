import { html, nothing } from "lit";
import { t } from "../strings.js";
import type {
  ApprovalEntry,
  ApprovalsListResult,
  WhitelistEntry,
} from "../controllers/approvals.ts";
import type { Tab } from "../navigation.ts";

export type ApprovalsProps = {
  approvalsLoading: boolean;
  approvalsResult: ApprovalsListResult | null;
  approvalsError: string | null;
  onApprovalsRefresh: () => void;
  onApprove: (requestId: string) => void;
  onDeny: (requestId: string, reason?: string) => void;
  onWhitelistSession: (requestId: string) => void;
  pathForTab: (tab: Tab) => string;
};

function formatApprovalTime(ms: number | undefined): string {
  if (ms == null || ms === 0) return "—";
  return new Date(ms).toLocaleString();
}

function formatTtl(ttlSeconds: number | undefined): string {
  if (ttlSeconds == null) return "—";
  if (ttlSeconds < 0) return t("approvalsTtlPermanent");
  if (ttlSeconds < 60) return `${ttlSeconds}s`;
  if (ttlSeconds < 3600) return `${Math.floor(ttlSeconds / 60)}m`;
  return `${Math.floor(ttlSeconds / 3600)}h`;
}

function renderStatusBadge(status: string): string {
  const labels: Record<string, string> = {
    pending: t("approvalsPending"),
    approved: t("approvalsSectionApproved"),
    denied: t("approvalsSectionDenied"),
    expired: t("approvalsExpired"),
    whitelisted: t("approvalsSectionWhitelisted"),
    whitelist_expired: t("approvalsExpired"),
  };
  return labels[status] ?? status;
}

/** 排序：有效期的放最上面。pending 未过期 > approved 未过期 > expired > denied */
function sortQueueEntries(entries: ApprovalEntry[]): ApprovalEntry[] {
  return [...entries].sort((a, b) => {
    const score = (e: ApprovalEntry) => {
      const isPending = e.status === "pending";
      const isExpired = e.status === "expired" || e.expired === true;
      const isDenied = e.status === "denied";
      if (isPending && !isExpired) return 0; // 待审批未过期，最前
      if (e.status === "approved" && !isExpired) return 1; // 已审批未过期
      if (isExpired) return 2; // 已过期
      if (isDenied) return 3; // 已拒绝
      return 1;
    };
    return score(a) - score(b);
  });
}

export function renderApprovals(props: ApprovalsProps) {
  const r = props.approvalsResult;
  const whitelisted = r?.whitelisted ?? [];
  const queueEntries = sortQueueEntries([...(r?.entries ?? [])]);

  return html`
    <section class="card">
      <div class="row" style="justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px;">
        <div>
          <div class="card-title">${t("navTitleApprovals")}</div>
          <div class="card-sub">${t("subtitleApprovals")}</div>
        </div>
        <button class="btn primary" ?disabled=${props.approvalsLoading} @click=${props.onApprovalsRefresh}>
          ${props.approvalsLoading ? t("commonLoading") : t("commonRefresh")}
        </button>
      </div>

      ${props.approvalsError ? html`<div class="callout danger" style="margin-top: 16px;">${props.approvalsError}</div>` : nothing}

      <!-- 审批队列：已审批 + 待审批 合并，有效期的放最上面 -->
      <div style="margin-top: 20px;">
        <div class="card-sub" style="margin-bottom: 12px; font-size: 14px;">${t("approvalsList")} (${queueEntries.length})</div>
        ${queueEntries.length === 0 && !props.approvalsLoading
          ? html`<div class="muted" style="padding: 24px; text-align: center;">${t("approvalsNoEntries")}</div>`
          : html`
              <div class="mcp-table table approvals-queue-table">
                <div class="mcp-table-head table-head">
                  <div>${t("approvalsCommand")}</div>
                  <div>${t("approvalsStatus")}</div>
                  <div>${t("approvalsExpiresAt")}</div>
                  <div>${t("approvalsTTL")}</div>
                  <div style="text-align: right;">${t("mcpTableActions")}</div>
                </div>
                ${queueEntries.map((e) => {
                  const sessionKey = e.sessionKey ?? e.sessionId;
                  const sessionPath = sessionKey ? `${props.pathForTab("message")}?session=${encodeURIComponent(sessionKey)}` : "";
                  const canAct = e.status === "pending" && !e.expired;
                  const expiresAt = e.expiresAt ?? e.timeoutAt;
                  return html`
                    <div class="mcp-table-row table-row">
                      <div class="mcp-table-cell mono" style="max-width: 240px; overflow: hidden; text-overflow: ellipsis;" title=${e.command}>${e.command}</div>
                      <div class="mcp-table-cell">${renderStatusBadge(e.status)}${e.approver ? ` · ${e.approver}` : ""}</div>
                      <div class="mcp-table-cell muted" style="font-size: 12px;">${formatApprovalTime(expiresAt)}</div>
                      <div class="mcp-table-cell muted" style="font-size: 12px;">${e.ttlSeconds != null ? formatTtl(e.ttlSeconds) : "—"}</div>
                      <div class="mcp-table-cell approvals-actions-cell">
                        <button class="btn btn--sm btn-ok" ?disabled=${!canAct} @click=${() => canAct && props.onApprove(e.id)}>${t("approvalsApproveOnce")}</button>
                        <button class="btn btn--sm" ?disabled=${!canAct} @click=${() => canAct && props.onWhitelistSession(e.id)}>${t("approvalsWhitelistSession")}</button>
                        <button class="btn btn--sm" style="color: var(--danger);" ?disabled=${!canAct} @click=${() => canAct && props.onDeny(e.id)}>${t("approvalsDeny")}</button>
                        ${sessionPath ? html`<a class="btn btn--sm" href="${sessionPath}">${t("approvalsViewSession")}</a>` : nothing}
                      </div>
                    </div>
                  `;
                })}
              </div>
            `}
      </div>

      <!-- 会话免审：独立模块 -->
      ${whitelisted.length > 0 ? html`
        <div style="margin-top: 24px;">
          <div class="card-sub" style="margin-bottom: 12px; font-size: 14px;">${t("approvalsSectionWhitelisted")} (${whitelisted.length})</div>
          <div class="mcp-table table">
            <div class="mcp-table-head table-head">
              <div>${t("approvalsSessionId")}</div>
              <div>${t("approvalsStatus")}</div>
              <div>${t("approvalsExpiresAt")}</div>
              <div>${t("approvalsTTL")}</div>
            </div>
            ${whitelisted.map((e: WhitelistEntry) => html`
              <div class="mcp-table-row table-row">
                <div class="mcp-table-cell mono" style="max-width: 200px; overflow: hidden; text-overflow: ellipsis;">${e.sessionId}</div>
                <div class="mcp-table-cell">${renderStatusBadge(e.status)}</div>
                <div class="mcp-table-cell muted" style="font-size: 12px;">${e.expiresAt ? formatApprovalTime(e.expiresAt) : t("approvalsTtlPermanent")}</div>
                <div class="mcp-table-cell muted" style="font-size: 12px;">${e.ttlSeconds != null ? (e.ttlSeconds < 0 ? t("approvalsTtlPermanent") : formatTtl(e.ttlSeconds)) : "—"}</div>
              </div>
            `)}
          </div>
        </div>
      ` : nothing}
    </section>
  `;
}

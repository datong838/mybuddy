import type { OpenClawApp } from "./app.ts";
import type { ApprovalsListResult } from "./controllers/approvals.ts";
import {
  collectActivePendingApprovalIds,
  normalizeApprovalsListResult,
} from "./controllers/approvals.ts";

export const APPROVAL_BANNER_POLL_MS = 3000;

export async function runApprovalsBannerPoll(host: OpenClawApp): Promise<void> {
  if (!host.client || !host.connected) {
    return;
  }
  try {
    const res = await host.client.request<ApprovalsListResult | undefined>("approvals.list", {});
    const result = normalizeApprovalsListResult(res);
    host.approvalsResult = result;

    const currentIds = new Set(collectActivePendingApprovalIds(result));
    const baseline = new Set(host.approvalBannerBaselineIds);
    for (const id of baseline) {
      if (!currentIds.has(id)) {
        baseline.delete(id);
      }
    }

    const newIds = [...currentIds].filter((id) => !baseline.has(id));

    if (!host.approvalBannerPollInitialized) {
      host.approvalBannerPollInitialized = true;
      host.approvalBannerBaselineIds = [...currentIds];
      if (currentIds.size > 0) {
        host.approvalBannerVisible = true;
        host.approvalBannerPendingCount = currentIds.size;
      } else {
        host.approvalBannerPendingCount = 0;
      }
      return;
    }

    if (newIds.length > 0) {
      for (const id of newIds) {
        baseline.add(id);
      }
      host.approvalBannerVisible = true;
    }
    host.approvalBannerBaselineIds = [...baseline];
    if (currentIds.size === 0) {
      host.approvalBannerVisible = false;
      host.approvalBannerPendingCount = 0;
    } else if (host.approvalBannerVisible) {
      host.approvalBannerPendingCount = currentIds.size;
    }
  } catch {
    /* 轮询失败不打扰用户 */
  }
}

export function startApprovalsBannerPolling(host: OpenClawApp): void {
  if (host.approvalBannerPollInterval != null) {
    return;
  }
  host.approvalBannerPollInterval = window.setInterval(() => {
    void runApprovalsBannerPoll(host);
  }, APPROVAL_BANNER_POLL_MS);
}

export function stopApprovalsBannerPolling(host: OpenClawApp): void {
  if (host.approvalBannerPollInterval == null) {
    return;
  }
  clearInterval(host.approvalBannerPollInterval);
  host.approvalBannerPollInterval = null;
}

export function resetApprovalsBannerState(host: OpenClawApp): void {
  host.approvalBannerVisible = false;
  host.approvalBannerPollInitialized = false;
  host.approvalBannerBaselineIds = [];
  host.approvalBannerPendingCount = 0;
}

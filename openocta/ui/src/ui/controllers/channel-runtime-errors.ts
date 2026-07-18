import type { GatewayBrowserClient } from "../gateway.ts";
import type { ChannelsStatusSnapshot } from "../types.ts";

/** 避免从 app.ts 回引入造成循环依赖 */
export type ChannelRuntimeErrorHost = {
  client: GatewayBrowserClient | null;
  connected: boolean;
  channelsLoading: boolean;
  channelsSnapshot: ChannelsStatusSnapshot | null;
};
import { nativeAlert } from "../native-dialog-bridge.ts";
import { t } from "../strings.js";
import type { ChannelsState } from "./channels.types.ts";
import { loadChannels } from "./channels.ts";

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/** 配置里标记为启用的渠道 id（小写），用于保存后探测运行时是否报错 */
export function channelIdsEnabledInFormChannels(channels: Record<string, unknown> | null | undefined): string[] {
  if (!channels || typeof channels !== "object") {
    return [];
  }
  const out: string[] = [];
  for (const [key, raw] of Object.entries(channels)) {
    if (key === "defaults") {
      continue;
    }
    if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
      continue;
    }
    if ((raw as Record<string, unknown>).enabled === true) {
      out.push(key.toLowerCase());
    }
  }
  return out;
}

function labelForChannel(snapshot: ChannelsStatusSnapshot | null, id: string): string {
  const lower = id.toLowerCase();
  return (
    snapshot?.channelLabels?.[lower] ??
    snapshot?.channelLabels?.[id] ??
    snapshot?.channelDetailLabels?.[lower] ??
    id
  );
}

function resolveLastErrorFromSnapshot(
  snapshot: ChannelsStatusSnapshot | null,
  channelId: string,
): string {
  const lower = channelId.toLowerCase();
  const summary = snapshot?.channels?.[lower] ?? snapshot?.channels?.[channelId];
  if (summary && typeof summary === "object" && !Array.isArray(summary)) {
    const le = (summary as Record<string, unknown>).lastError;
    if (typeof le === "string" && le.trim()) {
      return le.trim();
    }
  }
  const accounts = snapshot?.channelAccounts?.[lower] ?? snapshot?.channelAccounts?.[channelId] ?? [];
  for (const acc of accounts) {
    const a = acc?.lastError;
    if (typeof a === "string" && a.trim()) {
      return a.trim();
    }
  }
  return "";
}

/** 多次刷新 channels.status，收集启用渠道的 lastError（运行时启动/连接失败会写在此处） */
export async function probeChannelRuntimeErrors(
  host: ChannelRuntimeErrorHost,
  channelIds: string[],
): Promise<Array<{ id: string; label: string; message: string }>> {
  if (!channelIds.length || !host.client || !host.connected) {
    return [];
  }
  const delaysMs = [350, 650, 1100];
  const seen = new Map<string, string>();

  for (const ms of delaysMs) {
    await sleep(ms);
    for (let i = 0; i < 60 && host.channelsLoading; i++) {
      await sleep(50);
    }
    await loadChannels(host as ChannelsState, true);
    const snap = host.channelsSnapshot;
    for (const id of channelIds) {
      const msg = resolveLastErrorFromSnapshot(snap, id);
      if (msg) {
        seen.set(id.toLowerCase(), msg);
      }
    }
  }

  const out: Array<{ id: string; label: string; message: string }> = [];
  for (const id of channelIds) {
    const key = id.toLowerCase();
    const msg = seen.get(key);
    if (msg) {
      out.push({ id: key, label: labelForChannel(host.channelsSnapshot, key), message: msg });
    }
  }
  return out;
}

/**
 * 保存渠道配置后：若表单中有已启用渠道，则探测网关侧运行时错误并弹窗提示。
 * @param opts.onlyChannelIds 若提供，只检查这些 id（通常为当前配置面板对应的渠道），避免其它已启用渠道的残留 lastError 误报。
 */
export async function alertChannelRuntimeErrorsIfAny(
  host: ChannelRuntimeErrorHost,
  channelsFromForm: Record<string, unknown> | null | undefined,
  opts?: { onlyChannelIds?: string[] },
): Promise<void> {
  let enabledIds = channelIdsEnabledInFormChannels(channelsFromForm);
  if (opts?.onlyChannelIds?.length) {
    const allow = new Set(opts.onlyChannelIds.map((x) => x.toLowerCase()));
    enabledIds = enabledIds.filter((id) => allow.has(id));
  }
  if (!enabledIds.length) {
    return;
  }
  const issues = await probeChannelRuntimeErrors(host, enabledIds);
  if (!issues.length) {
    return;
  }
  const body = issues.map((x) => `${x.label}（${x.id}）\n${x.message}`).join("\n\n—\n\n");
  await nativeAlert(`${t("channelsRuntimeStartErrorTitle")}\n\n${body}`);
}

import { html, nothing } from "lit";
import type {
  ChannelAccountSnapshot,
  ChannelUiMetaEntry,
  ChannelsStatusSnapshot,
  WeWorkStatus,
  WeixinStatus,
} from "../types.ts";
import type { ChannelKey, ChannelsChannelData, ChannelsProps } from "./channels.types.ts";
import { formatAgo } from "../format.ts";
import { t } from "../strings.js";
import { renderChannelConfigPanel } from "./channels.config.ts";
import { channelEnabled, renderChannelAccountCount } from "./channels.shared.ts";
import { renderWeWorkCard } from "./channels.wework.ts";
import { renderWeixinCard } from "./channels.weixin.ts";

export function renderChannels(props: ChannelsProps) {
  const channels = props.snapshot?.channels as Record<string, unknown> | null;
  const wework = (channels?.wework ?? undefined) as WeWorkStatus | undefined;
  const weixin = (channels?.weixin ?? undefined) as WeixinStatus | undefined;
  const channelOrder = resolveChannelOrder(props.snapshot);
  const orderedChannels = channelOrder
    .map((key, index) => ({
      key,
      enabled: channelEnabled(key, props),
      order: index,
    }))
    .toSorted((a, b) => {
      if (a.enabled !== b.enabled) {
        return a.enabled ? -1 : 1;
      }
      return a.order - b.order;
    });

  return html`
    <section class="grid grid-cols-2">
      ${orderedChannels.map((channel) =>
        renderChannel(channel.key, props, {
          wework,
          weixin,
          channelAccounts: props.snapshot?.channelAccounts ?? null,
        }),
      )}
    </section>

    ${renderChannelConfigPanel(props)}

    <section class="card">
      <div class="row" style="justify-content: space-between;">
        <div>
          <div class="card-title">${t("channelsHealth")}</div>
          <div class="card-sub">${t("channelsHealthSub")}</div>
        </div>
        <div class="muted">${props.lastSuccessAt ? formatAgo(props.lastSuccessAt) : t("commonNA")}</div>
      </div>
      ${
        props.lastError
          ? html`<div class="callout danger" style="margin-top: 12px;">
            ${props.lastError}
          </div>`
          : nothing
      }
      <pre class="code-block" style="margin-top: 12px;">
${props.snapshot ? JSON.stringify(props.snapshot, null, 2) : t("channelsNoSnapshot")}
      </pre>
    </section>
  `;
}

function resolveChannelOrder(snapshot: ChannelsStatusSnapshot | null): ChannelKey[] {
  if (snapshot?.channelMeta?.length) {
    return snapshot.channelMeta.map((entry) => entry.id);
  }
  if (snapshot?.channelOrder?.length) {
    return snapshot.channelOrder;
  }
  return ["wework", "weixin", "dingtalk", "feishu", "qq"];
}

function renderChannel(key: ChannelKey, props: ChannelsProps, data: ChannelsChannelData) {
  const accountCountLabel = renderChannelAccountCount(key, data.channelAccounts);
  switch (key) {
    case "wework":
      return renderWeWorkCard({
        props,
        wework: data.wework,
        accountCountLabel,
      });
    case "weixin":
      return renderWeixinCard({
        props,
        weixin: data.weixin,
        accountCountLabel,
      });
    default:
      return renderGenericChannelCard(key, props, data.channelAccounts ?? {});
  }
}

function renderGenericChannelCard(
  key: ChannelKey,
  props: ChannelsProps,
  channelAccounts: Record<string, ChannelAccountSnapshot[]>,
) {
  const label = resolveChannelLabel(props.snapshot, key);
  const status = props.snapshot?.channels?.[key] as Record<string, unknown> | undefined;
  const configured = typeof status?.configured === "boolean" ? status.configured : undefined;
  const running = typeof status?.running === "boolean" ? status.running : undefined;
  const connected = typeof status?.connected === "boolean" ? status.connected : undefined;
  const lastError = typeof status?.lastError === "string" ? status.lastError : undefined;
  const accounts = channelAccounts[key] ?? [];
  const accountCountLabel = renderChannelAccountCount(key, channelAccounts);

  return html`
    <div class="card">
      <div class="card-title">${label}</div>
      <div class="card-sub">${t("channelGenericSub")}</div>
      ${accountCountLabel}

      ${
        accounts.length > 0
          ? html`
            <div class="account-card-list">
              ${accounts.map((account) => renderGenericAccount(account))}
            </div>
          `
          : html`
            <div class="status-list" style="margin-top: 16px;">
              <div>
                <span class="label">${t("channelConfigured")}</span>
                <span>${configured == null ? t("commonNA") : configured ? t("commonYes") : t("commonNo")}</span>
              </div>
              <div>
                <span class="label">${t("channelRunning")}</span>
                <span>${running == null ? t("commonNA") : running ? t("commonYes") : t("commonNo")}</span>
              </div>
              <div>
                <span class="label">${t("channelConnected")}</span>
                <span>${connected == null ? t("commonNA") : connected ? t("commonYes") : t("commonNo")}</span>
              </div>
            </div>
          `
      }

      ${
        lastError
          ? html`<div class="callout danger" style="margin-top: 12px;">
            ${lastError}
          </div>`
          : nothing
      }

      <div class="row" style="margin-top: 12px;">
        <button class="btn primary" @click=${() => props.onChannelSelect(key)}>
          ${t("channelsConfigure")}
        </button>
        <button class="btn" @click=${() => props.onRefresh(true)}>
          ${t("commonRefresh")}
        </button>
      </div>
    </div>
  `;
}

function resolveChannelMetaMap(
  snapshot: ChannelsStatusSnapshot | null,
): Record<string, ChannelUiMetaEntry> {
  if (!snapshot?.channelMeta?.length) {
    return {};
  }
  return Object.fromEntries(snapshot.channelMeta.map((entry) => [entry.id, entry]));
}

function resolveChannelLabel(snapshot: ChannelsStatusSnapshot | null, key: string): string {
  const meta = resolveChannelMetaMap(snapshot)[key];
  return meta?.label ?? snapshot?.channelLabels?.[key] ?? key;
}

const RECENT_ACTIVITY_THRESHOLD_MS = 10 * 60 * 1000; // 10 minutes

function hasRecentActivity(account: ChannelAccountSnapshot): boolean {
  if (!account.lastInboundAt) {
    return false;
  }
  return Date.now() - account.lastInboundAt < RECENT_ACTIVITY_THRESHOLD_MS;
}

type StatusKey = "commonYes" | "commonNo" | "channelActive" | "commonNA";

function deriveRunningStatus(account: ChannelAccountSnapshot): StatusKey {
  if (account.running) {
    return "commonYes";
  }
  if (hasRecentActivity(account)) {
    return "channelActive";
  }
  return "commonNo";
}

function deriveConnectedStatus(account: ChannelAccountSnapshot): StatusKey {
  if (account.connected === true) {
    return "commonYes";
  }
  if (account.connected === false) {
    return "commonNo";
  }
  if (hasRecentActivity(account)) {
    return "channelActive";
  }
  return "commonNA";
}

function renderGenericAccount(account: ChannelAccountSnapshot) {
  const runningStatus = deriveRunningStatus(account);
  const connectedStatus = deriveConnectedStatus(account);

  return html`
    <div class="account-card">
      <div class="account-card-header">
        <div class="account-card-title">${account.name || account.accountId}</div>
        <div class="account-card-id">${account.accountId}</div>
      </div>
      <div class="status-list account-card-status">
        <div>
          <span class="label">${t("channelRunning")}</span>
          <span>${t(runningStatus)}</span>
        </div>
        <div>
          <span class="label">${t("channelConfigured")}</span>
          <span>${account.configured ? t("commonYes") : t("commonNo")}</span>
        </div>
        <div>
          <span class="label">${t("channelConnected")}</span>
          <span>${t(connectedStatus)}</span>
        </div>
        <div>
          <span class="label">${t("channelLastInbound")}</span>
          <span>${account.lastInboundAt ? formatAgo(account.lastInboundAt) : t("commonNA")}</span>
        </div>
        ${
          account.lastError
            ? html`
              <div class="account-card-error">
                ${account.lastError}
              </div>
            `
            : nothing
        }
      </div>
    </div>
  `;
}

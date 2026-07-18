import type {
  ChannelAccountSnapshot,
  ChannelsStatusSnapshot,
  ConfigUiHints,
  NostrProfile,
  WeWorkStatus,
  WeixinStatus,
} from "../types.ts";
import type { NostrProfileFormState } from "./channels.nostr-profile-form.ts";

export type ChannelKey = string;

export type ChannelsProps = {
  connected: boolean;
  loading: boolean;
  snapshot: ChannelsStatusSnapshot | null;
  lastError: string | null;
  lastSuccessAt: number | null;
  /** 本地已配置的数字员工列表（用于渠道配置下拉选择） */
  digitalEmployees?: Array<{
    id: string;
    name?: string;
    enabled?: boolean;
    builtin?: boolean;
    from?: string;
    type?: string;
  }>;
  /** 数字员工列表是否仍在加载中（用于避免误判“已删除”提示） */
  digitalEmployeesLoading?: boolean;
  whatsappMessage: string | null;
  whatsappQrDataUrl: string | null;
  whatsappConnected: boolean | null;
  whatsappBusy: boolean;
  weworkQrModalOpen: boolean;
  weworkQrModalLoading: boolean;
  weworkQrModalPolling: boolean;
  weworkQrModalSuccess: boolean;
  weworkQrModalError: string | null;
  weworkQrModalReplaceWarn: boolean;
  weworkQrModalAuthUrl: string | null;
  weworkQrModalGenPageUrl: string | null;
  weixinQrModalOpen: boolean;
  weixinQrModalLoading: boolean;
  weixinQrModalPolling: boolean;
  weixinQrModalSuccess: boolean;
  weixinQrModalError: string | null;
  weixinQrModalReplaceWarn: boolean;
  weixinQrModalImageSrc: string | null;
  /** liteapp 等 http(s) 扫码页，用于新窗口打开 */
  weixinQrModalScanPageUrl: string | null;
  weixinQrModalScanned: boolean;
  configSchema: unknown;
  configSchemaLoading: boolean;
  configForm: Record<string, unknown> | null;
  configUiHints: ConfigUiHints;
  configSaving: boolean;
  configFormDirty: boolean;
  /** Channel key being edited in the slide-out panel (e.g. "discord", "feishu") */
  selectedChannelId: string | null;
  nostrProfileFormState: NostrProfileFormState | null;
  nostrProfileAccountId: string | null;
  onRefresh: (probe: boolean) => void;
  onChannelSelect: (channelId: string | null) => void;
  onWhatsAppStart: (force: boolean) => void;
  onWhatsAppWait: () => void;
  onWhatsAppLogout: () => void;
  onWeWorkQrStart: () => void;
  onWeWorkQrModalClose: () => void;
  onWeixinQrStart: () => void;
  onWeixinQrModalClose: () => void;
  onConfigPatch: (path: Array<string | number>, value: unknown) => void;
  onConfigSave: () => void;
  onConfigReload: () => void;
  onNostrProfileEdit: (accountId: string, profile: NostrProfile | null) => void;
  onNostrProfileCancel: () => void;
  onNostrProfileFieldChange: (field: keyof NostrProfile, value: string) => void;
  onNostrProfileSave: () => void;
  onNostrProfileImport: () => void;
  onNostrProfileToggleAdvanced: () => void;
};

export type ChannelsChannelData = {
  wework?: WeWorkStatus;
  weixin?: WeixinStatus;
  channelAccounts?: Record<string, ChannelAccountSnapshot[]> | null;
};

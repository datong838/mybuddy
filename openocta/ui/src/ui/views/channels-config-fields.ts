/**
 * Channel config form field definitions based on src/pkg/channels config_runtime.go files.
 * Each channel type has required and optional fields.
 */

export type ChannelFieldDef = {
  path: string[];
  label: string;
  required: boolean;
  type: "string" | "number" | "boolean" | "string[]";
  placeholder?: string;
  /** Shown under the control in the channel config panel */
  help?: string;
};

export type ChannelFormDef = {
  fields: ChannelFieldDef[];
};

const CHANNEL_FORM_DEFS: Record<string, ChannelFormDef> = {
  feishu: {
    fields: [
      { path: ["credentials", "appId"], label: "App ID", required: true, type: "string", placeholder: "cli_xxx" },
      { path: ["credentials", "appSecret"], label: "App Secret", required: true, type: "string", placeholder: "xxx" },
      { path: ["credentials", "domain"], label: "Domain", required: false, type: "string", placeholder: "open.feishu.cn" },
      { path: ["credentials", "encryptKey"], label: "Encrypt Key", required: false, type: "string" },
      { path: ["credentials", "verificationToken"], label: "Verification Token", required: false, type: "string" },
      { path: ["enabled"], label: "Enabled", required: false, type: "boolean" },
      { path: ["allowedIds"], label: "Allowed IDs", required: false, type: "string[]", placeholder: "user-id-1, user-id-2" },
    ],
  },
  dingtalk: {
    fields: [
      { path: ["credentials", "clientId"], label: "Client ID", required: true, type: "string", placeholder: "your-client-id" },
      { path: ["credentials", "clientSecret"], label: "Client Secret", required: true, type: "string", placeholder: "your-client-secret" },
      { path: ["enabled"], label: "Enabled", required: false, type: "boolean" },
      { path: ["allowedIds"], label: "Allowed IDs", required: false, type: "string[]", placeholder: "user-id-1, user-id-2" },
    ],
  },
  weixin: {
    fields: [
      {
        path: ["credentials", "botToken"],
        label: "Bot Token",
        required: true,
        type: "string",
        placeholder: "iLink 扫码登录后的 bot_token",
        help: "个人微信 iLink 通道凭据（与企业微信的 Bot Secret 不同）。扫码成功后自动写入。",
      },
      {
        path: ["credentials", "botId"],
        label: "Bot ID (iLink)",
        required: true,
        type: "string",
        placeholder: "ilink_bot_id",
        help: "iLink 侧 Bot ID，可与 openclaw-weixin / openilink 文档中的 ilink_bot_id 对应。",
      },
      {
        path: ["credentials", "baseUrl"],
        label: "API Base URL",
        required: false,
        type: "string",
        placeholder: "https://ilinkai.weixin.qq.com",
        help: "一般留空；若扫码返回了专属 baseurl 可填在此。",
      },
      {
        path: ["credentials", "userId"],
        label: "绑定的微信 userId（可选）",
        required: false,
        type: "string",
        placeholder: "ilink_user_id",
        help: "登录账号标识，便于辨认；不影响连接。",
      },
      {
        path: ["credentials", "botType"],
        label: "bot_type（可选）",
        required: false,
        type: "string",
        placeholder: "默认 3",
        help: "仅在使用非默认扫码参数时填写。",
      },
      {
        path: ["credentials", "routeTag"],
        label: "SKRouteTag（可选）",
        required: false,
        type: "string",
        help: "与服务端路由相关的可选请求头。",
      },
      {
        path: ["credentials", "getUpdatesBuf"],
        label: "get_updates_buf（可选）",
        required: false,
        type: "string",
        help: "长轮询同步游标；高级用法，用于断点续拉（多数场景留空）。",
      },
      { path: ["enabled"], label: "Enabled", required: false, type: "boolean" },
      {
        path: ["allowedIds"],
        label: "Allowed peer user IDs",
        required: false,
        type: "string[]",
        placeholder: "ilink_user_id_1, …",
        help: "空表示不限制；仅允许列表内用户与 Bot 的对话进入系统。",
      },
    ],
  },
  wework: {
    fields: [
      {
        path: ["credentials", "botId"],
        label: "Bot ID",
        required: true,
        type: "string",
        placeholder: "智能机器人 BotId",
        help: "企业微信后台或扫码创建后获得，对应长连接 BotID。",
      },
      {
        path: ["credentials", "botSecret"],
        label: "Bot Secret",
        required: true,
        type: "string",
        placeholder: "机器人 Secret",
        help: "与 Bot ID 配对，请妥善保管。",
      },
      {
        path: ["credentials", "wsUrl"],
        label: "WebSocket URL",
        required: false,
        type: "string",
        placeholder: "wss://openws.work.weixin.qq.com",
        help: "一般留空使用官方默认地址。",
      },
      { path: ["enabled"], label: "Enabled", required: false, type: "boolean" },
      {
        path: ["allowedIds"],
        label: "Allowed user IDs",
        required: false,
        type: "string[]",
        placeholder: "userid-1, userid-2",
        help: "空表示不限制；仅允许列表内用户发消息进入系统。",
      },
    ],
  },
  qq: {
    fields: [
      { path: ["credentials", "appId"], label: "App ID", required: true, type: "string", placeholder: "your-app-id" },
      { path: ["credentials", "appSecret"], label: "App Secret", required: true, type: "string", placeholder: "your-app-secret" },
      { path: ["enabled"], label: "Enabled", required: false, type: "boolean" },
      { path: ["allowedIds"], label: "Allowed IDs", required: false, type: "string[]", placeholder: "user-openid-1" },
    ],
  },
};

export function getChannelFormDef(channelId: string): ChannelFormDef | null {
  const id = channelId.toLowerCase();
  return CHANNEL_FORM_DEFS[id] ?? null;
}

export function getValueAtPath(obj: Record<string, unknown>, path: string[]): unknown {
  let current: unknown = obj;
  for (const key of path) {
    if (current == null || typeof current !== "object") {
      return undefined;
    }
    current = (current as Record<string, unknown>)[key];
  }
  return current;
}

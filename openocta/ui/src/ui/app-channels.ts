import type { OpenClawApp } from "./app.ts";
import type { NostrProfile } from "./types.ts";
import {
  loadChannels,
  logoutWhatsApp,
  startWhatsAppLogin,
  waitWhatsAppLogin,
} from "./controllers/channels.ts";
import {
  loadConfig,
  saveConfig,
  saveConfigPatch,
  updateConfigFormValue,
  type ConfigState,
} from "./controllers/config.ts";
import { cloneConfigObject } from "./controllers/config/form-utils.ts";
import {
  alertChannelRuntimeErrorsIfAny,
  probeChannelRuntimeErrors,
} from "./controllers/channel-runtime-errors.ts";
import { nativeConfirm } from "./native-dialog-bridge.ts";
import { t } from "./strings.js";
import { createNostrProfileFormState } from "./views/channels.nostr-profile-form.ts";
import { resolveWeixinQrDisplay } from "./views/channels.weixin.ts";

function clearWeWorkQrPoll(host: OpenClawApp) {
  const id = host.weworkQrPollTimer;
  if (id != null) {
    window.clearInterval(id);
    host.weworkQrPollTimer = null;
  }
}

function clearWeWorkQrSuccessTimer(host: OpenClawApp) {
  const id = host.weworkQrSuccessCloseTimer;
  if (id != null) {
    window.clearTimeout(id);
    host.weworkQrSuccessCloseTimer = null;
  }
}

function hasWeWorkBotConfigured(host: OpenClawApp): boolean {
  const channels = host.configForm?.channels as Record<string, unknown> | undefined;
  const wework = channels?.wework as Record<string, unknown> | undefined;
  const creds = wework?.credentials as Record<string, unknown> | undefined;
  const botId = typeof creds?.botId === "string" ? creds.botId.trim() : "";
  const botSecret = typeof creds?.botSecret === "string" ? creds.botSecret.trim() : "";
  if (botId !== "" && botSecret !== "") {
    return true;
  }
  const snap = host.channelsSnapshot?.channels?.wework as { configured?: boolean } | undefined;
  if (snap?.configured) {
    return true;
  }
  const acc = host.channelsSnapshot?.channelAccounts?.wework?.[0];
  if (acc?.configured) {
    return true;
  }
  return false;
}

/** 关闭弹框并清理轮询（不重置表单中已写入的凭据） */
function resetWeWorkQrModal(host: OpenClawApp) {
  clearWeWorkQrPoll(host);
  clearWeWorkQrSuccessTimer(host);
  host.weworkQrModalOpen = false;
  host.weworkQrModalLoading = false;
  host.weworkQrModalPolling = false;
  host.weworkQrModalSuccess = false;
  host.weworkQrModalError = null;
  host.weworkQrModalReplaceWarn = false;
  host.weworkQrModalAuthUrl = null;
  host.weworkQrModalGenPageUrl = null;
}

async function pollWeWorkQrOnce(host: OpenClawApp, scode: string) {
  if (!host.client || !host.connected) {
    clearWeWorkQrPoll(host);
    host.weworkQrModalPolling = false;
    return;
  }
  try {
    const res = await host.client.request<{
      status?: string;
      botId?: string;
      botSecret?: string;
    }>("channels.wework.qr.poll", { scode, timeoutMs: 35000 });
    const st = (res.status ?? "").trim();
    if (st === "success" && res.botId && res.botSecret) {
      clearWeWorkQrPoll(host);
      updateConfigFormValue(host, ["channels", "wework", "credentials", "botId"], res.botId);
      updateConfigFormValue(host, ["channels", "wework", "credentials", "botSecret"], res.botSecret);
      updateConfigFormValue(host, ["channels", "wework", "enabled"], true);

      const ch = host.configForm?.channels as Record<string, unknown> | undefined;
      const wework = ch?.wework;
      if (!wework || typeof wework !== "object") {
        host.weworkQrModalError = t("channelWeWorkQrSaveMissingForm");
        host.weworkQrModalPolling = false;
        return;
      }

      host.lastError = null;
      await saveConfigPatch(host as ConfigState, {
        channels: { wework: cloneConfigObject(wework as Record<string, unknown>) },
      });
      if (host.lastError) {
        host.weworkQrModalError = host.lastError;
        host.weworkQrModalPolling = false;
        return;
      }

      const runtimeIssues = await probeChannelRuntimeErrors(host, ["wework"]);
      if (runtimeIssues.length) {
        host.weworkQrModalError = runtimeIssues.map((x) => `${x.label}: ${x.message}`).join("\n");
        host.weworkQrModalPolling = false;
        return;
      }

      host.weworkQrModalPolling = false;
      host.weworkQrModalSuccess = true;
      host.weworkQrModalAuthUrl = null;
      host.weworkQrModalGenPageUrl = null;
      clearWeWorkQrSuccessTimer(host);
      host.weworkQrSuccessCloseTimer = window.setTimeout(() => {
        host.weworkQrSuccessCloseTimer = null;
        resetWeWorkQrModal(host);
        void loadChannels(host, true);
      }, 1600);
    }
  } catch (err) {
    host.weworkQrModalError = String(err);
    clearWeWorkQrPoll(host);
    host.weworkQrModalPolling = false;
  }
}

export async function handleWeWorkQrStart(host: OpenClawApp) {
  if (!host.client || !host.connected) {
    return;
  }
  clearWeWorkQrPoll(host);
  clearWeWorkQrSuccessTimer(host);
  host.weworkQrModalOpen = true;
  host.weworkQrModalReplaceWarn = hasWeWorkBotConfigured(host);
  host.weworkQrModalError = null;
  host.weworkQrModalSuccess = false;
  host.weworkQrModalLoading = true;
  host.weworkQrModalPolling = false;
  host.weworkQrModalAuthUrl = null;
  host.weworkQrModalGenPageUrl = null;
  try {
    await loadConfig(host);
    const res = await host.client.request<{
      scode?: string;
      authUrl?: string;
      genPageUrl?: string;
    }>("channels.wework.qr.start", { timeoutMs: 60000 });
    const scode = res.scode?.trim() ?? "";
    if (!scode) {
      host.weworkQrModalLoading = false;
      host.weworkQrModalError = t("channelWeWorkQrStartFailed");
      return;
    }
    host.weworkQrModalLoading = false;
    host.weworkQrModalAuthUrl = res.authUrl ?? null;
    host.weworkQrModalGenPageUrl = res.genPageUrl ?? null;
    host.weworkQrModalPolling = true;
    host.weworkQrPollTimer = window.setInterval(() => {
      void pollWeWorkQrOnce(host, scode);
    }, 2800);
  } catch (err) {
    host.weworkQrModalLoading = false;
    host.weworkQrModalPolling = false;
    host.weworkQrModalError = String(err);
  }
}

export function handleWeWorkQrModalClose(host: OpenClawApp) {
  resetWeWorkQrModal(host);
}

function hasWeixinConfigured(host: OpenClawApp): boolean {
  const channels = host.configForm?.channels as Record<string, unknown> | undefined;
  const weixin = channels?.weixin as Record<string, unknown> | undefined;
  const creds = weixin?.credentials as Record<string, unknown> | undefined;
  const token = typeof creds?.botToken === "string" ? creds.botToken.trim() : "";
  const botId = typeof creds?.botId === "string" ? creds.botId.trim() : "";
  if (token !== "" && botId !== "") {
    return true;
  }
  const snap = host.channelsSnapshot?.channels?.weixin as { configured?: boolean } | undefined;
  if (snap?.configured) {
    return true;
  }
  const acc = host.channelsSnapshot?.channelAccounts?.weixin?.[0];
  if (acc?.configured) {
    return true;
  }
  return false;
}

function clearWeixinQrSuccessTimer(host: OpenClawApp) {
  const id = host.weixinQrSuccessCloseTimer;
  if (id != null) {
    window.clearTimeout(id);
    host.weixinQrSuccessCloseTimer = null;
  }
}

function resetWeixinQrModal(host: OpenClawApp) {
  host.weixinQrPollAbort = true;
  clearWeixinQrSuccessTimer(host);
  host.weixinQrModalOpen = false;
  host.weixinQrModalLoading = false;
  host.weixinQrModalPolling = false;
  host.weixinQrModalSuccess = false;
  host.weixinQrModalError = null;
  host.weixinQrModalReplaceWarn = false;
  host.weixinQrModalImageSrc = null;
  host.weixinQrModalScanPageUrl = null;
  host.weixinQrModalScanned = false;
  host.weixinQrSessionQrcode = "";
  host.weixinQrSessionBaseUrl = "";
  host.weixinQrSessionBotType = "";
}

async function runWeixinQrPollLoop(host: OpenClawApp) {
  const qrcode = host.weixinQrSessionQrcode.trim();
  if (!qrcode || !host.client || !host.connected) {
    host.weixinQrModalPolling = false;
    return;
  }
  while (
    !host.weixinQrPollAbort &&
    host.weixinQrModalOpen &&
    host.weixinQrModalPolling &&
    host.client &&
    host.connected
  ) {
    try {
      const res = await host.client.request<{
        status?: string;
        botToken?: string;
        botId?: string;
        baseUrl?: string;
        userId?: string;
      }>("channels.weixin.qr.poll", {
        qrcode,
        baseUrl: host.weixinQrSessionBaseUrl || undefined,
        botType: host.weixinQrSessionBotType || undefined,
        timeoutMs: 45000,
      });
      const st = (res.status ?? "").trim();
      if (st === "scaned") {
        host.weixinQrModalScanned = true;
      }
      if (st === "expired") {
        host.weixinQrModalError = t("channelWeixinQrExpired");
        host.weixinQrModalPolling = false;
        return;
      }
      if (st === "confirmed" && res.botToken && res.botId) {
        updateConfigFormValue(host, ["channels", "weixin", "credentials", "botToken"], res.botToken);
        updateConfigFormValue(host, ["channels", "weixin", "credentials", "botId"], res.botId);
        if (res.baseUrl) {
          updateConfigFormValue(host, ["channels", "weixin", "credentials", "baseUrl"], res.baseUrl);
        }
        if (res.userId) {
          updateConfigFormValue(host, ["channels", "weixin", "credentials", "userId"], res.userId);
        }
        updateConfigFormValue(host, ["channels", "weixin", "enabled"], true);

        const ch = host.configForm?.channels as Record<string, unknown> | undefined;
        const weixin = ch?.weixin;
        if (!weixin || typeof weixin !== "object") {
          host.weixinQrModalError = t("channelWeixinQrSaveMissingForm");
          host.weixinQrModalPolling = false;
          return;
        }

        host.lastError = null;
        await saveConfigPatch(host as ConfigState, {
          channels: { weixin: cloneConfigObject(weixin as Record<string, unknown>) },
        });
        if (host.lastError) {
          host.weixinQrModalError = host.lastError;
          host.weixinQrModalPolling = false;
          return;
        }

        const runtimeIssues = await probeChannelRuntimeErrors(host, ["weixin"]);
        if (runtimeIssues.length) {
          host.weixinQrModalError = runtimeIssues.map((x) => `${x.label}: ${x.message}`).join("\n");
          host.weixinQrModalPolling = false;
          return;
        }

        host.weixinQrModalPolling = false;
        host.weixinQrModalSuccess = true;
        host.weixinQrModalImageSrc = null;
        host.weixinQrModalScanPageUrl = null;
        host.weixinQrSuccessCloseTimer = window.setTimeout(() => {
          host.weixinQrSuccessCloseTimer = null;
          resetWeixinQrModal(host);
          void loadChannels(host, true);
        }, 1600);
        return;
      }
    } catch (err) {
      host.weixinQrModalError = String(err);
      host.weixinQrModalPolling = false;
      return;
    }
  }
  host.weixinQrModalPolling = false;
}

export async function handleWeixinQrStart(host: OpenClawApp) {
  if (!host.client || !host.connected) {
    return;
  }
  host.weixinQrPollAbort = true;
  clearWeixinQrSuccessTimer(host);
  host.weixinQrModalOpen = true;
  host.weixinQrPollAbort = false;
  host.weixinQrModalReplaceWarn = hasWeixinConfigured(host);
  host.weixinQrModalError = null;
  host.weixinQrModalSuccess = false;
  host.weixinQrModalLoading = true;
  host.weixinQrModalPolling = false;
  host.weixinQrModalImageSrc = null;
  host.weixinQrModalScanPageUrl = null;
  host.weixinQrModalScanned = false;
  host.weixinQrSessionQrcode = "";
  host.weixinQrSessionBaseUrl = "";
  host.weixinQrSessionBotType = "";
  try {
    await loadConfig(host);
    const res = await host.client.request<{
      qrcode?: string;
      qrImageContent?: string;
      baseUrl?: string;
      botType?: string;
    }>("channels.weixin.qr.start", { timeoutMs: 60000 });
    const qc = res.qrcode?.trim() ?? "";
    if (!qc) {
      host.weixinQrModalLoading = false;
      host.weixinQrModalError = t("channelWeixinQrStartFailed");
      return;
    }
    host.weixinQrSessionQrcode = qc;
    host.weixinQrSessionBaseUrl = (res.baseUrl ?? "").trim();
    host.weixinQrSessionBotType = (res.botType ?? "").trim();
    const { imgSrc, openPageUrl } = resolveWeixinQrDisplay(res.qrImageContent ?? "");
    host.weixinQrModalLoading = false;
    host.weixinQrModalImageSrc = imgSrc || null;
    host.weixinQrModalScanPageUrl = openPageUrl;
    host.weixinQrModalPolling = true;
    void runWeixinQrPollLoop(host);
  } catch (err) {
    host.weixinQrModalLoading = false;
    host.weixinQrModalPolling = false;
    host.weixinQrModalError = String(err);
  }
}

export function handleWeixinQrModalClose(host: OpenClawApp) {
  resetWeixinQrModal(host);
}

export async function handleWhatsAppStart(host: OpenClawApp, force: boolean) {
  await startWhatsAppLogin(host, force);
  await loadChannels(host, true);
}

export async function handleWhatsAppWait(host: OpenClawApp) {
  await waitWhatsAppLogin(host);
  await loadChannels(host, true);
}

export async function handleWhatsAppLogout(host: OpenClawApp) {
  await logoutWhatsApp(host);
  await loadChannels(host, true);
}

export async function handleChannelConfigSave(host: OpenClawApp) {
  const channels = host.configForm?.channels;
  const isChannelsPatch = channels != null && typeof channels === "object";
  if (isChannelsPatch && !(await nativeConfirm(t("channelsConfigSaveConfirm")))) {
    return;
  }
  if (isChannelsPatch) {
    await saveConfigPatch(host, { channels });
  } else {
    await saveConfig(host);
  }
  await loadConfig(host);
  await loadChannels(host, true);
  if (!host.lastError) {
    const ch = host.configForm?.channels as Record<string, unknown> | undefined;
    const sel = host.channelsSelectedChannelId?.trim().toLowerCase() || "";
    const scope = isChannelsPatch && sel ? { onlyChannelIds: [sel] } : undefined;
    await alertChannelRuntimeErrorsIfAny(host, ch, scope);
  }
}

export async function handleChannelConfigReload(host: OpenClawApp) {
  await loadConfig(host);
  await loadChannels(host, true);
}

function parseValidationErrors(details: unknown): Record<string, string> {
  if (!Array.isArray(details)) {
    return {};
  }
  const errors: Record<string, string> = {};
  for (const entry of details) {
    if (typeof entry !== "string") {
      continue;
    }
    const [rawField, ...rest] = entry.split(":");
    if (!rawField || rest.length === 0) {
      continue;
    }
    const field = rawField.trim();
    const message = rest.join(":").trim();
    if (field && message) {
      errors[field] = message;
    }
  }
  return errors;
}

function resolveNostrAccountId(host: OpenClawApp): string {
  const accounts = host.channelsSnapshot?.channelAccounts?.nostr ?? [];
  return accounts[0]?.accountId ?? host.nostrProfileAccountId ?? "default";
}

function buildNostrProfileUrl(accountId: string, suffix = ""): string {
  return `/api/channels/nostr/${encodeURIComponent(accountId)}/profile${suffix}`;
}

export function handleNostrProfileEdit(
  host: OpenClawApp,
  accountId: string,
  profile: NostrProfile | null,
) {
  host.nostrProfileAccountId = accountId;
  host.nostrProfileFormState = createNostrProfileFormState(profile ?? undefined);
}

export function handleNostrProfileCancel(host: OpenClawApp) {
  host.nostrProfileFormState = null;
  host.nostrProfileAccountId = null;
}

export function handleNostrProfileFieldChange(
  host: OpenClawApp,
  field: keyof NostrProfile,
  value: string,
) {
  const state = host.nostrProfileFormState;
  if (!state) {
    return;
  }
  host.nostrProfileFormState = {
    ...state,
    values: {
      ...state.values,
      [field]: value,
    },
    fieldErrors: {
      ...state.fieldErrors,
      [field]: "",
    },
  };
}

export function handleNostrProfileToggleAdvanced(host: OpenClawApp) {
  const state = host.nostrProfileFormState;
  if (!state) {
    return;
  }
  host.nostrProfileFormState = {
    ...state,
    showAdvanced: !state.showAdvanced,
  };
}

export async function handleNostrProfileSave(host: OpenClawApp) {
  const state = host.nostrProfileFormState;
  if (!state || state.saving) {
    return;
  }
  const accountId = resolveNostrAccountId(host);

  host.nostrProfileFormState = {
    ...state,
    saving: true,
    error: null,
    success: null,
    fieldErrors: {},
  };

  try {
    const response = await fetch(buildNostrProfileUrl(accountId), {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(state.values),
    });
    const data = (await response.json().catch(() => null)) as {
      ok?: boolean;
      error?: string;
      details?: unknown;
      persisted?: boolean;
    } | null;

    if (!response.ok || data?.ok === false || !data) {
      const errorMessage = data?.error ?? `Profile update failed (${response.status})`;
      host.nostrProfileFormState = {
        ...state,
        saving: false,
        error: errorMessage,
        success: null,
        fieldErrors: parseValidationErrors(data?.details),
      };
      return;
    }

    if (!data.persisted) {
      host.nostrProfileFormState = {
        ...state,
        saving: false,
        error: "Profile publish failed on all relays.",
        success: null,
      };
      return;
    }

    host.nostrProfileFormState = {
      ...state,
      saving: false,
      error: null,
      success: "Profile published to relays.",
      fieldErrors: {},
      original: { ...state.values },
    };
    await loadChannels(host, true);
  } catch (err) {
    host.nostrProfileFormState = {
      ...state,
      saving: false,
      error: `Profile update failed: ${String(err)}`,
      success: null,
    };
  }
}

export async function handleNostrProfileImport(host: OpenClawApp) {
  const state = host.nostrProfileFormState;
  if (!state || state.importing) {
    return;
  }
  const accountId = resolveNostrAccountId(host);

  host.nostrProfileFormState = {
    ...state,
    importing: true,
    error: null,
    success: null,
  };

  try {
    const response = await fetch(buildNostrProfileUrl(accountId, "/import"), {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ autoMerge: true }),
    });
    const data = (await response.json().catch(() => null)) as {
      ok?: boolean;
      error?: string;
      imported?: NostrProfile;
      merged?: NostrProfile;
      saved?: boolean;
    } | null;

    if (!response.ok || data?.ok === false || !data) {
      const errorMessage = data?.error ?? `Profile import failed (${response.status})`;
      host.nostrProfileFormState = {
        ...state,
        importing: false,
        error: errorMessage,
        success: null,
      };
      return;
    }

    const merged = data.merged ?? data.imported ?? null;
    const nextValues = merged ? { ...state.values, ...merged } : state.values;
    const showAdvanced = Boolean(
      nextValues.banner || nextValues.website || nextValues.nip05 || nextValues.lud16,
    );

    host.nostrProfileFormState = {
      ...state,
      importing: false,
      values: nextValues,
      error: null,
      success: data.saved
        ? "Profile imported from relays. Review and publish."
        : "Profile imported. Review and publish.",
      showAdvanced,
    };

    if (data.saved) {
      await loadChannels(host, true);
    }
  } catch (err) {
    host.nostrProfileFormState = {
      ...state,
      importing: false,
      error: `Profile import failed: ${String(err)}`,
      success: null,
    };
  }
}

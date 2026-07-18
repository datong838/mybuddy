import type { AppViewState } from "./app-view-state.ts";
import {
  fetchAppUpdateCheck,
  fetchAppUpdateInstallStatus,
  skipAppUpdateVersion,
  startAppUpdateInstall,
  type AppUpdateCheckResult,
} from "./controllers/app-update.ts";
import { isLocalGatewayHost } from "./gateway-url.ts";
import { nativeAlert } from "./native-dialog-bridge.ts";
import { loadConfig } from "./controllers/config.ts";
import { isDesktopShell } from "./open-external-url.ts";

let installPollTimer: number | null = null;

type UpdateHost = AppViewState;

export function isAppUpdateUIAvailable(state: AppViewState): boolean {
  if (!state.connected) {
    return false;
  }
  if (isDesktopShell()) {
    return true;
  }
  return isLocalGatewayHost(state.settings.gatewayUrl);
}

function gatewayOpts(state: AppViewState) {
  return {
    gatewayHost: state.settings.gatewayUrl,
    token: state.settings.token,
  };
}

function applyUpdateInfo(state: UpdateHost, data: AppUpdateCheckResult | undefined) {
  state.appUpdateInfo = data ?? null;
  state.appUpdateInstallProgress = data?.progress ?? null;
  if (data?.installError) {
    state.appUpdateError = data.installError;
  }
  if (data?.manualInstallHint) {
    state.appUpdateManualHint = data.manualInstallHint;
  }
}

export async function maybeRunDailyAppUpdateCheck(state: UpdateHost) {
  if (!isAppUpdateUIAvailable(state)) {
    return;
  }

  state.appUpdateCheckLoading = true;
  state.appUpdateError = null;
  try {
    const res = await fetchAppUpdateCheck({
      ...gatewayOpts(state),
      dailyOnly: true,
      record: true,
    });
    if (!res.ok || !res.data) {
      return;
    }
    applyUpdateInfo(state, res.data);
    state.appUpdateManualHint = res.data.manualInstallHint ?? null;
    if (res.data.skippedDaily) {
      return;
    }
    if (res.data.hasUpdate && !res.data.skipped) {
      state.appUpdateModalOpen = true;
    }
  } finally {
    state.appUpdateCheckLoading = false;
  }
}

export async function handleManualAppUpdateCheck(state: UpdateHost) {
  if (!isAppUpdateUIAvailable(state)) {
    await nativeAlert("检查更新仅在本机桌面应用或连接本机 Gateway 时可用。");
    return;
  }
  state.appUpdateCheckLoading = true;
  state.appUpdateError = null;
  try {
    const res = await fetchAppUpdateCheck({
      ...gatewayOpts(state),
      force: true,
      record: true,
    });
    if (!res.ok) {
      await nativeAlert(res.error ?? "检查更新失败");
      return;
    }
    applyUpdateInfo(state, res.data);
    state.appUpdateManualHint = res.data?.manualInstallHint ?? null;
    if (!res.data?.hasUpdate) {
      const current = res.data?.currentVersion ?? "当前版本";
      await nativeAlert(`已是最新版本（${current}）。`);
      return;
    }
    state.appUpdateModalOpen = true;
  } finally {
    state.appUpdateCheckLoading = false;
  }
}

export async function handleAppUpdateSkip(state: UpdateHost) {
  const version = state.appUpdateInfo?.latestVersion?.trim();
  if (!version) {
    state.appUpdateModalOpen = false;
    return;
  }
  state.appUpdateError = null;
  const res = await skipAppUpdateVersion({ ...gatewayOpts(state), version });
  if (!res.ok) {
    state.appUpdateError = res.error ?? "跳过版本失败";
    return;
  }
  state.appUpdateModalOpen = false;
  await loadConfig(state);
}

export function closeAppUpdateModal(state: UpdateHost) {
  state.appUpdateModalOpen = false;
}

export async function handleAppUpdateInstall(state: UpdateHost) {
  const version = state.appUpdateInfo?.latestVersion?.trim();
  if (!version) {
    state.appUpdateError = "无法确定要安装的版本";
    return;
  }
  if (!state.appUpdateInfo?.autoInstallSupported) {
    state.appUpdateError = "当前环境不支持自动安装，请使用下方手动命令或下载链接。";
    return;
  }
  state.appUpdateInstalling = true;
  state.appUpdateError = null;
  try {
    const res = await startAppUpdateInstall({ ...gatewayOpts(state), version });
    if (!res.ok) {
      state.appUpdateError = res.error ?? "启动更新失败";
      if (state.appUpdateInfo?.manualInstallHint) {
        state.appUpdateManualHint = state.appUpdateInfo.manualInstallHint;
      }
      return;
    }
    applyUpdateInfo(state, res.data);
    if (res.data?.manualInstallHint) {
      state.appUpdateManualHint = res.data.manualInstallHint;
    }
    startAppUpdateInstallPolling(state);
  } finally {
    state.appUpdateInstalling = false;
  }
}

export async function copyAppUpdateManualHint(state: UpdateHost) {
  const hint = state.appUpdateManualHint ?? state.appUpdateInfo?.manualInstallHint;
  if (!hint?.trim()) {
    await nativeAlert("暂无手动安装说明。");
    return;
  }
  try {
    await navigator.clipboard.writeText(hint);
    await nativeAlert("手动安装命令已复制到剪贴板。");
  } catch {
    await nativeAlert(hint);
  }
}

export function stopAppUpdateInstallPolling(_state: UpdateHost) {
  if (installPollTimer != null) {
    window.clearInterval(installPollTimer);
    installPollTimer = null;
  }
}

export function startAppUpdateInstallPolling(state: UpdateHost) {
  stopAppUpdateInstallPolling(state);
  installPollTimer = window.setInterval(() => {
    void pollAppUpdateInstallStatus(state);
  }, 1500);
  void pollAppUpdateInstallStatus(state);
}

async function pollAppUpdateInstallStatus(state: UpdateHost) {
  const res = await fetchAppUpdateInstallStatus(gatewayOpts(state));
  if (!res.ok || !res.data) {
    return;
  }
  applyUpdateInfo(state, res.data);
  if (res.data.manualInstallHint) {
    state.appUpdateManualHint = res.data.manualInstallHint;
  }
  if (res.data.installing) {
    return;
  }
  stopAppUpdateInstallPolling(state);
  if (res.data.installError) {
    state.appUpdateError = res.data.installError;
  }
}

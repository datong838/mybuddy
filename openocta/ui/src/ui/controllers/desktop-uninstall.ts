import { gatewayHttpBase } from "../gateway-url.ts";

export type DesktopUninstallMode = "program" | "full";

export type DesktopUninstallResult = {
  ok: boolean;
  message?: string;
  detail?: string;
  httpStatus?: number;
};

/**
 * POST /api/desktop/uninstall — 安排延迟卸载（需网关 Token；仅桌面模式或 OPENOCTA_ALLOW_UNINSTALL=1）。
 */
export async function requestDesktopUninstall(opts: {
  gatewayHost: string;
  token: string;
  mode: DesktopUninstallMode;
}): Promise<DesktopUninstallResult> {
  const base = gatewayHttpBase(opts.gatewayHost.trim());
  if (!base) {
    return { ok: false, detail: "未配置网关地址（Gateway URL）" };
  }
  const url = `${base.replace(/\/$/, "")}/api/desktop/uninstall`;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  const tok = (opts.token ?? "").trim();
  if (tok) {
    headers.Authorization = `Bearer ${tok}`;
    headers["X-Gateway-Token"] = tok;
  }
  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify({ mode: opts.mode }),
    });
  } catch (e) {
    return { ok: false, detail: e instanceof Error ? e.message : String(e) };
  }
  let data: { ok?: boolean; message?: string; detail?: string; error?: string; code?: string } = {};
  try {
    data = (await res.json()) as typeof data;
  } catch {
    // ignore
  }
  if (!res.ok) {
    const msg =
      data.message ??
      data.error ??
      (res.status === 401 ? "网关令牌无效或未提供" : `请求失败（HTTP ${res.status}）`);
    return {
      ok: false,
      message: msg,
      detail: data.detail,
      httpStatus: res.status,
    };
  }
  if (data.ok === false) {
    return {
      ok: false,
      message: data.message,
      detail: data.detail,
      httpStatus: res.status,
    };
  }
  return {
    ok: true,
    message: data.message,
    detail: data.detail,
  };
}

/**
 * POST /api/desktop/clear-workspace — 清空默认工作区目录（状态目录下的 workspace），需网关 Token；
 * 仅桌面模式或 OPENOCTA_ALLOW_UNINSTALL=1 时允许。
 */
export async function requestDesktopClearWorkspace(opts: {
  gatewayHost: string;
  token: string;
}): Promise<DesktopUninstallResult> {
  const base = gatewayHttpBase(opts.gatewayHost.trim());
  if (!base) {
    return { ok: false, detail: "未配置网关地址（Gateway URL）" };
  }
  const url = `${base.replace(/\/$/, "")}/api/desktop/clear-workspace`;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  const tok = (opts.token ?? "").trim();
  if (tok) {
    headers.Authorization = `Bearer ${tok}`;
    headers["X-Gateway-Token"] = tok;
  }
  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers,
      body: JSON.stringify({}),
    });
  } catch (e) {
    return { ok: false, detail: e instanceof Error ? e.message : String(e) };
  }
  let data: { ok?: boolean; message?: string; detail?: string; error?: string; code?: string } = {};
  try {
    data = (await res.json()) as typeof data;
  } catch {
    // ignore
  }
  if (!res.ok) {
    const msg =
      data.message ??
      data.error ??
      (res.status === 401 ? "网关令牌无效或未提供" : `请求失败（HTTP ${res.status}）`);
    return {
      ok: false,
      message: msg,
      detail: data.detail,
      httpStatus: res.status,
    };
  }
  if (data.ok === false) {
    return {
      ok: false,
      message: data.message,
      detail: data.detail,
      httpStatus: res.status,
    };
  }
  return {
    ok: true,
    message: data.message,
    detail: data.detail,
  };
}

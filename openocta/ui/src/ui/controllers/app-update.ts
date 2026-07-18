import { gatewayHttpBase } from "../gateway-url.ts";

export type AppUpdateProgress = {
  phase?: string;
  percent?: number;
  message?: string;
};

export type AppUpdateCheckResult = {
  ok?: boolean;
  message?: string;
  currentVersion?: string;
  latestVersion?: string;
  hasUpdate?: boolean;
  skipped?: boolean;
  downloadSupported?: boolean;
  downloadUrl?: string;
  autoInstallSupported?: boolean;
  installAllowed?: boolean;
  packageFormat?: string;
  downloadUrls?: Record<string, string>;
  manualInstallHint?: string;
  lastCheckAt?: string;
  skippedDaily?: boolean;
  desktopMode?: boolean;
  installing?: boolean;
  installDone?: boolean;
  installError?: string;
  progress?: AppUpdateProgress;
};

function authHeaders(token: string): Record<string, string> {
  const headers: Record<string, string> = { Accept: "application/json" };
  const tok = (token ?? "").trim();
  if (tok) {
    headers.Authorization = `Bearer ${tok}`;
    headers["X-Gateway-Token"] = tok;
  }
  return headers;
}

function apiBase(gatewayHost: string): string | null {
  const base = gatewayHttpBase(gatewayHost.trim());
  if (!base) {
    return null;
  }
  return base.replace(/\/$/, "");
}

export async function fetchAppUpdateCheck(opts: {
  gatewayHost: string;
  token: string;
  force?: boolean;
  dailyOnly?: boolean;
  record?: boolean;
}): Promise<{ ok: boolean; data?: AppUpdateCheckResult; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  const params = new URLSearchParams();
  if (opts.force) {
    params.set("force", "1");
  }
  if (opts.dailyOnly) {
    params.set("dailyOnly", "1");
  }
  if (opts.record !== false) {
    params.set("record", "1");
  }
  const qs = params.toString();
  const url = `${base}/api/desktop/update/check${qs ? `?${qs}` : ""}`;
  let res: Response;
  try {
    res = await fetch(url, { method: "GET", headers: authHeaders(opts.token) });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: AppUpdateCheckResult = {};
  try {
    data = (await res.json()) as AppUpdateCheckResult;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: data.message ?? `请求失败（HTTP ${res.status}）`, data };
  }
  return { ok: true, data };
}

export async function skipAppUpdateVersion(opts: {
  gatewayHost: string;
  token: string;
  version: string;
}): Promise<{ ok: boolean; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/desktop/update/skip`, {
      method: "POST",
      headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
      body: JSON.stringify({ version: opts.version }),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: { ok?: boolean; message?: string } = {};
  try {
    data = (await res.json()) as typeof data;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: data.message ?? `请求失败（HTTP ${res.status}）` };
  }
  return { ok: true };
}

export async function startAppUpdateInstall(opts: {
  gatewayHost: string;
  token: string;
  version?: string;
}): Promise<{ ok: boolean; data?: AppUpdateCheckResult; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/desktop/update/install`, {
      method: "POST",
      headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
      body: JSON.stringify({ version: opts.version ?? "" }),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: AppUpdateCheckResult & { started?: boolean; status?: AppUpdateCheckResult } = {};
  try {
    data = (await res.json()) as typeof data;
  } catch {
    // ignore
  }
  const status = data.status ?? data;
  if (!res.ok || data.ok === false) {
    return { ok: false, error: data.message ?? status.message ?? `请求失败（HTTP ${res.status}）`, data: status };
  }
  return { ok: true, data: status };
}

export async function fetchAppUpdateInstallStatus(opts: {
  gatewayHost: string;
  token: string;
}): Promise<{ ok: boolean; data?: AppUpdateCheckResult; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/desktop/update/status`, {
      method: "GET",
      headers: authHeaders(opts.token),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: AppUpdateCheckResult = {};
  try {
    data = (await res.json()) as AppUpdateCheckResult;
  } catch {
    // ignore
  }
  if (!res.ok) {
    return { ok: false, error: data.message ?? `请求失败（HTTP ${res.status}）`, data };
  }
  return { ok: true, data };
}

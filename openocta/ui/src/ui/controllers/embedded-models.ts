import { gatewayHttpBase } from "../gateway-url.ts";
import type {
  EmbeddedRecommendationsResult,
  LocalHardwareProfile,
} from "./model-recommendation.ts";

export type EmbeddedModelProgress = {
  phase?: string;
  percent?: number;
  message?: string;
  bytesDone?: number;
  bytesTotal?: number;
};

export type EmbeddedModelKind = "chat" | "embedding";

export type EmbeddedModelFile = {
  name: string;
  url: string;
  size?: number;
};

export type EmbeddedModelEntry = {
  id: string;
  kind?: EmbeddedModelKind;
  kindLabel?: string;
  name: string;
  description: string;
  tags?: string[];
  builtin?: boolean;
  multimodal?: boolean;
  files?: EmbeddedModelFile[];
  installed?: boolean;
  running?: boolean;
  port?: number;
  endpoint?: string;
  lastError?: string;
  installedAt?: string;
  paramsB?: number;
  activeParamsB?: number;
  contextLength?: number;
  capabilities?: string[];
  provider?: string;
  license?: string;
  releasedAt?: string;
  architecture?: string;
  quantization?: string;
  toolCalling?: boolean;
  ollamaName?: string;
  vramGbQ4?: number;
  diskGbQ4?: number;
  downloadable?: boolean;
  featured?: boolean;
  hfUrl?: string;
  /** True when discovered from ~/.openocta/embedded-models scan (manual import). */
  sideloaded?: boolean;
};

export type EmbeddedModelsCatalogResult = {
  ok?: boolean;
  message?: string;
  models?: EmbeddedModelEntry[];
  runtime?: Record<string, unknown>;
  download?: EmbeddedDownloadStatus;
  provider?: Record<string, unknown>;
};

export type EmbeddedDownloadStatus = {
  ok?: boolean;
  downloading?: boolean;
  done?: boolean;
  modelId?: string;
  error?: string;
  progress?: EmbeddedModelProgress;
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

export async function fetchEmbeddedModelsCatalog(opts: {
  gatewayHost: string;
  token: string;
}): Promise<{ ok: boolean; data?: EmbeddedModelsCatalogResult; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/catalog`, {
      method: "GET",
      headers: authHeaders(opts.token),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: EmbeddedModelsCatalogResult = {};
  try {
    data = (await res.json()) as EmbeddedModelsCatalogResult;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: data.message ?? `请求失败（HTTP ${res.status}）`, data };
  }
  return { ok: true, data };
}

export async function startEmbeddedModelDownload(opts: {
  gatewayHost: string;
  token: string;
  modelId: string;
}): Promise<{ ok: boolean; data?: EmbeddedDownloadStatus; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/download`, {
      method: "POST",
      headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
      body: JSON.stringify({ modelId: opts.modelId }),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: { ok?: boolean; message?: string; download?: EmbeddedDownloadStatus } = {};
  try {
    data = (await res.json()) as typeof data;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: data.message ?? `请求失败（HTTP ${res.status}）`, data: data.download };
  }
  return { ok: true, data: data.download };
}

export async function fetchEmbeddedDownloadStatus(opts: {
  gatewayHost: string;
  token: string;
}): Promise<{ ok: boolean; data?: EmbeddedDownloadStatus; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/download/status`, {
      method: "GET",
      headers: authHeaders(opts.token),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: EmbeddedDownloadStatus = {};
  try {
    data = (await res.json()) as EmbeddedDownloadStatus;
  } catch {
    // ignore
  }
  return { ok: res.ok, data };
}

export async function cancelEmbeddedModelDownload(opts: {
  gatewayHost: string;
  token: string;
}): Promise<{ ok: boolean; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/download/cancel`, {
      method: "POST",
      headers: authHeaders(opts.token),
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

export async function startEmbeddedModelRuntime(opts: {
  gatewayHost: string;
  token: string;
  modelId: string;
}): Promise<{ ok: boolean; data?: Record<string, unknown>; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/start`, {
      method: "POST",
      headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
      body: JSON.stringify({ modelId: opts.modelId }),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: Record<string, unknown> = {};
  try {
    data = (await res.json()) as Record<string, unknown>;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: (data.message as string) ?? `请求失败（HTTP ${res.status}）`, data };
  }
  return { ok: true, data };
}

export async function stopEmbeddedModelRuntime(opts: {
  gatewayHost: string;
  token: string;
  modelId?: string;
}): Promise<{ ok: boolean; data?: Record<string, unknown>; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/stop`, {
      method: "POST",
      headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
      body: JSON.stringify({ modelId: opts.modelId ?? "" }),
    });
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: Record<string, unknown> = {};
  try {
    data = (await res.json()) as Record<string, unknown>;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: (data.message as string) ?? `请求失败（HTTP ${res.status}）` };
  }
  return { ok: true, data };
}

export async function deleteEmbeddedModel(opts: {
  gatewayHost: string;
  token: string;
  modelId: string;
}): Promise<{ ok: boolean; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    res = await fetch(`${base}/api/embedded-models/delete`, {
      method: "POST",
      headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
      body: JSON.stringify({ modelId: opts.modelId }),
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

export function embeddedModelKindLabel(kind?: EmbeddedModelKind, fallback?: string): string {
  if (fallback?.trim()) {
    return fallback.trim();
  }
  return kind === "embedding" ? "Embedding 向量" : "Chat 对话";
}

export function isEmbeddingModel(model: EmbeddedModelEntry): boolean {
  return model.kind === "embedding";
}

export function formatModelSize(bytes?: number): string {
  if (!bytes || bytes <= 0) {
    return "";
  }
  const gb = bytes / (1024 * 1024 * 1024);
  if (gb >= 1) {
    return `${gb.toFixed(1)} GB`;
  }
  const mb = bytes / (1024 * 1024);
  return `${Math.round(mb)} MB`;
}

export function totalModelSize(files?: EmbeddedModelFile[]): number {
  if (!files?.length) {
    return 0;
  }
  return files.reduce((sum, f) => sum + (f.size ?? 0), 0);
}

export async function fetchEmbeddedRecommendations(opts: {
  gatewayHost: string;
  token: string;
  /** When set, recalculates recommendations with manual hardware overrides. */
  hardwareOverride?: LocalHardwareProfile | null;
}): Promise<{ ok: boolean; data?: EmbeddedRecommendationsResult; error?: string }> {
  const base = apiBase(opts.gatewayHost);
  if (!base) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  let res: Response;
  try {
    if (opts.hardwareOverride) {
      const hw = opts.hardwareOverride;
      res = await fetch(`${base}/api/embedded-models/recommendations`, {
        method: "POST",
        headers: { ...authHeaders(opts.token), "Content-Type": "application/json" },
        body: JSON.stringify({
          gpuName: hw.gpuName,
          vramGb: hw.vramGb,
          ramGb: hw.ramGb,
          cpuCores: hw.cpuCores,
          bandwidthGbs: hw.bandwidthGbs,
          isAppleSilicon: hw.isAppleSilicon,
        }),
      });
    } else {
      res = await fetch(`${base}/api/embedded-models/recommendations`, {
        method: "GET",
        headers: authHeaders(opts.token),
      });
    }
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
  let data: EmbeddedRecommendationsResult = {};
  try {
    data = (await res.json()) as EmbeddedRecommendationsResult;
  } catch {
    // ignore
  }
  if (!res.ok || data.ok === false) {
    return { ok: false, error: data.message ?? `请求失败（HTTP ${res.status}）`, data };
  }
  return { ok: true, data };
}

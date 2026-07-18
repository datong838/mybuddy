import type { AppViewState } from "./app-view-state.ts";
import { getSecurityFromConfig, saveSecurityConfig, SECURITY_PRESETS } from "./controllers/security.ts";
import type { SecurityConfigForm } from "./controllers/security.ts";
import { loadConfig } from "./controllers/config.ts";
import { saveConfigPatch } from "./controllers/config.ts";
import { cloneConfigObject } from "./controllers/config/form-utils.ts";
import { setPathValue } from "./controllers/config/form-utils.ts";

/** Sync security form from config when entering tab or after config load. */
export function syncSecurityFromConfig(host: AppViewState): SecurityConfigForm | null {
  const current = getSecurityFromConfig(host);
  if (current == null) return null;
  return cloneConfigObject(current) as SecurityConfigForm;
}

export function handleSecurityPresetApply(host: AppViewState, preset: "off" | "loose" | "standard" | "strict") {
  if (!host.client || !host.connected) return;
  const p = SECURITY_PRESETS[preset];
  const base = cloneConfigObject(host.configForm ?? host.configSnapshot?.config ?? {}) as Record<string, unknown>;
  if (!base.security || typeof base.security !== "object") {
    base.security = {};
  }
  const security = base.security as Record<string, unknown>;
  Object.assign(security, p);
  // Ensure commandPolicy uses deny/ask/allow only (explicitly remove legacy rules)
  const cp = security.commandPolicy as Record<string, unknown> | undefined;
  if (cp && typeof cp === "object") {
    cp.rules = null;
  }
  host.securityForm = cloneConfigObject(p) as SecurityConfigForm;
  host.configSaving = true;
  (host as { lastError?: string | null }).lastError = null;
  saveConfigPatch(host, { security })
    .then(() => loadConfig(host))
    .finally(() => {
      host.configSaving = false;
    });
}

export function handleSecurityPatch(
  host: AppViewState,
  form: Record<string, unknown>,
  path: string[],
  value: unknown,
) {
  setPathValue(form, path, value);
  host.securityForm = cloneConfigObject(form) as Record<string, unknown>;
}

export async function handleSecuritySave(host: AppViewState, form: SecurityConfigForm | Record<string, unknown>) {
  const current = (form as SecurityConfigForm) ?? {};
  const normalized = cloneConfigObject(current) as SecurityConfigForm;
  const rl = (normalized.sandbox?.resourceLimit ?? {}) as {
    maxCpuPercent?: number;
    maxMemoryBytes?: number | string;
    maxDiskBytes?: number | string;
  };
  let errorMsg: string | null = null;
  if (typeof rl.maxMemoryBytes === "string") {
    const parsed = parseByteSize(rl.maxMemoryBytes);
    if (parsed == null && rl.maxMemoryBytes.trim() !== "") {
      errorMsg = "Invalid max memory format, use e.g. 1G, 512M, 1024";
    } else {
      rl.maxMemoryBytes = parsed ?? undefined;
    }
  }
  if (!errorMsg && typeof rl.maxDiskBytes === "string") {
    const parsed = parseByteSize(rl.maxDiskBytes);
    if (parsed == null && rl.maxDiskBytes.trim() !== "") {
      errorMsg = "Invalid max disk format, use e.g. 10G, 100G, 10240";
    } else {
      rl.maxDiskBytes = parsed ?? undefined;
    }
  }
  if (errorMsg) {
    (host as { lastError?: string | null }).lastError = errorMsg;
    return;
  }
  if (!rl.maxCpuPercent || rl.maxCpuPercent <= 0) {
    rl.maxCpuPercent = 60;
  }
  if (typeof rl.maxMemoryBytes !== "number" || rl.maxMemoryBytes <= 0) {
    rl.maxMemoryBytes = 1024 ** 3;
  }
  if (typeof rl.maxDiskBytes !== "number" || rl.maxDiskBytes <= 0) {
    rl.maxDiskBytes = 1024 ** 3;
  }
  if (!normalized.sandbox) normalized.sandbox = {};
  normalized.sandbox.resourceLimit = rl;

  await saveSecurityConfig(host, normalized);
  host.securityForm = syncSecurityFromConfig(host);
}

function parseByteSize(input: string): number | null {
  const trimmed = input.trim();
  if (!trimmed) return null;
  const match = trimmed.match(/^(\d+(?:\.\d+)?)(\s*)([kKmMgGtT]?[bB]?)?$/);
  if (!match) return null;
  const value = Number.parseFloat(match[1]);
  if (!Number.isFinite(value)) return null;
  const unitRaw = (match[3] ?? "").toUpperCase();
  let multiplier = 1;
  switch (unitRaw) {
    case "K":
    case "KB":
      multiplier = 1024;
      break;
    case "M":
    case "MB":
      multiplier = 1024 ** 2;
      break;
    case "G":
    case "GB":
      multiplier = 1024 ** 3;
      break;
    case "T":
    case "TB":
      multiplier = 1024 ** 4;
      break;
    default:
      multiplier = 1;
      break;
  }
  return Math.round(value * multiplier);
}

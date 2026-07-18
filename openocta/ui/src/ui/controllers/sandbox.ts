import type { ConfigState } from "./config.ts";
import { loadConfig, saveConfigPatch } from "./config.ts";

export type SandboxConfigForm = {
  enabled?: boolean;
  allowedPaths?: string[];
  networkAllow?: string[];
  root?: string;
  resourceLimit?: { maxCpuPercent?: number; maxMemoryBytes?: number | string; maxDiskBytes?: number | string };
  validator?: {
    banCommands?: string[];
    banArguments?: string[];
    banFragments?: string[];
    maxLength?: number;
    secretPatterns?: string[];
  };
  approvalStore?: string;
  approvalQueue?: {
    enabled?: boolean;
    timeoutSeconds?: number;
    blockOnApproval?: boolean;
    allow?: string[];
    ask?: string[];
    deny?: string[];
  };
};

/** Read sandbox config from current config form or snapshot. */
export function getSandboxFromConfig(state: ConfigState): SandboxConfigForm | null {
  const cfg = state.configForm ?? (state.configSnapshot?.config as Record<string, unknown> | undefined);
  if (!cfg || typeof cfg !== "object") return null;
  const security = (cfg.security ?? {}) as {
    sandbox?: SandboxConfigForm;
    validator?: SandboxConfigForm["validator"];
    approvalQueue?: SandboxConfigForm["approvalQueue"];
  };
  const sandbox = (security.sandbox ?? {}) as SandboxConfigForm;
  if (!sandbox || typeof sandbox !== "object") return null;
  // Merge validator / approvalQueue from top-level security.* if present.
  const merged: SandboxConfigForm = {
    ...sandbox,
    validator: security.validator ?? sandbox.validator,
    approvalQueue: security.approvalQueue ?? sandbox.approvalQueue,
  };
  return merged;
}

/** Save sandbox config via config.patch and reload config. */
export async function saveSandboxConfig(state: ConfigState, sandbox: SandboxConfigForm) {
  if (!state.client || !state.connected) return;
  state.configSaving = true;
  (state as { lastError?: string | null }).lastError = null;
  try {
    const approvalQueue = sandbox.approvalQueue
      ? {
          ...sandbox.approvalQueue,
          timeoutSeconds:
            sandbox.approvalQueue.timeoutSeconds == null
              ? 300
              : sandbox.approvalQueue.timeoutSeconds,
        }
      : undefined;

    const security = {
      sandbox: {
        enabled: sandbox.enabled,
        allowedPaths: sandbox.allowedPaths,
        networkAllow: sandbox.networkAllow,
        root: sandbox.root,
        resourceLimit: sandbox.resourceLimit,
        approvalStore: sandbox.approvalStore,
      },
      validator: sandbox.validator,
      approvalQueue,
    };
    await saveConfigPatch(state, { security });
    await loadConfig(state);
  } finally {
    state.configSaving = false;
  }
}

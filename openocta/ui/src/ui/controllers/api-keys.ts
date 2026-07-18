import type { GatewayBrowserClient } from "../gateway.ts";
import { DEFAULT_API_KEY_FORM } from "../app-defaults.ts";

export type ApiKeyEntry = {
  id: string;
  name: string;
  keyPrefix: string;
  allowedPaths: string[];
  bindingMode: "resources" | "employee";
  allowedModels?: string[];
  skillKeys?: string[];
  mcpServers?: string[];
  digitalEmployeeId?: string;
  monthlyTokenLimit?: number | null;
  monthlyTokensUsed?: number;
  usageMonth?: string;
  enabled: boolean;
  createdAt: number;
  updatedAt: number;
};

export type ApiKeyFormState = {
  editId: string;
  name: string;
  allowedPaths: string[];
  bindingMode: "resources" | "employee";
  allowedModels: string[];
  skillKeys: string[];
  mcpServers: string[];
  digitalEmployeeId: string;
  monthlyTokenLimit: string;
};

export type ApiKeySecretView = {
  id: string;
  name: string;
  secret: string;
};

export type ApiKeysState = {
  client: GatewayBrowserClient | null;
  connected: boolean;
  apiKeysLoading: boolean;
  apiKeys: ApiKeyEntry[];
  apiKeysError: string | null;
  apiKeysBusy: boolean;
  apiKeysForm: ApiKeyFormState;
  apiKeysFormModalOpen: boolean;
  apiKeysFormModalMode: "create" | "edit";
  apiKeysViewSecret: ApiKeySecretView | null;
  apiKeysCreatedSecret: string | null;
  apiKeysExamplesModalOpen: boolean;
};

export function formFromApiKeyEntry(entry: ApiKeyEntry): ApiKeyFormState {
  return {
    editId: entry.id,
    name: entry.name,
    allowedPaths: entry.allowedPaths?.length
      ? [...entry.allowedPaths]
      : ["/openocta/open/v1/ping", "/openocta/open/v1/completion"],
    bindingMode: entry.bindingMode === "employee" ? "employee" : "resources",
    allowedModels: [...(entry.allowedModels ?? [])],
    skillKeys: [...(entry.skillKeys ?? [])],
    mcpServers: [...(entry.mcpServers ?? [])],
    digitalEmployeeId: entry.digitalEmployeeId ?? "",
    monthlyTokenLimit:
      entry.monthlyTokenLimit != null && entry.monthlyTokenLimit > 0
        ? String(entry.monthlyTokenLimit)
        : "",
  };
}

export async function loadApiKeys(state: ApiKeysState) {
  if (!state.client || !state.connected) {
    return;
  }
  if (state.apiKeysLoading) {
    return;
  }
  state.apiKeysLoading = true;
  state.apiKeysError = null;
  try {
    const res = await state.client.request<{ keys?: ApiKeyEntry[] }>("apiKeys.list", {});
    state.apiKeys = Array.isArray(res.keys) ? res.keys : [];
  } catch (err) {
    state.apiKeysError = String(err);
  } finally {
    state.apiKeysLoading = false;
  }
}

export async function loadApiKeyDefaults(state: ApiKeysState): Promise<string[]> {
  if (!state.client || !state.connected) {
    return [];
  }
  try {
    const res = await state.client.request<{ allowedPaths?: string[] }>("apiKeys.defaults", {});
    return Array.isArray(res.allowedPaths) ? res.allowedPaths : [];
  } catch {
    return ["/openocta/open/v1/ping", "/openocta/open/v1/completion"];
  }
}

function buildFormPayload(form: ApiKeyFormState) {
  const limitRaw = form.monthlyTokenLimit.trim();
  let monthlyTokenLimit: number | undefined;
  if (limitRaw) {
    const n = Number(limitRaw);
    if (Number.isFinite(n) && n > 0) {
      monthlyTokenLimit = Math.floor(n);
    }
  }
  return {
    name: form.name.trim(),
    allowedPaths: form.allowedPaths.filter((p) => p.trim()),
    bindingMode: form.bindingMode,
    allowedModels: form.allowedModels,
    skillKeys: form.bindingMode === "resources" ? form.skillKeys : [],
    mcpServers: form.bindingMode === "resources" ? form.mcpServers : [],
    digitalEmployeeId: form.bindingMode === "employee" ? form.digitalEmployeeId.trim() : "",
    monthlyTokenLimit,
  };
}

export async function createApiKey(state: ApiKeysState) {
  if (!state.client || !state.connected) {
    return;
  }
  state.apiKeysBusy = true;
  state.apiKeysError = null;
  state.apiKeysCreatedSecret = null;
  try {
    const res = await state.client.request<{ entry?: ApiKeyEntry; secret?: string }>(
      "apiKeys.create",
      buildFormPayload(state.apiKeysForm),
    );
    if (res.secret) {
      state.apiKeysCreatedSecret = res.secret;
    }
    await loadApiKeys(state);
    state.apiKeysFormModalOpen = false;
  } catch (err) {
    state.apiKeysError = String(err);
  } finally {
    state.apiKeysBusy = false;
  }
}

export async function updateApiKey(state: ApiKeysState) {
  if (!state.client || !state.connected) {
    return;
  }
  const id = state.apiKeysForm.editId.trim();
  if (!id) {
    return;
  }
  state.apiKeysBusy = true;
  state.apiKeysError = null;
  try {
    await state.client.request("apiKeys.update", {
      id,
      ...buildFormPayload(state.apiKeysForm),
    });
    await loadApiKeys(state);
    state.apiKeysFormModalOpen = false;
  } catch (err) {
    state.apiKeysError = String(err);
  } finally {
    state.apiKeysBusy = false;
  }
}

export async function viewApiKeySecret(state: ApiKeysState, entry: ApiKeyEntry) {
  if (!state.client || !state.connected) {
    return;
  }
  state.apiKeysBusy = true;
  state.apiKeysError = null;
  try {
    const res = await state.client.request<{ id?: string; secret?: string }>("apiKeys.secret", {
      id: entry.id,
    });
    if (!res.secret) {
      throw new Error("未返回 API Key");
    }
    state.apiKeysViewSecret = {
      id: entry.id,
      name: entry.name,
      secret: res.secret,
    };
  } catch (err) {
    const message = String(err);
    if (message.includes("secret unavailable") || message.includes("unavailable")) {
      state.apiKeysError = "该 Key 无法读取（可能创建于旧版本），可点击「重新生成 Key」获取新密钥。";
    } else {
      state.apiKeysError = message;
    }
  } finally {
    state.apiKeysBusy = false;
  }
}

export async function regenerateApiKeySecret(state: ApiKeysState, entry: ApiKeyEntry) {
  if (!state.client || !state.connected) {
    return;
  }
  state.apiKeysBusy = true;
  state.apiKeysError = null;
  try {
    const res = await state.client.request<{ entry?: ApiKeyEntry; secret?: string }>(
      "apiKeys.regenerate",
      { id: entry.id },
    );
    if (!res.secret) {
      throw new Error("未返回 API Key");
    }
    state.apiKeysViewSecret = {
      id: entry.id,
      name: entry.name,
      secret: res.secret,
    };
    await loadApiKeys(state);
  } catch (err) {
    state.apiKeysError = String(err);
  } finally {
    state.apiKeysBusy = false;
  }
}

export async function removeApiKey(state: ApiKeysState, id: string) {
  if (!state.client || !state.connected) {
    return;
  }
  state.apiKeysBusy = true;
  state.apiKeysError = null;
  try {
    await state.client.request("apiKeys.remove", { id });
    if (state.apiKeysViewSecret?.id === id) {
      state.apiKeysViewSecret = null;
    }
    await loadApiKeys(state);
  } catch (err) {
    state.apiKeysError = String(err);
  } finally {
    state.apiKeysBusy = false;
  }
}

export async function toggleApiKey(state: ApiKeysState, entry: ApiKeyEntry, enabled: boolean) {
  if (!state.client || !state.connected) {
    return;
  }
  state.apiKeysBusy = true;
  state.apiKeysError = null;
  try {
    await state.client.request("apiKeys.update", { id: entry.id, enabled });
    await loadApiKeys(state);
  } catch (err) {
    state.apiKeysError = String(err);
  } finally {
    state.apiKeysBusy = false;
  }
}

export async function copyApiKeySecret(secret: string) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(secret);
    return;
  }
  const textarea = document.createElement("textarea");
  textarea.value = secret;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand("copy");
  document.body.removeChild(textarea);
}

export type ApiKeysPropsExtras = {
  gatewayBaseUrl: string;
  modelOptions: Array<{ value: string; label: string }>;
  skillOptions: import("../types.ts").SkillStatusEntry[];
  mcpOptions: Array<{ key: string; label: string }>;
  digitalEmployees: Array<{ id: string; name?: string; enabled?: boolean }>;
  digitalEmployeesLoading?: boolean;
  onRestoreDefaultPaths: () => void | Promise<void>;
};

export function buildApiKeysProps(
  host: ApiKeysState,
  extras: ApiKeysPropsExtras,
): import("../views/api-keys.ts").ApiKeysProps {
  return {
    loading: host.apiKeysLoading,
    keys: host.apiKeys,
    error: host.apiKeysError,
    busy: host.apiKeysBusy,
    connected: host.connected,
    form: host.apiKeysForm,
    formModalOpen: host.apiKeysFormModalOpen,
    formModalMode: host.apiKeysFormModalMode,
    viewSecret: host.apiKeysViewSecret,
    createdSecret: host.apiKeysCreatedSecret,
    examplesModalOpen: host.apiKeysExamplesModalOpen,
    gatewayBaseUrl: extras.gatewayBaseUrl,
    modelOptions: extras.modelOptions,
    skillOptions: extras.skillOptions,
    mcpOptions: extras.mcpOptions,
    digitalEmployees: extras.digitalEmployees,
    digitalEmployeesLoading: extras.digitalEmployeesLoading,
    onRefresh: () => loadApiKeys(host),
    onOpenCreateModal: () => {
      host.apiKeysForm = { ...DEFAULT_API_KEY_FORM };
      host.apiKeysError = null;
      host.apiKeysFormModalMode = "create";
      host.apiKeysFormModalOpen = true;
    },
    onOpenEditModal: (entry) => {
      host.apiKeysForm = formFromApiKeyEntry(entry);
      host.apiKeysError = null;
      host.apiKeysFormModalMode = "edit";
      host.apiKeysFormModalOpen = true;
    },
    onCloseFormModal: () => {
      host.apiKeysFormModalOpen = false;
    },
    onFormChange: (patch) => {
      host.apiKeysForm = { ...host.apiKeysForm, ...patch };
    },
    onRestoreDefaultPaths: () => void extras.onRestoreDefaultPaths(),
    onCreate: () => createApiKey(host),
    onUpdate: () => updateApiKey(host),
    onViewSecret: (entry) => viewApiKeySecret(host, entry),
    onRegenerateSecret: (entry) => regenerateApiKeySecret(host, entry),
    onCloseViewSecret: () => {
      host.apiKeysViewSecret = null;
    },
    onRemove: (entry) => removeApiKey(host, entry.id),
    onToggleEnabled: (entry, enabled) => toggleApiKey(host, entry, enabled),
    onDismissCreatedSecret: () => {
      host.apiKeysCreatedSecret = null;
    },
    onOpenExamplesModal: () => {
      host.apiKeysExamplesModalOpen = true;
    },
    onCloseExamplesModal: () => {
      host.apiKeysExamplesModalOpen = false;
    },
  };
}

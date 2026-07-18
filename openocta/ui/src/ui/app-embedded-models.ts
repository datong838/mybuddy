import type { AppViewState } from "./app-view-state.ts";
import {
  cancelEmbeddedModelDownload,
  deleteEmbeddedModel,
  fetchEmbeddedDownloadStatus,
  fetchEmbeddedModelsCatalog,
  fetchEmbeddedRecommendations,
  startEmbeddedModelDownload,
  startEmbeddedModelRuntime,
  stopEmbeddedModelRuntime,
  type EmbeddedDownloadStatus,
  type EmbeddedModelEntry,
  type EmbeddedModelProgress,
} from "./controllers/embedded-models.ts";
import {
  buildLocalPlazaModelDetail,
  type PlazaModelDetailInfo,
} from "./controllers/plaza-model-detail.ts";
import {
  detectLocalHardware,
  type LocalHardwareProfile,
  type ServerModelRecommendation,
} from "./controllers/model-recommendation.ts";
import { buildPlazaModelList, defaultPlazaModelList } from "./data/plaza-catalog.ts";
import { saveConfigPatch } from "./controllers/config.ts";
import {
  postEmbeddedChatCompletion,
  type PlazaChatMessage,
} from "./controllers/embedded-chat-test.ts";
import { nativeAlert, nativeConfirm } from "./native-dialog-bridge.ts";

let downloadPollTimer: number | null = null;
let hardwareDebounceTimer: number | null = null;

const HARDWARE_RECOMMEND_DEBOUNCE_MS = 400;

type Host = AppViewState;

function gatewayOpts(state: AppViewState) {
  return {
    gatewayHost: state.settings.gatewayUrl,
    token: state.settings.token,
  };
}

async function persistEmbeddedProviderPatch(state: Host, provider: Record<string, unknown> | undefined) {
  if (!state.configSnapshot?.hash) {
    return;
  }
  const existing =
    (state.configSnapshot.config?.models as { providers?: Record<string, unknown> })?.providers ?? {};
  const next = { ...existing };
  if (provider && Object.keys(provider).length > 0) {
    Object.assign(next, provider);
  } else {
    delete next["openocta-embedded-chat"];
    delete next["openocta-embedded-embedding"];
  }
  await saveConfigPatch(state, {
    models: {
      providers: next,
    },
  });
}

function syncEmbeddedPlazaChatModel(state: Host) {
  const open = state.embeddedPlazaChatModel;
  if (!open) {
    return;
  }
  const fresh = state.embeddedModels.find((m) => m.id === open.id);
  if (fresh) {
    state.embeddedPlazaChatModel = fresh;
    if (!fresh.running) {
      state.embeddedPlazaChatError = "当前模型已停止运行，请重新启动后再测试。";
      state.embeddedPlazaChatLoading = false;
    }
  }
}

async function loadEmbeddedRecommendations(state: Host, hardwareOverride?: LocalHardwareProfile | null) {
  if (!state.connected) {
    state.embeddedPlazaServerRecommendations = null;
    if (!state.embeddedPlazaHardware) {
      state.embeddedPlazaHardware = detectLocalHardware();
    }
    return;
  }
  state.embeddedPlazaRecommendationsLoading = true;
  try {
    const res = await fetchEmbeddedRecommendations({
      ...gatewayOpts(state),
      hardwareOverride: hardwareOverride ?? undefined,
    });
    if (!res.ok || !res.data) {
      state.embeddedPlazaServerRecommendations = null;
      if (!state.embeddedPlazaHardware) {
        state.embeddedPlazaHardware = detectLocalHardware();
      }
      return;
    }
    if (res.data.hardware) {
      state.embeddedPlazaHardware = { ...res.data.hardware, serverSide: true };
    }
    state.embeddedPlazaServerRecommendations = res.data.recommendations ?? null;
  } finally {
    state.embeddedPlazaRecommendationsLoading = false;
  }
}

export async function loadEmbeddedModels(state: Host) {
  state.embeddedModelsLoading = true;
  state.embeddedModelsError = null;
  try {
    if (!state.connected) {
      state.embeddedModels = defaultPlazaModelList();
      return;
    }
    const res = await fetchEmbeddedModelsCatalog(gatewayOpts(state));
    if (!res.ok) {
      state.embeddedModelsError = res.error ?? "加载模型广场失败";
      state.embeddedModels = defaultPlazaModelList();
      return;
    }
    state.embeddedModels = buildPlazaModelList(res.data?.models ?? []);
    syncEmbeddedPlazaChatModel(state);
    const provider = res.data?.provider as Record<string, unknown> | undefined;
    if (provider && Object.keys(provider).length > 0) {
      await persistEmbeddedProviderPatch(state, provider);
    }
    const dl = res.data?.download;
    if (dl?.downloading) {
      state.embeddedDownloadStatus = dl;
      startEmbeddedDownloadPolling(state);
    }
    await loadEmbeddedRecommendations(state);
  } finally {
    state.embeddedModelsLoading = false;
  }
}

function startEmbeddedDownloadPolling(state: Host) {
  stopEmbeddedDownloadPolling();
  downloadPollTimer = window.setInterval(() => {
    void pollEmbeddedDownload(state);
  }, 800);
}

export function stopEmbeddedDownloadPolling() {
  if (downloadPollTimer != null) {
    window.clearInterval(downloadPollTimer);
    downloadPollTimer = null;
  }
}

async function pollEmbeddedDownload(state: Host) {
  const res = await fetchEmbeddedDownloadStatus(gatewayOpts(state));
  if (!res.ok || !res.data) {
    return;
  }
  state.embeddedDownloadStatus = res.data;
  if (!res.data.downloading) {
    stopEmbeddedDownloadPolling();
    if (res.data.error) {
      state.embeddedModelsError = res.data.error;
    } else if (res.data.done) {
      await loadEmbeddedModels(state);
    }
  }
}

export async function handleEmbeddedModelDownload(state: Host, modelId: string) {
  const model = state.embeddedModels.find((m) => m.id === modelId);
  if (model?.downloadable === false) {
    const hint = model.ollamaName
      ? `此模型请通过 Ollama 安装：\n\nollama pull ${model.ollamaName}`
      : "此模型暂不支持内嵌下载，请通过 Ollama 安装。";
    await nativeAlert(hint);
    return;
  }
  state.embeddedModelsError = null;
  const res = await startEmbeddedModelDownload({ ...gatewayOpts(state), modelId });
  if (!res.ok) {
    await nativeAlert(res.error ?? "开始下载失败");
    return;
  }
  state.embeddedDownloadStatus = res.data ?? { downloading: true, modelId };
  startEmbeddedDownloadPolling(state);
}

export async function handleEmbeddedDownloadCancel(state: Host) {
  await cancelEmbeddedModelDownload(gatewayOpts(state));
  stopEmbeddedDownloadPolling();
  state.embeddedDownloadStatus = null;
  await loadEmbeddedModels(state);
}

export async function handleEmbeddedModelStart(state: Host, model: EmbeddedModelEntry) {
  state.embeddedModelsBusyId = model.id;
  state.embeddedModelsError = null;
  try {
    const res = await startEmbeddedModelRuntime({ ...gatewayOpts(state), modelId: model.id });
    if (!res.ok) {
      await nativeAlert(res.error ?? "启动失败");
      return;
    }
    const provider = res.data?.provider as Record<string, unknown> | undefined;
    await persistEmbeddedProviderPatch(state, provider);
    await loadEmbeddedModels(state);
  } finally {
    state.embeddedModelsBusyId = null;
  }
}

export async function handleEmbeddedModelStop(state: Host, modelId: string) {
  state.embeddedModelsBusyId = modelId;
  try {
    const res = await stopEmbeddedModelRuntime({ ...gatewayOpts(state), modelId });
    if (!res.ok) {
      await nativeAlert(res.error ?? "停止失败");
      return;
    }
    const provider = res.data?.provider as Record<string, unknown> | undefined;
    await persistEmbeddedProviderPatch(state, provider);
    await loadEmbeddedModels(state);
  } finally {
    state.embeddedModelsBusyId = null;
  }
}

export async function handleEmbeddedModelDelete(state: Host, model: EmbeddedModelEntry) {
  const ok = await nativeConfirm(`确定删除内嵌模型「${model.name}」？此操作不可恢复。`);
  if (!ok) {
    return;
  }
  state.embeddedModelsBusyId = model.id;
  try {
    const res = await deleteEmbeddedModel({ ...gatewayOpts(state), modelId: model.id });
    if (!res.ok) {
      await nativeAlert(res.error ?? "删除失败");
      return;
    }
    await loadEmbeddedModels(state);
  } finally {
    state.embeddedModelsBusyId = null;
  }
}

export function embeddedDownloadProgress(state: Host): EmbeddedModelProgress | null {
  return state.embeddedDownloadStatus?.progress ?? null;
}

export function isEmbeddedDownloading(state: Host, modelId: string): boolean {
  const st = state.embeddedDownloadStatus;
  return Boolean(st?.downloading && st.modelId === modelId);
}

export function installedEmbeddedModels(state: Host): EmbeddedModelEntry[] {
  return (state.embeddedModels ?? []).filter((m) => m.installed);
}

export function openEmbeddedPlazaRecommend(state: Host) {
  if (!state.connected && !state.embeddedPlazaHardware) {
    state.embeddedPlazaHardware = detectLocalHardware();
  }
  if (state.connected) {
    void loadEmbeddedRecommendations(state);
  }
  state.embeddedPlazaRecommendOpen = true;
}

export function closeEmbeddedPlazaRecommend(state: Host) {
  state.embeddedPlazaRecommendOpen = false;
}

export function openEmbeddedPlazaManualImport(state: Host) {
  state.embeddedPlazaManualImportOpen = true;
}

export function closeEmbeddedPlazaManualImport(state: Host) {
  state.embeddedPlazaManualImportOpen = false;
}

export function setEmbeddedPlazaHardware(state: Host, hw: LocalHardwareProfile) {
  state.embeddedPlazaHardware = hw;
  if (!state.connected) {
    return;
  }
  if (hardwareDebounceTimer != null) {
    window.clearTimeout(hardwareDebounceTimer);
  }
  hardwareDebounceTimer = window.setTimeout(() => {
    hardwareDebounceTimer = null;
    const current = state.embeddedPlazaHardware;
    if (current) {
      void loadEmbeddedRecommendations(state, current);
    }
  }, HARDWARE_RECOMMEND_DEBOUNCE_MS);
}

export function selectEmbeddedPlazaModel(state: Host, model: EmbeddedModelEntry | null) {
  state.embeddedPlazaDetailModel = model;
  state.embeddedPlazaDetailInfo = null;
  state.embeddedPlazaDetailError = null;
  state.embeddedPlazaDetailLoading = false;
  if (!model) {
    return;
  }
  if (!state.embeddedPlazaHardware && !state.connected) {
    state.embeddedPlazaHardware = detectLocalHardware();
  }
  state.embeddedPlazaDetailInfo = buildLocalPlazaModelDetail(model);
}

export function openEmbeddedPlazaChatTest(state: Host, model: EmbeddedModelEntry) {
  if (model.kind === "embedding") {
    void nativeAlert("向量模型不支持对话测试，请选择对话模型。");
    return;
  }
  if (!model.running) {
    void nativeAlert("请先启动模型后再测试对话。");
    return;
  }
  state.embeddedPlazaChatModel = model;
  state.embeddedPlazaChatMessages = [];
  state.embeddedPlazaChatInput = "";
  state.embeddedPlazaChatLoading = false;
  state.embeddedPlazaChatError = null;
}

export function closeEmbeddedPlazaChatTest(state: Host) {
  state.embeddedPlazaChatModel = null;
  state.embeddedPlazaChatMessages = [];
  state.embeddedPlazaChatInput = "";
  state.embeddedPlazaChatLoading = false;
  state.embeddedPlazaChatError = null;
}

export function setEmbeddedPlazaChatInput(state: Host, value: string) {
  state.embeddedPlazaChatInput = value;
}

export async function sendEmbeddedPlazaChatTest(state: Host) {
  let model = state.embeddedPlazaChatModel;
  if (!model || state.embeddedPlazaChatLoading) {
    return;
  }
  const fresh = state.embeddedModels.find((m) => m.id === model!.id);
  if (fresh) {
    model = fresh;
    state.embeddedPlazaChatModel = fresh;
  }
  if (!model.running) {
    state.embeddedPlazaChatError = "模型未运行，请先启动模型。";
    return;
  }
  const text = state.embeddedPlazaChatInput.trim();
  if (!text) {
    return;
  }

  const userMsg: PlazaChatMessage = { role: "user", content: text, modelId: model.id };
  const history = [...state.embeddedPlazaChatMessages, userMsg];
  state.embeddedPlazaChatMessages = history;
  state.embeddedPlazaChatInput = "";
  state.embeddedPlazaChatLoading = true;
  state.embeddedPlazaChatError = null;

  const res = await postEmbeddedChatCompletion({
    gatewayHost: state.settings.gatewayUrl,
    token: state.settings.token,
    model,
    messages: history,
  });
  state.embeddedPlazaChatLoading = false;
  if (!res.ok || !res.content) {
    state.embeddedPlazaChatError = res.error ?? "对话失败";
    return;
  }
  state.embeddedPlazaChatMessages = [...history, {
    role: "assistant",
    content: res.content ?? "",
    modelId: model.id,
    thinking: res.thinking ?? null,
  }];
}

export type { EmbeddedModelEntry, EmbeddedDownloadStatus, LocalHardwareProfile, PlazaChatMessage, PlazaModelDetailInfo, ServerModelRecommendation };

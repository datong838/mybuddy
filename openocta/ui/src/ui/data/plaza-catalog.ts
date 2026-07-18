import rawCatalog from "./plaza-catalog.json";
import type { EmbeddedModelEntry, EmbeddedModelKind } from "../controllers/embedded-models.ts";

/** One model row from CanIRun.ai-derived plaza catalog. */
export type PlazaCatalogItem = {
  id: string;
  kind: EmbeddedModelKind;
  name: string;
  description: string;
  provider: string;
  paramsB: number;
  activeParamsB?: number;
  contextLength: number;
  capabilities: string[];
  license: string;
  releasedAt: string;
  architecture: string;
  quantization: string;
  toolCalling: boolean;
  multimodal: boolean;
  ollamaName: string;
  hfUrl: string;
  featured: boolean;
  vramGbQ4: number;
  diskGbQ4: number;
  downloadable: boolean;
  builtin: boolean;
};

/** OpenOcta-only models not in CanIRun.ai static list. */
const EXTRA_CATALOG: PlazaCatalogItem[] = [
  {
    id: "qwen3-embedding-0.6b",
    kind: "embedding",
    name: "Qwen3-Embedding-0.6B",
    description: "通义千问 3 系列 0.6B 向量模型，支持 100+ 语言，适用于知识库检索与语义搜索。",
    provider: "Alibaba",
    paramsB: 0.6,
    contextLength: 8192,
    capabilities: ["embedding", "rag"],
    license: "Apache 2.0",
    releasedAt: "2025-04",
    architecture: "Dense",
    quantization: "Q8_0",
    toolCalling: false,
    multimodal: false,
    ollamaName: "qwen3-embedding:0.6b",
    hfUrl: "https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF",
    featured: true,
    vramGbQ4: 0.6,
    diskGbQ4: 0.6,
    downloadable: true,
    builtin: true,
  },
  {
    id: "qwen2.5-vl-3b",
    kind: "chat",
    name: "Qwen2.5-VL-3B",
    description: "通义千问 2.5 视觉语言对话模型，支持图像理解与多模态对话。",
    provider: "Alibaba",
    paramsB: 3,
    contextLength: 32768,
    capabilities: ["chat", "vision", "code", "toolCalling"],
    license: "Apache 2.0",
    releasedAt: "2024-11",
    architecture: "Dense",
    quantization: "Q4_K_M",
    toolCalling: true,
    multimodal: true,
    ollamaName: "qwen2.5vl:3b",
    hfUrl: "https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF",
    featured: false,
    vramGbQ4: 2.4,
    diskGbQ4: 2.4,
    downloadable: true,
    builtin: false,
  },
  {
    id: "smollm2-135m",
    kind: "chat",
    name: "SmolLM2-135M",
    description: "超轻量英文对话模型，体积极小，适合快速验证内嵌 Chat 推理链路。",
    provider: "HuggingFace",
    paramsB: 0.135,
    contextLength: 8192,
    capabilities: ["chat", "edge"],
    license: "Apache 2.0",
    releasedAt: "2024-09",
    architecture: "Dense",
    quantization: "Q4_K_M",
    toolCalling: false,
    multimodal: false,
    ollamaName: "smollm2:135m",
    hfUrl: "https://huggingface.co/QuantFactory/SmolLM2-135M-GGUF",
    featured: false,
    vramGbQ4: 0.1,
    diskGbQ4: 0.09,
    downloadable: true,
    builtin: false,
  },
];

function mergeCatalogItems(): PlazaCatalogItem[] {
  const base = rawCatalog as PlazaCatalogItem[];
  const byId = new Map(base.map((m) => [m.id, m]));
  for (const extra of EXTRA_CATALOG) {
    if (!byId.has(extra.id)) {
      byId.set(extra.id, extra);
    } else {
      byId.set(extra.id, { ...byId.get(extra.id)!, ...extra });
    }
  }
  const qwenChat = byId.get("qwen3-0.6b");
  if (qwenChat) {
    byId.set("qwen3-0.6b", {
      ...qwenChat,
      downloadable: true,
      builtin: true,
      featured: true,
      ollamaName: "qwen3:0.6b",
      toolCalling: true,
      capabilities: ["chat", "code", "reasoning", "edge"],
      description: "阿里通义千问 3 系列 0.6B 对话模型，中文理解优秀，适合本地轻量对话与工具调用。",
    });
  }
  return [...byId.values()];
}

export const PLAZA_CATALOG: PlazaCatalogItem[] = mergeCatalogItems();

export const PLAZA_CATALOG_COUNT = PLAZA_CATALOG.length;

function catalogItemToEntry(item: PlazaCatalogItem, backend?: EmbeddedModelEntry): EmbeddedModelEntry {
  const sizeBytes = backend?.files?.reduce((s, f) => s + (f.size ?? 0), 0) ?? 0;
  const estimatedBytes =
    sizeBytes > 0 ? sizeBytes : Math.round((item.diskGbQ4 || item.vramGbQ4 || 0) * 1024 * 1024 * 1024);
  const sideloadedInstalled = Boolean(backend?.sideloaded && backend?.installed);

  return {
    id: item.id,
    kind: backend?.kind ?? item.kind,
    kindLabel: (backend?.kind ?? item.kind) === "embedding" ? "Embedding 向量" : "Chat 对话",
    name: backend?.sideloaded && backend?.name ? backend.name : item.name,
    description: backend?.sideloaded && backend?.description ? backend.description : item.description,
    tags: [
      item.provider,
      item.featured ? "推荐" : "",
      item.builtin ? "内置" : "",
      backend?.sideloaded ? "手动导入" : "",
      (backend?.downloadable ?? item.downloadable) ? "" : "目录",
    ].filter(Boolean),
    builtin: item.builtin,
    multimodal: backend?.multimodal ?? item.multimodal,
    files: backend?.files?.length
      ? backend.files
      : estimatedBytes > 0
        ? [{ name: `${item.id}.gguf`, url: item.hfUrl, size: estimatedBytes }]
        : [],
    installed: backend?.installed ?? sideloadedInstalled,
    running: backend?.running ?? false,
    port: backend?.port,
    endpoint: backend?.endpoint,
    lastError: backend?.lastError,
    installedAt: backend?.installedAt,
    paramsB: item.paramsB,
    activeParamsB: item.activeParamsB,
    contextLength: backend?.contextLength ?? item.contextLength,
    capabilities: backend?.capabilities ?? item.capabilities,
    provider: backend?.provider ?? item.provider,
    license: backend?.license ?? item.license,
    releasedAt: item.releasedAt,
    architecture: item.architecture,
    quantization: backend?.quantization ?? item.quantization,
    toolCalling: item.toolCalling,
    ollamaName: item.ollamaName,
    vramGbQ4: item.vramGbQ4,
    diskGbQ4: item.diskGbQ4,
    downloadable: backend?.sideloaded ? false : (backend?.downloadable ?? item.downloadable),
    featured: item.featured,
    hfUrl: item.hfUrl,
    sideloaded: backend?.sideloaded,
  };
}

/** Merge CanIRun.ai catalog with gateway install/runtime status. */
export function buildPlazaModelList(backendModels: EmbeddedModelEntry[]): EmbeddedModelEntry[] {
  const backendById = new Map(backendModels.map((m) => [m.id, m]));
  const plazaIds = new Set(PLAZA_CATALOG.map((item) => item.id));
  const sideloadedOnly = backendModels
    .filter((m) => m.sideloaded && !plazaIds.has(m.id))
    .map((m) => ({
      ...m,
      kindLabel: m.kind === "embedding" ? "Embedding 向量" : "Chat 对话",
      downloadable: false,
      tags: [...(m.tags ?? []), "手动导入"],
    }));
  const plaza = PLAZA_CATALOG.map((item) => catalogItemToEntry(item, backendById.get(item.id)));
  return [...sideloadedOnly, ...plaza];
}

/** Catalog for offline / disconnected preview. */
export function defaultPlazaModelList(): EmbeddedModelEntry[] {
  return PLAZA_CATALOG.map((item) => catalogItemToEntry(item));
}

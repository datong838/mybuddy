#!/usr/bin/env node
/**
 * Generate plaza-catalog.json from CanIRun.ai model definitions.
 * Source: https://github.com/midudev/canirun.ai (packages/models/src/index.ts)
 */
import { writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const SOURCE_URL =
  "https://raw.githubusercontent.com/midudev/canirun.ai/main/packages/models/src/index.ts";

const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_UI_PATH = join(__dirname, "../src/ui/data/plaza-catalog.json");
const OUT_GO_PATH = join(__dirname, "../../src/pkg/embeddedmodels/plaza-catalog.json");

/** IDs with explicit multi-file overrides in Go backend. */
const DOWNLOADABLE_IDS = new Set([
  "qwen3-0.6b",
  "qwen3-embedding-0.6b",
  "qwen2.5-vl-3b",
  "smollm2-135m",
]);

function isGgufHfRepo(url) {
  const u = (url ?? "").trim().toLowerCase();
  return u.includes("huggingface.co") && u.includes("gguf");
}

function mapUseCaseToKind(useCase) {
  if (useCase.includes("embedding")) {
    return "embedding";
  }
  return "chat";
}

function mapCapabilities(useCase, model) {
  const caps = [...useCase];
  if (model.tools && !caps.includes("toolCalling")) {
    caps.push("toolCalling");
  }
  if (useCase.includes("vision") && !caps.includes("vision")) {
    caps.push("vision");
  }
  return caps;
}

/** OpenOcta-only models not present in CanIRun.ai STATIC_MODELS. */
const EXTRA_CATALOG = [
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

async function main() {
  const res = await fetch(SOURCE_URL);
  if (!res.ok) {
    throw new Error(`Failed to fetch CanIRun.ai models: HTTP ${res.status}`);
  }
  let src = await res.text();
  src = src.replace(/^export interface[\s\S]*?^}/gm, "");
  src = src.replace(/^import[\s\S]*?;\n/gm, "");
  src = src.replace(/^let scrapedModels[\s\S]*?;\n/gm, "");
  src = src.replace(/^export[\s\S]*$/gm, "");
  src = src.replace(/: AIModel\[\]/g, "");
  src = src.replace(/: Quantization\[\]/g, "");
  src = src.replace(/: Record<string, Record<string, number>>/g, "");
  src = src.replace(/const ggufSizes[\s\S]*?;\n/gm, "const ggufSizes = {};\n");
  src = src.replace(/function applyRealSizes[\s\S]*?^}/gm, "function applyRealSizes(_id, quants) { return quants; }");
  src = src.replace(/\((\w+): number\)/g, "($1)");
  src = src.replace(/: \{ min: number; rec: number \}/g, "");
  src = src.replace(/: string/g, "");
  src = src.replace(/: boolean/g, "");

  const fn = new Function(`${src}; return STATIC_MODELS;`);
  const models = fn();

  const catalog = models.map((m) => {
    const q4 = m.quants?.find((q) => q.name === "Q4_K_M") ?? m.quants?.[0];
    const vramGb = q4?.vramGB ?? q4?.diskGB ?? m.minRamGB;
    const diskGb = q4?.diskGB ?? vramGb;
    const useCase = m.useCase ?? [];
    const kind = mapUseCaseToKind(useCase);
    const id = m.id;
    const downloadable = DOWNLOADABLE_IDS.has(id) || isGgufHfRepo(m.url);

    return {
      id,
      kind,
      name: m.name,
      description: m.description,
      provider: m.provider,
      paramsB: m.paramsBillions,
      activeParamsB: m.moe?.activeParameters
        ? m.moe.activeParameters / 1_000_000_000
        : undefined,
      contextLength: m.contextLength,
      capabilities: mapCapabilities(useCase, m),
      license: m.license ?? "",
      releasedAt: m.releaseDate ?? "",
      architecture: m.architecture === "moe" ? "MoE" : "Dense",
      quantization: q4?.name ?? "Q4_K_M",
      toolCalling: Boolean(m.tools),
      multimodal: useCase.includes("vision"),
      ollamaName: m.ollamaId ?? "",
      hfUrl: m.url,
      featured: Boolean(m.featured),
      vramGbQ4: vramGb,
      diskGbQ4: diskGb,
      downloadable,
      builtin: id === "qwen3-0.6b" || id === "qwen3-embedding-0.6b",
    };
  });

  const byId = new Map(catalog.map((m) => [m.id, m]));
  for (const extra of EXTRA_CATALOG) {
    if (!byId.has(extra.id)) {
      byId.set(extra.id, extra);
    } else {
      byId.set(extra.id, { ...byId.get(extra.id), ...extra });
    }
  }
  const merged = [...byId.values()];

  const payload = `${JSON.stringify(merged, null, 2)}\n`;
  writeFileSync(OUT_UI_PATH, payload, "utf8");
  writeFileSync(OUT_GO_PATH, payload, "utf8");
  console.log(`Wrote ${merged.length} models to ${OUT_UI_PATH} and ${OUT_GO_PATH}`);
  console.log("Tip: run `node ui/scripts/resolve-plaza-gguf-links.mjs` to fill GGUF download URLs.");
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

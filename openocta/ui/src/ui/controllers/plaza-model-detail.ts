import type { EmbeddedModelEntry } from "./embedded-models.ts";
import { formatContextLength, formatParamsB } from "./model-recommendation.ts";

export type QuantizationOption = {
  quant: string;
  bits: number;
  vramGb: number;
  quality: string;
  qualityKey: "low" | "good" | "very-good" | "excellent" | "original";
};

export type PlazaModelDetailInfo = {
  repo?: string;
  description?: string;
  readme?: string;
  tags?: string[];
  downloads?: number;
  likes?: number;
  pipelineTag?: string;
  license?: string;
  local?: boolean;
};

/** 部分模型的中文详细说明（目录 description 较短时补充） */
const MODEL_ABOUT_OVERRIDES: Record<string, string> = {
  "qwen3-0.6b":
    "阿里通义千问 3 系列 0.6B 超轻量对话模型，中文理解优秀，适合本地轻量对话、工具调用与边缘设备部署。",
  "qwen3.5-9b":
    "Qwen 3.5 9B 是通义千问 3.5 系列中等规模模型，支持多模态与长上下文，在推理、代码与视觉理解方面表现均衡，是 16GB 统一内存设备的常见首选。",
  "qwen3-embedding-0.6b":
    "通义千问 3 系列 0.6B 向量模型，支持 100+ 语言，适用于知识库检索、语义搜索与 RAG 流水线。",
  "llama3.1-8b":
    "Meta Llama 3.1 8B 是广泛使用的开源对话基座，英文与多语言表现稳定，社区生态成熟，适合通用对话与微调。",
  "deepseek-r1":
    "DeepSeek R1 系列推理模型，强调链式思考与复杂问题求解，适合数学、逻辑与深度分析任务，对显存要求较高。",
};

const QUANT_SPECS: Array<{
  quant: string;
  bits: number;
  factor: number;
  quality: string;
  qualityKey: QuantizationOption["qualityKey"];
}> = [
  { quant: "Q2_K", bits: 2, factor: 0.35, quality: "较低", qualityKey: "low" },
  { quant: "Q4_K_M", bits: 4, factor: 0.55, quality: "良好", qualityKey: "good" },
  { quant: "Q6_K", bits: 6, factor: 0.8, quality: "很好", qualityKey: "very-good" },
  { quant: "Q8_0", bits: 8, factor: 1.05, quality: "极好", qualityKey: "excellent" },
  { quant: "F16", bits: 16, factor: 2.0, quality: "原始精度", qualityKey: "original" },
];

function buildGeneratedAbout(model: EmbeddedModelEntry): string {
  const params = formatParamsB(model.paramsB);
  const ctx = formatContextLength(model.contextLength);
  const caps = useCaseLabels(model);
  const parts: string[] = [];

  const head = [
    model.provider ? `${model.provider} 出品` : "",
    params ? `${params} 参数` : "",
    model.architecture ?? "",
  ]
    .filter(Boolean)
    .join(" · ");

  if (head) {
    parts.push(`${model.name} 是 ${head} 的${model.kind === "embedding" ? "向量" : "对话"}模型。`);
  }

  parts.push(
    `默认量化 ${model.quantization ?? "Q4_K_M"}，上下文 ${ctx}。` +
      (model.vramGbQ4 ? `Q4_K_M 约需 ${model.vramGbQ4} GB 显存。` : ""),
  );

  if (caps.length > 0) {
    parts.push(`适用场景：${caps.join("、")}。`);
  }
  if (model.toolCalling) {
    parts.push("支持工具调用，可用于 Agent 与插件扩展。");
  }
  if (model.multimodal) {
    parts.push("支持多模态输入，可处理图像与视觉理解任务。");
  }
  if (model.license) {
    parts.push(`开源许可：${model.license}。`);
  }

  return parts.join("\n\n");
}

/** 基于目录元数据生成本地模型说明，不依赖 HuggingFace。 */
export function buildLocalPlazaModelDetail(model: EmbeddedModelEntry): PlazaModelDetailInfo {
  const override = MODEL_ABOUT_OVERRIDES[model.id];
  const catalogDesc = (model.description ?? "").trim();
  const generated = buildGeneratedAbout(model);

  let description = override ?? catalogDesc;
  if (description && generated && description !== generated) {
    description = `${description}\n\n${generated}`;
  } else if (!description) {
    description = generated;
  }

  return {
    description,
    license: model.license,
    local: true,
  };
}

/** Estimate VRAM for common quant formats from parameter count or Q4 baseline. */
export function estimateQuantizationOptions(model: EmbeddedModelEntry): QuantizationOption[] {
  const paramsB = model.paramsB ?? 0;
  const q4Base = model.vramGbQ4 && model.vramGbQ4 > 0 ? model.vramGbQ4 : paramsB > 0 ? paramsB * 0.55 : 0;
  if (q4Base <= 0) {
    return [];
  }
  return QUANT_SPECS.map((spec) => ({
    quant: spec.quant,
    bits: spec.bits,
    vramGb: Math.round(q4Base * (spec.factor / 0.55) * 10) / 10,
    quality: spec.quality,
    qualityKey: spec.qualityKey,
  }));
}

export function useCaseLabels(model: EmbeddedModelEntry): string[] {
  const caps = new Set(model.capabilities ?? []);
  if (model.toolCalling) {
    caps.add("toolCalling");
  }
  if (model.multimodal) {
    caps.add("vision");
  }
  const labels: Record<string, string> = {
    chat: "对话",
    code: "代码",
    reasoning: "推理",
    vision: "视觉",
    embedding: "向量",
    rag: "检索增强",
    edge: "边缘设备",
    multilingual: "多语言",
    toolCalling: "工具调用",
  };
  return [...caps].map((c) => labels[c] ?? c);
}

export function formatCompactCount(n?: number): string {
  if (!n || n <= 0) {
    return "—";
  }
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`;
  }
  return String(n);
}

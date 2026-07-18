import type { EmbeddedModelEntry } from "./embedded-models.ts";
import { totalModelSize } from "./embedded-models.ts";

/** Hardware profile for model recommendation (inspired by CanIRun.ai). */
export type LocalHardwareProfile = {
  gpuName: string;
  vramGb: number;
  ramGb: number;
  cpuCores: number;
  bandwidthGbs: number;
  isAppleSilicon: boolean;
  detected: boolean;
  /** True when profile comes from Gateway server detection. */
  serverSide?: boolean;
};

export type ServerModelRecommendation = {
  modelId: string;
  score: number;
  tier: ModelTier;
  tierLabel: string;
  fitStatus: ModelFitStatus;
  vramGb: number;
  tokensPerSec: number;
  memoryPct: number;
};

export type EmbeddedRecommendationsResult = {
  ok?: boolean;
  message?: string;
  hardware?: LocalHardwareProfile;
  recommendations?: ServerModelRecommendation[];
};

export type ModelTier = "S" | "A" | "B" | "C" | "D" | "F";

export type ModelFitStatus = "can-run" | "tight" | "cant-run";

export type ModelRecommendation = {
  model: EmbeddedModelEntry;
  score: number;
  tier: ModelTier;
  tierLabel: string;
  fitStatus: ModelFitStatus;
  vramGb: number;
  tokensPerSec: number;
  memoryPct: number;
};

export const TIER_ORDER: ModelTier[] = ["S", "A", "B", "C", "D", "F"];

export const TIER_LABELS: Record<ModelTier, string> = {
  S: "运行极佳",
  A: "运行良好",
  B: "尚可",
  C: "勉强可用",
  D: "非常慢",
  F: "无法运行",
};

export const TIER_DESCRIPTIONS: Record<ModelTier, string> = {
  S: "推理速度快，内存充裕",
  A: "速度良好，运行舒适",
  B: "可用但非最佳",
  C: "较慢，上下文受限",
  D: "输出极慢",
  F: "内存不足，无法加载",
};

const RUNTIME_OVERHEAD_GB = 0.5;

type GpuSpec = { vram: number; bw: number };

const GPU_DB: Record<string, GpuSpec> = {
  "4090": { vram: 24, bw: 1008 },
  "4080": { vram: 16, bw: 717 },
  "4070 ti": { vram: 12, bw: 504 },
  "4070": { vram: 12, bw: 504 },
  "4060 ti": { vram: 16, bw: 272 },
  "4060": { vram: 8, bw: 272 },
  "3090": { vram: 24, bw: 936 },
  "3080": { vram: 10, bw: 760 },
  "3070": { vram: 8, bw: 448 },
  "3060": { vram: 12, bw: 360 },
  "7900 xtx": { vram: 24, bw: 960 },
  "7900 xt": { vram: 20, bw: 800 },
  "7800 xt": { vram: 16, bw: 624 },
  "7700 xt": { vram: 12, bw: 432 },
  "7600": { vram: 8, bw: 288 },
  "arc a770": { vram: 16, bw: 560 },
  "arc a750": { vram: 8, bw: 512 },
};

const APPLE_DB: Record<string, GpuSpec> = {
  "m4 max": { vram: 36, bw: 546 },
  "m4 pro": { vram: 24, bw: 273 },
  "m4": { vram: 16, bw: 120 },
  "m3 max": { vram: 36, bw: 400 },
  "m3 pro": { vram: 18, bw: 200 },
  "m3": { vram: 16, bw: 100 },
  "m2 max": { vram: 32, bw: 400 },
  "m2 pro": { vram: 16, bw: 200 },
  "m2": { vram: 8, bw: 100 },
  "m1 max": { vram: 32, bw: 400 },
  "m1 pro": { vram: 16, bw: 200 },
  "m1": { vram: 8, bw: 68 },
};

function lookupGpuSpec(gpuName: string): { spec: GpuSpec | null; isApple: boolean } {
  const lower = gpuName.toLowerCase();
  if (lower.includes("apple") || lower.includes("m1") || lower.includes("m2") || lower.includes("m3") || lower.includes("m4")) {
    for (const [key, spec] of Object.entries(APPLE_DB)) {
      if (lower.includes(key.replace(" ", "")) || lower.includes(key)) {
        return { spec, isApple: true };
      }
    }
    const ramMatch = lower.match(/(\d+)\s*gb/i);
    const ram = ramMatch ? Number(ramMatch[1]) : 16;
    return { spec: { vram: ram, bw: 120 }, isApple: true };
  }
  for (const [key, spec] of Object.entries(GPU_DB)) {
    if (lower.includes(key)) {
      return { spec, isApple: false };
    }
  }
  return { spec: null, isApple: false };
}

function detectGpuName(): string {
  try {
    const canvas = document.createElement("canvas");
    const gl = canvas.getContext("webgl2") ?? canvas.getContext("webgl");
    if (!gl) {
      return "未知 GPU";
    }
    const ext = gl.getExtension("WEBGL_debug_renderer_info");
    if (ext) {
      const renderer = gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) as string;
      if (renderer?.trim()) {
        return renderer.trim();
      }
    }
  } catch {
    // ignore
  }
  return "未知 GPU";
}

export function detectLocalHardware(): LocalHardwareProfile {
  const gpuName = detectGpuName();
  const { spec, isApple } = lookupGpuSpec(gpuName);
  const ramGb = typeof navigator.deviceMemory === "number" ? navigator.deviceMemory : 8;
  const cpuCores = navigator.hardwareConcurrency || 4;

  if (isApple) {
    const unified = spec?.vram ?? ramGb;
    return {
      gpuName,
      vramGb: unified,
      ramGb: unified,
      cpuCores,
      bandwidthGbs: spec?.bw ?? 120,
      isAppleSilicon: true,
      detected: true,
    };
  }

  return {
    gpuName,
    vramGb: spec?.vram ?? Math.min(ramGb, 8),
    ramGb,
    cpuCores,
    bandwidthGbs: spec?.bw ?? 200,
    isAppleSilicon: false,
    detected: Boolean(spec),
  };
}

export function modelVramGb(model: EmbeddedModelEntry): number {
  const fileSize = totalModelSize(model.files);
  if (fileSize > 0) {
    return fileSize / (1024 * 1024 * 1024) + RUNTIME_OVERHEAD_GB;
  }
  if (model.vramGbQ4 && model.vramGbQ4 > 0) {
    return model.vramGbQ4 + RUNTIME_OVERHEAD_GB;
  }
  const params = model.paramsB ?? 0;
  if (params > 0) {
    return params * 0.55 + RUNTIME_OVERHEAD_GB;
  }
  return 1;
}

function availableMemoryGb(hw: LocalHardwareProfile): number {
  if (hw.isAppleSilicon) {
    return hw.ramGb * 0.75;
  }
  return hw.vramGb;
}

function classifyFit(modelVram: number, hw: LocalHardwareProfile): ModelFitStatus {
  const total = availableMemoryGb(hw);
  const pct = (modelVram / total) * 100;
  if (hw.isAppleSilicon) {
    if (pct <= 52.5) {
      return "can-run";
    }
    if (pct <= 75) {
      return "tight";
    }
    return "cant-run";
  }
  if (pct <= 85) {
    return "can-run";
  }
  if (pct <= 110) {
    return "tight";
  }
  return "cant-run";
}

function speedScore(tokensPerSec: number): number {
  if (tokensPerSec >= 80) return 100;
  if (tokensPerSec >= 40) return 85;
  if (tokensPerSec >= 20) return 65;
  if (tokensPerSec >= 10) return 45;
  if (tokensPerSec >= 5) return 25;
  return 10;
}

function memoryScore(memPct: number): number {
  if (memPct <= 30) return 100;
  if (memPct <= 50) return 80;
  if (memPct <= 70) return 55;
  if (memPct <= 85) return 30;
  return 10;
}

function qualityBonus(paramsB: number): number {
  return Math.min(15, Math.log2(paramsB + 1) * 2.5);
}

function scoreToTier(score: number, fit: ModelFitStatus): ModelTier {
  if (fit === "cant-run") {
    return "F";
  }
  if (score >= 85) return "S";
  if (score >= 70) return "A";
  if (score >= 55) return "B";
  if (score >= 40) return "C";
  if (score >= 20) return "D";
  return "F";
}

export function recommendModels(
  models: EmbeddedModelEntry[],
  hw: LocalHardwareProfile,
): ModelRecommendation[] {
  const totalMem = availableMemoryGb(hw);
  const efficiency = hw.isAppleSilicon ? 0.65 : 0.7;

  return models
    .map((model) => {
      const vram = modelVramGb(model);
      const fit = classifyFit(vram, hw);
      const memPct = (vram / totalMem) * 100;
      const tokensPerSec = fit === "cant-run" ? 0 : (hw.bandwidthGbs / vram) * efficiency;

      let score = speedScore(tokensPerSec) * 0.55 + memoryScore(memPct) * 0.35;
      score += qualityBonus(model.paramsB ?? 0.5);
      if (fit === "tight") {
        score *= 0.65;
      }
      if (fit === "cant-run") {
        score = Math.min(score, 15);
      }

      const tier = scoreToTier(score, fit);
      return {
        model,
        score: Math.round(score),
        tier,
        tierLabel: TIER_LABELS[tier],
        fitStatus: fit,
        vramGb: Math.round(vram * 10) / 10,
        tokensPerSec: Math.round(tokensPerSec),
        memoryPct: Math.round(memPct),
      };
    })
    .sort((a, b) => b.score - a.score);
}

/** Merge server-side recommendation rows with the local model catalog. */
export function mergeServerRecommendations(
  models: EmbeddedModelEntry[],
  serverRecs: ServerModelRecommendation[],
): ModelRecommendation[] {
  const modelById = new Map(models.map((m) => [m.id, m]));
  const out: ModelRecommendation[] = [];
  for (const rec of serverRecs) {
    const model = modelById.get(rec.modelId);
    if (!model) {
      continue;
    }
    out.push({
      model,
      score: rec.score,
      tier: rec.tier,
      tierLabel: rec.tierLabel,
      fitStatus: rec.fitStatus,
      vramGb: rec.vramGb,
      tokensPerSec: rec.tokensPerSec,
      memoryPct: rec.memoryPct,
    });
  }
  return out;
}

export function resolvePlazaRecommendations(
  models: EmbeddedModelEntry[],
  hw: LocalHardwareProfile | null,
  serverRecs: ServerModelRecommendation[] | null,
): ModelRecommendation[] {
  if (serverRecs && serverRecs.length > 0) {
    const merged = mergeServerRecommendations(models, serverRecs);
    if (merged.length > 0) {
      return merged;
    }
  }
  const profile = hw ?? detectLocalHardware();
  return recommendModels(models, profile);
}

export function formatContextLength(n?: number): string {
  if (!n || n <= 0) {
    return "—";
  }
  if (n >= 1024 * 1024) {
    return `${Math.round(n / (1024 * 1024))}M ctx`;
  }
  if (n >= 1024) {
    return `${Math.round(n / 1024)}K ctx`;
  }
  return `${n} ctx`;
}

export function formatParamsB(n?: number): string {
  if (!n || n <= 0) {
    return "";
  }
  if (n < 1) {
    return `${Math.round(n * 1000)}M`;
  }
  return `${n}B`;
}

export function capabilityLabel(cap: string): string {
  const map: Record<string, string> = {
    chat: "对话",
    code: "代码",
    reasoning: "推理",
    vision: "视觉",
    embedding: "向量",
    rag: "检索",
    edge: "边缘",
    multilingual: "多语言",
    toolCalling: "工具调用",
  };
  return map[cap] ?? cap;
}

export function groupByTier(recommendations: ModelRecommendation[]): Map<ModelTier, ModelRecommendation[]> {
  const map = new Map<ModelTier, ModelRecommendation[]>();
  for (const tier of TIER_ORDER) {
    map.set(tier, []);
  }
  for (const rec of recommendations) {
    map.get(rec.tier)?.push(rec);
  }
  return map;
}

export function tierCounts(recommendations: ModelRecommendation[]): Record<ModelTier, number> {
  const counts = Object.fromEntries(TIER_ORDER.map((t) => [t, 0])) as Record<ModelTier, number>;
  for (const rec of recommendations) {
    counts[rec.tier]++;
  }
  return counts;
}

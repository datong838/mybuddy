/** 模型广场内置中文说明（参考 CanIRun.ai 文档，本地化改写） */

export type PlazaDocSection = {
  id: string;
  title: string;
  body: string;
};

export const PLAZA_DOCS: Record<string, PlazaDocSection> = {
  parameters: {
    id: "parameters",
    title: "参数量",
    body: `「7B」「70B」表示模型权重规模（十亿参数）。

• 1–3B：极快，适合边缘设备
• 7–8B：能力与速度的平衡点
• 13–14B：质量明显提升
• 27–34B：接近高端体验，需较大内存
• 70B+：接近前沿质量，对硬件要求很高

参数量越大通常越聪明，但也更占内存、推理更慢。`,
  },
  quantization: {
    id: "quantization",
    title: "量化（Quantization）",
    body: `量化通过降低权重精度来缩小体积、提升速度，会轻微损失质量。

常见 GGUF 格式：
• Q2_K — 2 bit，体积最小，质量损失明显
• Q4_K_M — 4 bit，★ 最常用，体积与质量平衡好
• Q6_K — 6 bit，接近无损，体积适中
• Q8_0 — 8 bit，质量极好，文件较大
• F16 — 16 bit 全精度，体积最大

OpenOcta 模型广场默认以 Q4_K_M 估算显存占用与推荐等级。`,
  },
  vram: {
    id: "vram",
    title: "显存 / 统一内存",
    body: `运行模型时，量化后的权重需完整载入 GPU 显存（Apple Silicon 为统一内存）。

若模型需要 8 GB 而设备只有 6 GB，通常会加载失败或退回到极慢的 CPU 推理。

推荐等级中的「内存占用 %」= 模型预估占用 ÷ 可用内存。`,
  },
  tokensPerSec: {
    id: "tokensPerSec",
    title: "推理速度（tok/s）",
    body: `每秒生成的 token 数，数值越高交互越流畅：

• 60+ tok/s — 几乎即时，体验极佳
• 30–60 — 快速舒适
• 15–30 — 可用，略有等待
• 5–15 — 适合批处理
• <5 — 交互体验较差

速度受模型大小、量化精度、内存带宽和 CPU/GPU 架构共同影响。`,
  },
  bandwidth: {
    id: "bandwidth",
    title: "内存带宽",
    body: `内存带宽（GB/s）决定从显存读取权重的速度。推理时主要瓶颈是读权重，因此带宽越高，同等模型下 tok/s 越高。

Apple Silicon 统一内存带宽较高，往往能运行比同容量独显更大的模型；RTX 4090 带宽也显著高于入门显卡。`,
  },
  contextLength: {
    id: "contextLength",
    title: "上下文长度",
    body: `模型单次可处理的 token 总量（输入 + 输出）。128K 上下文约等于一次处理十万字级别的长文档。

上下文越长占用内存越多。日常对话 4K–8K 通常足够；分析长文档才需要 32K 以上。`,
  },
  architecture: {
    id: "architecture",
    title: "Dense vs MoE",
    body: `Dense（稠密）：每个 token 激活全部参数，内存与速度可预测。

MoE（混合专家）：总参数量大，但每次只激活部分专家。质量可能更高，但完整权重仍需全部载入显存，VRAM 需求往往高于「激活参数量」所示。`,
  },
  tierList: {
    id: "tierList",
    title: "推荐等级 S–F",
    body: `综合推理速度、内存占用与模型质量得出的适配等级：

• S — 运行极佳：速度快、内存充裕
• A — 运行良好：速度舒适
• B — 尚可：可用但非最佳
• C — 勉强可用：较慢或上下文受限
• D — 非常慢：勉强能跑
• F — 无法运行：内存不足

评分权重：速度 55% + 内存 55% + 参数量质量加成；内存紧张时会降档。`,
  },
  runScore: {
    id: "runScore",
    title: "运行评分",
    body: `评分与评级越高，在本机运行效果越好。详情见推荐说明。`,
  },
  quantizationOptions: {
    id: "quantizationOptions",
    title: "量化选项表",
    body: `下表根据参数量估算各量化格式的显存需求与质量等级，供选型参考。

实际占用以 GGUF 文件大小为准；MoE 模型需按总参数量估算。安装前请确保可用内存高于「推荐显存」。`,
  },
};

export function plazaDoc(id: keyof typeof PLAZA_DOCS): PlazaDocSection {
  return PLAZA_DOCS[id];
}

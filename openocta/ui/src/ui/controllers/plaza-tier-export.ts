import {
  groupByTier,
  TIER_DESCRIPTIONS,
  TIER_LABELS,
  TIER_ORDER,
  type LocalHardwareProfile,
  type ModelRecommendation,
  type ModelTier,
} from "./model-recommendation.ts";

const TIER_COLORS: Record<ModelTier, { bg: string; fg: string; row: string }> = {
  S: { bg: "#22c55e", fg: "#ffffff", row: "#ecfdf5" },
  A: { bg: "#84cc16", fg: "#ffffff", row: "#f7fee7" },
  B: { bg: "#eab308", fg: "#ffffff", row: "#fefce8" },
  C: { bg: "#f97316", fg: "#ffffff", row: "#fff7ed" },
  D: { bg: "#ef4444", fg: "#ffffff", row: "#fef2f2" },
  F: { bg: "#6b7280", fg: "#ffffff", row: "#f3f4f6" },
};

export function formatHardwareSummary(hw: LocalHardwareProfile): string {
  const mem = hw.isAppleSilicon ? `${hw.ramGb} GB 统一内存` : `${hw.vramGb} GB 显存`;
  return `${hw.gpuName} | ${mem} | ${hw.ramGb} GB RAM | ~${hw.bandwidthGbs} GB/s | ${hw.cpuCores} 核`;
}

export function buildTierListText(
  recommendations: ModelRecommendation[],
  hw: LocalHardwareProfile,
): string {
  const grouped = groupByTier(recommendations);
  const lines: string[] = [
    "OpenOcta 模型推荐分级表",
    formatHardwareSummary(hw),
    "量化基准：Q4_K_M · 实际表现因环境而异",
    "",
  ];
  for (const tier of TIER_ORDER) {
    const items = grouped.get(tier) ?? [];
    if (items.length === 0) {
      continue;
    }
    lines.push(`${tier} · ${TIER_LABELS[tier]} — ${TIER_DESCRIPTIONS[tier]}`);
    for (const rec of items) {
      const speed = rec.tokensPerSec > 0 ? ` ~${rec.tokensPerSec} tok/s` : "";
      lines.push(`  • ${rec.model.name} (${rec.score}分${speed})`);
    }
    lines.push("");
  }
  lines.push("等级说明：S 运行极佳 · A 运行良好 · B 尚可 · C 勉强可用 · D 非常慢 · F 无法运行");
  return lines.join("\n");
}

export async function copyTierListText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

function wrapText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string[] {
  const words = text.split(/\s+/);
  const lines: string[] = [];
  let current = "";
  for (const word of words) {
    const next = current ? `${current} ${word}` : word;
    if (ctx.measureText(next).width > maxWidth && current) {
      lines.push(current);
      current = word;
    } else {
      current = next;
    }
  }
  if (current) {
    lines.push(current);
  }
  return lines.length > 0 ? lines : [text];
}

function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
}

export function downloadTierListImage(
  recommendations: ModelRecommendation[],
  hw: LocalHardwareProfile,
  filename = "openocta-tier-list.png",
): void {
  const grouped = groupByTier(recommendations);
  const width = 920;
  const pad = 24;
  const tierLabelW = 52;
  const rowMinH = 56;
  let height = pad * 2 + 72;
  for (const tier of TIER_ORDER) {
    const items = grouped.get(tier) ?? [];
    if (items.length === 0) {
      continue;
    }
    const rows = Math.ceil(items.length / 4);
    height += Math.max(rowMinH, rows * 36 + 16);
  }
  height += 48;

  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) {
    return;
  }

  ctx.fillStyle = "#ffffff";
  ctx.fillRect(0, 0, width, height);

  roundRect(ctx, pad / 2, pad / 2, width - pad, height - pad, 16);
  ctx.fillStyle = "#fafafa";
  ctx.fill();
  ctx.strokeStyle = "#e5e7eb";
  ctx.lineWidth = 1;
  ctx.stroke();

  ctx.fillStyle = "#111827";
  ctx.font = "bold 20px system-ui, sans-serif";
  ctx.fillText("OpenOcta 模型分级", pad + 8, pad + 28);
  ctx.fillStyle = "#6b7280";
  ctx.font = "12px system-ui, sans-serif";
  const summary = formatHardwareSummary(hw);
  for (const [i, line] of wrapText(ctx, summary, width - pad * 2 - 16).entries()) {
    ctx.fillText(line, pad + 8, pad + 48 + i * 16);
  }

  let y = pad + 72;
  for (const tier of TIER_ORDER) {
    const items = grouped.get(tier) ?? [];
    if (items.length === 0) {
      continue;
    }
    const colors = TIER_COLORS[tier];
    const pillRows = Math.ceil(items.length / 4);
    const rowH = Math.max(rowMinH, pillRows * 36 + 16);

    ctx.fillStyle = colors.row;
    roundRect(ctx, pad + 4, y, width - pad - 8, rowH, 8);
    ctx.fill();

    ctx.fillStyle = colors.bg;
    roundRect(ctx, pad + 12, y + 10, tierLabelW, rowH - 20, 6);
    ctx.fill();
    ctx.fillStyle = colors.fg;
    ctx.font = "bold 22px system-ui, sans-serif";
    ctx.textAlign = "center";
    ctx.fillText(tier, pad + 12 + tierLabelW / 2, y + rowH / 2 + 8);
    ctx.textAlign = "left";

    let px = pad + tierLabelW + 24;
    let py = y + 14;
    let col = 0;
    ctx.font = "12px system-ui, sans-serif";
    for (const rec of items) {
      const label = rec.tokensPerSec > 0
        ? `${rec.model.name}  ${rec.tokensPerSec}t/s`
        : rec.model.name;
      const tw = Math.min(ctx.measureText(label).width + 20, 200);
      ctx.fillStyle = "#ffffff";
      roundRect(ctx, px, py, tw, 26, 13);
      ctx.fill();
      ctx.strokeStyle = "#d1d5db";
      ctx.stroke();
      ctx.fillStyle = "#374151";
      ctx.fillText(label, px + 10, py + 17);
      col++;
      if (col >= 4) {
        col = 0;
        px = pad + tierLabelW + 24;
        py += 32;
      } else {
        px += tw + 8;
      }
    }
    y += rowH + 6;
  }

  ctx.fillStyle = "#9ca3af";
  ctx.font = "11px system-ui, sans-serif";
  ctx.fillText("基于 Q4_K_M 量化估算 · 实际结果可能有所不同", pad + 8, height - pad - 8);

  canvas.toBlob((blob) => {
    if (!blob) {
      return;
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }, "image/png");
}

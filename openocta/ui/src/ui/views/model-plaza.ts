import { html, nothing } from "lit";
import { icons } from "../icons.js";
import { renderPlazaHelpHint } from "../components/plaza-help-hint.ts";
import {
  formatModelSize,
  totalModelSize,
  type EmbeddedModelEntry,
  type EmbeddedModelProgress,
} from "../controllers/embedded-models.ts";
import {
  resolveEmbeddedChatProxyURL,
  type PlazaChatMessage,
} from "../controllers/embedded-chat-test.ts";
import {
  estimateQuantizationOptions,
  useCaseLabels,
  type PlazaModelDetailInfo,
} from "../controllers/plaza-model-detail.ts";
import {
  buildTierListText,
  copyTierListText,
  downloadTierListImage,
  formatHardwareSummary,
} from "../controllers/plaza-tier-export.ts";
import {
  capabilityLabel,
  detectLocalHardware,
  formatContextLength,
  formatParamsB,
  groupByTier,
  resolvePlazaRecommendations,
  TIER_DESCRIPTIONS,
  TIER_LABELS,
  TIER_ORDER,
  type LocalHardwareProfile,
  type ModelRecommendation,
  type ModelTier,
  type ServerModelRecommendation,
} from "../controllers/model-recommendation.ts";
import { plazaDoc } from "../data/plaza-model-docs.ts";
import { renderPlazaManualImportModal } from "../components/plaza-manual-import-modal.ts";

/** 推荐说明弹窗内「隐藏 F 级」开关（仅 UI 态） */
let guideHideFTier = false;

export type ModelPlazaProps = {
  models: EmbeddedModelEntry[];
  loading: boolean;
  error: string | null;
  busyId: string | null;
  downloadProgress: EmbeddedModelProgress | null;
  downloadingModelId: string | null;
  detailModel: EmbeddedModelEntry | null;
  detailInfo: PlazaModelDetailInfo | null;
  detailLoading: boolean;
  detailError: string | null;
  recommendOpen: boolean;
  manualImportOpen: boolean;
  hardware: LocalHardwareProfile | null;
  serverRecommendations: ServerModelRecommendation[] | null;
  recommendationsLoading: boolean;
  onRefresh: () => void;
  onDownload: (modelId: string) => void;
  onCancelDownload: () => void;
  onStart: (model: EmbeddedModelEntry) => void;
  onStop: (modelId: string) => void;
  onDelete: (model: EmbeddedModelEntry) => void;
  onChat: (model: EmbeddedModelEntry) => void;
  onSelectModel: (model: EmbeddedModelEntry | null) => void;
  onOpenRecommend: () => void;
  onCloseRecommend: () => void;
  onOpenManualImport: () => void;
  onCloseManualImport: () => void;
  onHardwareChange: (hw: LocalHardwareProfile) => void;
  gatewayHost: string;
  chatModel: EmbeddedModelEntry | null;
  chatMessages: PlazaChatMessage[];
  chatInput: string;
  chatLoading: boolean;
  chatError: string | null;
  onCloseChat: () => void;
  onChatInput: (value: string) => void;
  onSendChat: () => void;
  searchQuery?: string;
  onSearchChange?: (query: string) => void;
  /** 主模型库页展示标题与右上角搜索；引导内嵌时为 false */
  showPageHeader?: boolean;
  pageTitle?: string;
};

export function filterEmbeddedModelsByQuery(models: EmbeddedModelEntry[], query: string) {
  const q = query.trim().toLowerCase();
  if (!q) {
    return models;
  }
  return models.filter((m) => {
    const haystack = `${m.name} ${m.description ?? ""} ${m.id} ${m.provider ?? ""}`.toLowerCase();
    return haystack.includes(q);
  });
}

function formatProgressBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) {
    return "";
  }
  return formatModelSize(bytes);
}

function renderProgressLabel(progress: EmbeddedModelProgress, pct: number) {
  const done = progress.bytesDone;
  const total = progress.bytesTotal;
  const sizeLabel =
    done && done > 0
      ? total && total > 0
        ? `${formatProgressBytes(done)} / ${formatProgressBytes(total)}`
        : formatProgressBytes(done)
      : "";
  return { message: progress.message ?? "下载中", sizeLabel, pct };
}

function renderProgressBar(progress: EmbeddedModelProgress | null, compact = false) {
  if (!progress?.message) {
    return nothing;
  }
  const pct = Math.max(0, Math.min(100, progress.percent ?? 0));
  const label = renderProgressLabel(progress, pct);
  return html`
    <div class="embedded-model-progress" @click=${(e: Event) => e.stopPropagation()}>
      <div class="embedded-model-progress__track" aria-hidden="true">
        <div class="embedded-model-progress__bar" style="width: ${pct}%"></div>
      </div>
      <div class="embedded-model-progress__label">
        ${compact
          ? nothing
          : html`<span class="embedded-model-progress__message">${label.message}</span>`}
        ${label.sizeLabel
          ? html`<span class="embedded-model-progress__meta">${label.sizeLabel} · ${label.pct}%</span>`
          : html`<span class="embedded-model-progress__meta">${label.pct}%</span>`}
      </div>
    </div>
  `;
}

function tierClass(tier: ModelTier): string {
  return `plaza-tier plaza-tier--${tier.toLowerCase()}`;
}

function tierScoreClass(tier: ModelTier): string {
  return `plaza-score plaza-score--${tier.toLowerCase()}`;
}

const CAPABILITY_ICONS: Record<string, typeof icons.plazaChat> = {
  toolCalling: icons.plazaWrench,
  chat: icons.plazaChat,
  code: icons.plazaCode,
  reasoning: icons.plazaBrain,
  vision: icons.plazaEye,
  embedding: icons.plazaLayers,
  rag: icons.plazaGlobe,
  edge: icons.plazaZap,
  multilingual: icons.plazaGlobe,
};

function formatReleasedAgo(releasedAt?: string): string {
  if (!releasedAt?.trim()) {
    return "";
  }
  const raw = releasedAt.trim();
  const match = raw.match(/^(\d{4})-(\d{2})/);
  if (!match) {
    return raw;
  }
  const released = new Date(Number(match[1]), Number(match[2]) - 1, 1);
  const now = new Date();
  const months =
    (now.getFullYear() - released.getFullYear()) * 12 + (now.getMonth() - released.getMonth());
  if (months < 1) {
    return "本月";
  }
  if (months < 12) {
    return `${months}mo ago`;
  }
  const years = Math.floor(months / 12);
  return years === 1 ? "1 year ago" : `${years} years ago`;
}

function licenseBadgeClass(license?: string): string {
  const key = (license ?? "").toLowerCase();
  if (key.includes("apache")) {
    return "plaza-license plaza-license--apache";
  }
  if (key.includes("mit")) {
    return "plaza-license plaza-license--mit";
  }
  if (key.includes("gemma")) {
    return "plaza-license plaza-license--gemma";
  }
  return "plaza-license";
}

function renderCapabilityIcons(model: EmbeddedModelEntry) {
  const caps = new Set(model.capabilities ?? []);
  if (model.toolCalling) {
    caps.add("toolCalling");
  }
  if (model.multimodal) {
    caps.add("vision");
  }
  if (caps.size === 0) {
    return nothing;
  }
  return html`
    <span class="plaza-list__cap-icons" aria-label="模型能力">
      ${[...caps].map((cap) => {
        const icon = CAPABILITY_ICONS[cap];
        if (!icon) {
          return nothing;
        }
        return html`
          <span class="plaza-cap-icon" title=${capabilityLabel(cap)} aria-hidden="true">${icon}</span>
        `;
      })}
    </span>
  `;
}

function renderPlazaCell(className: string, content: unknown) {
  const hasContent = content !== null && content !== undefined && content !== nothing && content !== "" && content !== "—";
  return html`
    <div class="plaza-list__cell ${className}">
      ${hasContent ? content : html`<span class="plaza-list__cell-empty">—</span>`}
    </div>
  `;
}

function renderPlazaActionSlot(content: unknown) {
  return html`
    <div class="plaza-list__action-slot">
      ${content ? content : html`<span class="plaza-list__action-slot-empty" aria-hidden="true"></span>`}
    </div>
  `;
}

function renderPlazaScoreHeader() {
  return html`
    <div class="plaza-list__head-score" @click=${(e: Event) => e.stopPropagation()}>
      <span class="plaza-list__head-label">运行评分</span>
      ${renderPlazaHelpHint(plazaDoc("runScore").body, "运行评分", { nowrap: true })}
    </div>
  `;
}

function renderPlazaScoreCell(rec?: ModelRecommendation) {
  if (!rec) {
    return nothing;
  }
  return html`
    <div class="plaza-list__score-group">
      <span class="plaza-list__score-num ${tierScoreClass(rec.tier)}" title="运行评分">
        ${rec.score}
      </span>
      <span
        class="plaza-list__tier-chip ${tierClass(rec.tier)}"
        title="${TIER_LABELS[rec.tier]}（推荐等级 ${rec.tier}）"
      >${rec.tier}</span>
    </div>
  `;
}

function renderPlazaRowActions(
  props: ModelPlazaProps,
  model: EmbeddedModelEntry,
  isBusy: boolean,
  isDownloading: boolean,
) {
  const stopClick = (e: Event) => e.stopPropagation();

  if (!model.installed) {
    if (model.downloadable === false) {
      return html`
        <div class="plaza-list__actions" @click=${stopClick}>
          ${renderPlazaActionSlot(nothing)}
          ${renderPlazaActionSlot(
            model.hfUrl
              ? html`
                  <a
                    class="market-card-icon-btn"
                    href=${model.hfUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="在 HuggingFace 查看"
                    @click=${stopClick}
                  >${icons.link}</a>
                `
              : html`<span class="plaza-list__hf-hint" title="HF 源仓库">HF</span>`,
          )}
        </div>
      `;
    }
    return html`
      <div class="plaza-list__actions" @click=${stopClick}>
        ${renderPlazaActionSlot(nothing)}
        ${renderPlazaActionSlot(html`
          <button
            class="market-card-icon-btn primary"
            type="button"
            title=${isDownloading ? "下载中" : "下载内嵌模型"}
            ?disabled=${props.loading || isBusy || Boolean(props.downloadingModelId)}
            @click=${() => props.onDownload(model.id)}
          >${isDownloading ? icons.loader2 : icons.download}</button>
        `)}
      </div>
    `;
  }

  if (model.running) {
    return html`
      <div class="plaza-list__actions" @click=${stopClick}>
        ${renderPlazaActionSlot(
          model.kind !== "embedding"
            ? html`
                <button
                  class="market-card-icon-btn primary"
                  type="button"
                  title="测试对话"
                  ?disabled=${isBusy}
                  @click=${() => props.onChat(model)}
                >${icons.plazaChat}</button>
              `
            : nothing,
        )}
        ${renderPlazaActionSlot(html`
          <button
            class="market-card-icon-btn"
            type="button"
            title="停止"
            ?disabled=${isBusy}
            @click=${() => props.onStop(model.id)}
          >${icons.powerOff}</button>
        `)}
      </div>
    `;
  }

  return html`
    <div class="plaza-list__actions" @click=${stopClick}>
      ${renderPlazaActionSlot(html`
        <button
          class="market-card-icon-btn primary"
          type="button"
          title="启动"
          ?disabled=${isBusy}
          @click=${() => props.onStart(model)}
        >${icons.power}</button>
      `)}
      ${renderPlazaActionSlot(html`
        <button
          class="market-card-icon-btn danger"
          type="button"
          title="删除"
          ?disabled=${isBusy}
          @click=${() => props.onDelete(model)}
        >${icons.trash}</button>
      `)}
    </div>
  `;
}

function renderPlazaListHeader() {
  return html`
    <div class="plaza-list__row plaza-list__row--header" aria-hidden="true">
      <div class="plaza-list__identity">
        <span class="plaza-list__head-label">模型</span>
      </div>
      ${renderPlazaCell("plaza-list__cell--released", html`<span class="plaza-list__head-label">发布</span>`)}
      ${renderPlazaCell("plaza-list__cell--size", html`<span class="plaza-list__head-label">体积 / 内存</span>`)}
      ${renderPlazaCell("plaza-list__cell--ctx", html`<span class="plaza-list__head-label">上下文</span>`)}
      ${renderPlazaCell("plaza-list__cell--quant", html`<span class="plaza-list__head-label">量化</span>`)}
      ${renderPlazaCell("plaza-list__cell--speed", html`<span class="plaza-list__head-label">推理速度</span>`)}
      ${renderPlazaCell("plaza-list__cell--score", renderPlazaScoreHeader())}
      <div class="plaza-list__head-actions">
        <span class="plaza-list__head-label">操作</span>
      </div>
    </div>
  `;
}

function renderModelRow(props: ModelPlazaProps, model: EmbeddedModelEntry, rec?: ModelRecommendation) {
  const sizeBytes = totalModelSize(model.files);
  const sizeLabel = formatModelSize(sizeBytes);
  const sizeGb = sizeBytes > 0 ? (sizeBytes / (1024 * 1024 * 1024)).toFixed(1) : rec ? rec.vramGb.toFixed(1) : "";
  const isBusy = props.busyId === model.id;
  const isDownloading = props.downloadingModelId === model.id;
  const paramsLabel = formatParamsB(model.paramsB);
  const ctxLabel = formatContextLength(model.contextLength);
  const releasedAgo = formatReleasedAgo(model.releasedAt);
  const providerLine = [model.provider, paramsLabel].filter(Boolean).join(" · ");

  return html`
    <div
      class="plaza-list__row ${model.running ? "plaza-list__row--active" : ""} ${isDownloading ? "plaza-list__row--downloading" : ""} ${isBusy ? "is-disabled" : ""}"
      role="button"
      tabindex="0"
      @click=${() => props.onSelectModel(model)}
      @keydown=${(e: KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          props.onSelectModel(model);
        }
      }}
    >
      <div class="plaza-list__identity">
        <div class="plaza-list__title-row">
          <h3 class="plaza-list__name">${model.name}</h3>
          ${renderCapabilityIcons(model)}
          ${model.builtin || model.featured
            ? html`<span class="plaza-star" title="推荐" aria-label="推荐">★</span>`
            : nothing}
          ${model.license
            ? html`<span class=${licenseBadgeClass(model.license)}>${model.license}</span>`
            : nothing}
          ${model.sideloaded
            ? html`<span class="plaza-status plaza-status--sideloaded">手动导入</span>`
            : nothing}
          ${model.installed
            ? model.running
              ? html`<span class="plaza-status plaza-status--running">运行中</span>`
              : html`<span class="plaza-status plaza-status--installed">已安装</span>`
            : isDownloading
              ? html`<span class="plaza-status plaza-status--downloading">正在下载</span>`
              : nothing}
        </div>
        ${providerLine ? html`<p class="plaza-list__provider">${providerLine}</p>` : nothing}
        <p class="plaza-list__desc">${model.description}</p>
      </div>

      ${renderPlazaCell(
        "plaza-list__cell--released",
        releasedAgo ? html`<span class="plaza-list__stat plaza-list__stat--muted">${releasedAgo}</span>` : nothing,
      )}
      ${renderPlazaCell(
        "plaza-list__cell--size",
        sizeGb
          ? html`
              <span class="plaza-list__stat plaza-list__stat--vram">
                ${sizeGb} GB
                ${rec
                  ? html`<span class="plaza-mem-badge ${tierClass(rec.tier)}" title="内存占用">${rec.memoryPct}%</span>`
                  : nothing}
              </span>
            `
          : sizeLabel
            ? html`<span class="plaza-list__stat">${sizeLabel}</span>`
            : nothing,
      )}
      ${renderPlazaCell(
        "plaza-list__cell--ctx",
        ctxLabel !== "—" ? html`<span class="plaza-list__stat">${ctxLabel}</span>` : nothing,
      )}
      ${renderPlazaCell(
        "plaza-list__cell--quant",
        model.quantization
          ? html`<span class="plaza-list__stat plaza-list__stat--muted">${model.quantization}</span>`
          : nothing,
      )}
      ${renderPlazaCell(
        "plaza-list__cell--speed",
        rec && rec.tokensPerSec > 0
          ? html`<span class="plaza-list__stat plaza-list__stat--speed">~${rec.tokensPerSec} tok/s</span>`
          : nothing,
      )}
      ${renderPlazaCell("plaza-list__cell--score", renderPlazaScoreCell(rec))}

      ${renderPlazaRowActions(props, model, isBusy, isDownloading)}
      ${isDownloading ? renderProgressBar(props.downloadProgress, true) : nothing}
    </div>
  `;
}

function renderRecommendBanner(
  props: ModelPlazaProps,
  topRecs: ModelRecommendation[],
) {
  return html`
    <div
      class="plaza-recommend-banner"
      role="button"
      tabindex="0"
      @click=${props.onOpenRecommend}
      @keydown=${(e: KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          props.onOpenRecommend();
        }
      }}
    >
      <div class="plaza-recommend-banner__body">
        <strong>尚未安装本地模型</strong>
        <p>根据${props.hardware?.serverSide ? "Gateway 服务器" : "当前设备"}资源，推荐优先安装：
          ${topRecs.slice(0, 3).map((r, i) => html`${i > 0 ? "、" : ""}${r.model.name}（${r.tier} 级）`)}</p>
      </div>
      <span class="plaza-recommend-banner__cta">查看推荐说明 →</span>
    </div>
  `;
}

function hardwareHintText(hw: LocalHardwareProfile): string {
  if (hw.serverSide) {
    return hw.detected
      ? "已自动检测 Gateway 服务器硬件，可手动调整以模拟其他配置。"
      : "未能完整识别服务器 GPU，请手动填写参数后重新计算推荐。";
  }
  return hw.detected
    ? "未连接 Gateway，使用浏览器本机估算（仅供参考）。"
    : "未连接 Gateway 且未能识别本机 GPU，推荐结果仅供参考。";
}

function resolveRecommendations(
  props: ModelPlazaProps,
  hw: LocalHardwareProfile,
): ModelRecommendation[] {
  return resolvePlazaRecommendations(props.models, hw, props.serverRecommendations);
}

function renderHardwareSummaryBar(hw: LocalHardwareProfile, onChange: (hw: LocalHardwareProfile) => void) {
  return html`
    <div class="plaza-hw-summary">
      <div class="plaza-hw-summary__row">
        <span class="plaza-hw-summary__label">处理器 / GPU</span>
        <span class="plaza-hw-summary__value" title=${hw.gpuName}>${hw.gpuName}</span>
      </div>
      <div class="plaza-hw-summary__grid">
        <label class="plaza-hw-form__field">
          <span>${hw.isAppleSilicon ? "统一内存 (GB)" : "显存 (GB)"}</span>
          <input
            type="number"
            min="1"
            max="512"
            step="1"
            .value=${String(hw.isAppleSilicon ? hw.ramGb : hw.vramGb)}
            @input=${(e: Event) => {
              const val = Number((e.target as HTMLInputElement).value) || 8;
              if (hw.isAppleSilicon) {
                onChange({ ...hw, ramGb: val, vramGb: val });
              } else {
                onChange({ ...hw, vramGb: val });
              }
            }}
          />
        </label>
        <label class="plaza-hw-form__field">
          <span>系统内存 (GB)</span>
          <input
            type="number"
            min="1"
            max="512"
            step="1"
            .value=${String(hw.ramGb)}
            @input=${(e: Event) => {
              const val = Number((e.target as HTMLInputElement).value) || 8;
              onChange({ ...hw, ramGb: val });
            }}
          />
        </label>
        <label class="plaza-hw-form__field">
          <span>内存带宽 (GB/s)</span>
          <input
            type="number"
            min="10"
            max="2000"
            step="1"
            .value=${String(hw.bandwidthGbs)}
            @input=${(e: Event) => {
              const val = Number((e.target as HTMLInputElement).value) || 200;
              onChange({ ...hw, bandwidthGbs: val });
            }}
          />
        </label>
        <label class="plaza-hw-form__field">
          <span>CPU 核心</span>
          <input
            type="number"
            min="1"
            max="128"
            step="1"
            .value=${String(hw.cpuCores)}
            @input=${(e: Event) => {
              const val = Number((e.target as HTMLInputElement).value) || 4;
              onChange({ ...hw, cpuCores: val });
            }}
          />
        </label>
      </div>
      <p class="plaza-hw-summary__hint muted">
        ${hardwareHintText(hw)}
      </p>
    </div>
  `;
}

function renderScoringCriteria() {
  const tierDoc = plazaDoc("tierList");
  const speedDoc = plazaDoc("tokensPerSec");
  return html`
    <section class="plaza-guide-section">
      <h3 class="plaza-guide-section__title">
        评分标准
        ${renderPlazaHelpHint(tierDoc.body, tierDoc.title)}
      </h3>
      <ul class="plaza-guide-criteria">
        <li><strong>综合得分</strong> = 推理速度 (55%) + 内存占用 (35%) + 参数量质量加成</li>
        <li><strong>推理速度</strong>：由内存带宽 ÷ 模型显存估算 tok/s（${speedDoc.body.split("\n")[1]?.trim() ?? "见说明"}）</li>
        <li><strong>内存占用</strong>：Q4_K_M 量化文件 + 运行时开销，占可用内存百分比</li>
        <li><strong>等级映射</strong>：S ≥85 · A ≥70 · B ≥55 · C ≥40 · D ≥20 · F 内存不足</li>
      </ul>
    </section>
  `;
}

function renderTierChart(
  recommendations: ModelRecommendation[],
  hw: LocalHardwareProfile,
  onSelectModel: (model: EmbeddedModelEntry) => void,
) {
  const filtered = guideHideFTier ? recommendations.filter((r) => r.tier !== "F") : recommendations;
  const grouped = groupByTier(filtered);

  return html`
    <div class="plaza-tier-chart" id="plaza-tier-chart-export">
      <div class="plaza-tier-chart__header">
        <span class="plaza-tier-chart__brand">OpenOcta</span>
        <span class="plaza-tier-chart__hw">${formatHardwareSummary(hw)}</span>
      </div>
      <div class="plaza-tier-chart__body">
        ${TIER_ORDER.map((tier) => {
          const items = grouped.get(tier) ?? [];
          if (items.length === 0) {
            return html`
              <div class="plaza-tier-chart__row plaza-tier-chart__row--empty">
                <span class=${tierClass(tier)}>${tier}</span>
                <span class="plaza-tier-chart__empty muted">—</span>
              </div>
            `;
          }
          return html`
            <div class="plaza-tier-chart__row">
              <span class=${tierClass(tier)} title="${TIER_LABELS[tier]}">${tier}</span>
              <div class="plaza-tier-chart__pills">
                ${items.map(
                  (rec) => html`
                    <button
                      type="button"
                      class="plaza-tier-pill"
                      title="${TIER_LABELS[rec.tier]} · ${rec.score} 分"
                      @click=${() => onSelectModel(rec.model)}
                    >
                      <span class="plaza-tier-pill__name">${rec.model.name}</span>
                      ${rec.tokensPerSec > 0
                        ? html`<span class="plaza-tier-pill__speed">${rec.tokensPerSec}t/s</span>`
                        : nothing}
                    </button>
                  `,
                )}
              </div>
            </div>
          `;
        })}
      </div>
      <div class="plaza-tier-chart__footer muted">
        <span>基于 Q4_K_M 量化估算</span>
        <span>实际表现因环境与量化版本而异</span>
      </div>
    </div>
    <div class="plaza-tier-legend">
      ${TIER_ORDER.map(
        (tier) => html`
          <span class="plaza-tier-legend__item">
            <span class=${tierClass(tier)}>${tier}</span>
            ${TIER_LABELS[tier]} — ${TIER_DESCRIPTIONS[tier]}
          </span>
        `,
      )}
    </div>
  `;
}

function renderRecommendGuideModal(props: ModelPlazaProps) {
  if (!props.recommendOpen) {
    return nothing;
  }
  const hw = props.hardware ?? detectLocalHardware();
  const recommendations = resolveRecommendations(props, hw);
  const exportText = buildTierListText(recommendations, hw);

  return html`
    <div class="modal-overlay" @click=${props.onCloseRecommend}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large plaza-guide-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div>
            <h2>模型推荐说明</h2>
            <p class="plaza-detail-sub">${hw.serverSide ? "Gateway 服务器资源" : "当前设备资源（离线估算）"}、评分标准与 S–F 分级一览</p>
          </div>
          <button class="emp-detail-modal__close" type="button" aria-label="关闭" @click=${props.onCloseRecommend}>
            ${icons.x}
          </button>
        </div>
        <div class="emp-detail-modal__body plaza-guide-modal__body">
          <section class="plaza-guide-section">
            <h3 class="plaza-guide-section__title">
              ${hw.serverSide ? "Gateway 服务器资源" : "当前设备资源（离线估算）"}
            </h3>
            ${renderHardwareSummaryBar(hw, props.onHardwareChange)}
          </section>
          ${renderScoringCriteria()}
          <section class="plaza-guide-section">
            <div class="plaza-guide-section__toolbar">
              <h3 class="plaza-guide-section__title">
                分级列表
                ${renderPlazaHelpHint(plazaDoc("tierList").body, "分级说明")}
              </h3>
              <div class="plaza-guide-section__actions">
                <label class="plaza-guide-checkbox">
                  <input
                    type="checkbox"
                    .checked=${guideHideFTier}
                    @change=${(e: Event) => {
                      guideHideFTier = (e.target as HTMLInputElement).checked;
                      props.onHardwareChange({ ...hw });
                    }}
                  />
                  隐藏 F 级
                </label>
                <button
                  class="btn btn--sm"
                  type="button"
                  title="复制分级表为文本"
                  @click=${async () => {
                    const ok = await copyTierListText(exportText);
                    if (!ok) {
                      window.prompt("复制以下内容：", exportText);
                    }
                  }}
                >${icons.copy} 复制</button>
                <button
                  class="btn btn--sm"
                  type="button"
                  title="下载分级表为 PNG 图片"
                  @click=${() => downloadTierListImage(recommendations, hw)}
                >${icons.download} 下载图片</button>
              </div>
            </div>
            ${renderTierChart(recommendations, hw, (model) => {
              props.onSelectModel(model);
            })}
          </section>
        </div>
      </div>
    </div>
  `;
}

function renderAboutParagraphs(text: string) {
  const paragraphs = text.split(/\n\n+/).map((p) => p.trim()).filter(Boolean);
  if (paragraphs.length === 0) {
    return nothing;
  }
  return paragraphs.map(
    (p) => html`<p class="emp-detail-summary plaza-detail-about">${p}</p>`,
  );
}

function renderQuantizationTable(model: EmbeddedModelEntry) {
  const options = estimateQuantizationOptions(model);
  if (options.length === 0) {
    return nothing;
  }
  const quantDoc = plazaDoc("quantizationOptions");
  return html`
    <section class="plaza-detail-section">
      <h3 class="plaza-detail-section__title">
        量化选项
        ${renderPlazaHelpHint(quantDoc.body, quantDoc.title)}
        ${renderPlazaHelpHint(plazaDoc("quantization").body, "量化说明")}
      </h3>
      <div class="plaza-quant-table-wrap">
        <table class="plaza-quant-table">
          <thead>
            <tr>
              <th>格式</th>
              <th>位数</th>
              <th>预估显存</th>
              <th>质量</th>
            </tr>
          </thead>
          <tbody>
            ${options.map(
              (opt) => html`
                <tr class="plaza-quant-table__row--${opt.qualityKey} ${model.quantization === opt.quant ? "is-default" : ""}">
                  <td><code>${opt.quant}</code>${model.quantization === opt.quant ? html` <span class="plaza-quant-default">默认</span>` : nothing}</td>
                  <td>${opt.bits}</td>
                  <td>${opt.vramGb} GB</td>
                  <td>${opt.quality}</td>
                </tr>
              `,
            )}
          </tbody>
        </table>
      </div>
    </section>
  `;
}

function renderDetailModal(props: ModelPlazaProps) {
  const model = props.detailModel;
  if (!model) {
    return nothing;
  }
  const hw = props.hardware ?? detectLocalHardware();
  const rec = resolveRecommendations(props, hw).find((r) => r.model.id === model.id);
  const sizeLabel = formatModelSize(totalModelSize(model.files));
  const info = props.detailInfo;
  const useCases = useCaseLabels(model);
  const aboutText = (info?.description ?? model.description ?? "").trim();

  return html`
    <div
      class="modal-overlay ${props.recommendOpen ? "plaza-modal-overlay--detail" : ""}"
      @click=${() => props.onSelectModel(null)}
    >
      <div
        class="modal card emp-detail-modal emp-detail-modal--large plaza-detail-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div class="emp-detail-title-wrap">
            <div class="emp-detail-logo emp-card__icon--default" aria-hidden="true">${icons.modelCube}</div>
            <div>
              <h2 class="emp-detail-title">
                ${model.name}
                ${rec ? html`<span class=${tierClass(rec.tier)}>${rec.tier}</span>` : nothing}
              </h2>
              <p class="plaza-detail-sub">
                ${[model.provider, formatParamsB(model.paramsB), model.architecture, model.license ?? info?.license]
                  .filter(Boolean)
                  .join(" · ")}
              </p>
            </div>
          </div>
          <button
            class="emp-detail-modal__close"
            type="button"
            aria-label="关闭"
            @click=${() => props.onSelectModel(null)}
          >
            ${icons.x}
          </button>
        </div>
        <div class="emp-detail-modal__body plaza-detail-modal__body">
          ${useCases.length > 0
            ? html`
                <section class="plaza-detail-section">
                  <h3 class="plaza-detail-section__title">适用场景</h3>
                  <div class="plaza-detail-tags">
                    ${useCases.map((u) => html`<span class="plaza-detail-tag">${u}</span>`)}
                  </div>
                </section>
              `
            : nothing}
          ${renderQuantizationTable(model)}
          <section class="plaza-detail-section">
            <h3 class="plaza-detail-section__title">模型说明</h3>
            ${aboutText ? renderAboutParagraphs(aboutText) : html`<p class="muted">暂无说明</p>`}
          </section>
          <section class="plaza-detail-section">
            <h3 class="plaza-detail-section__title">
              规格参数
              ${renderPlazaHelpHint(plazaDoc("parameters").body, "参数量")}
            </h3>
            <div class="plaza-spec-grid">
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">参数量</span>
                <span class="plaza-spec-item__value">${formatParamsB(model.paramsB) || "—"}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">架构</span>
                <span class="plaza-spec-item__value">${model.architecture ?? "—"}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">上下文</span>
                <span class="plaza-spec-item__value">${formatContextLength(model.contextLength)}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">工具调用</span>
                <span class="plaza-spec-item__value">${model.toolCalling ? "支持" : "不支持"}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">文件大小</span>
                <span class="plaza-spec-item__value">${sizeLabel || "—"}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">量化 ${renderPlazaHelpHint(plazaDoc("quantization").body, "量化")}</span>
                <span class="plaza-spec-item__value">${model.quantization ?? "—"}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">发布</span>
                <span class="plaza-spec-item__value">${model.releasedAt ?? "—"}</span>
              </div>
              <div class="plaza-spec-item">
                <span class="plaza-spec-item__label">许可</span>
                <span class="plaza-spec-item__value">${model.license ?? info?.license ?? "—"}</span>
              </div>
            </div>
          </section>
          ${rec
            ? html`
                <section class="plaza-detail-section">
                  <h3 class="plaza-detail-section__title">
                    本机适配
                    ${renderPlazaHelpHint(plazaDoc("tierList").body, "推荐等级")}
                  </h3>
                  <div class="emp-detail-meta-row">
                    <span>等级 <strong class=${tierClass(rec.tier)}>${rec.tier}</strong> · ${rec.tierLabel}</span>
                    ${rec.tokensPerSec > 0 ? html`<span>预估 ~${rec.tokensPerSec} tok/s ${renderPlazaHelpHint(plazaDoc("tokensPerSec").body, "tok/s")}</span>` : nothing}
                    <span>内存占用 ${rec.memoryPct}% ${renderPlazaHelpHint(plazaDoc("vram").body, "显存")}</span>
                    <span>综合 ${rec.score}/100</span>
                  </div>
                </section>
              `
            : nothing}
          <div class="emp-detail-info">
            <div class="emp-detail-meta-row plaza-detail-caps">${renderCapabilityIcons(model)}</div>
            ${model.hfUrl
              ? html`<div class="emp-detail-meta-row">
                  <a href=${model.hfUrl} target="_blank" rel="noopener noreferrer">HuggingFace</a>
                  ${model.ollamaName
                    ? html`<span>· Ollama: <code>ollama pull ${model.ollamaName}</code></span>`
                    : nothing}
                </div>`
              : nothing}
            ${model.lastError ? html`<p class="embedded-model-error">${model.lastError}</p>` : nothing}
          </div>
          <div class="market-card-actions">
            ${!model.installed
              ? model.downloadable === false
                ? html`
                    <p class="plaza-catalog-only-note">
                      此模型暂未配置<strong>内嵌 GGUF 下载</strong>，当前仅作性能推荐与目录展示。
                      ${model.hfUrl
                        ? html`
                            可前往
                            <a href=${model.hfUrl} target="_blank" rel="noopener noreferrer">HuggingFace</a>
                            查看权重来源。
                          `
                        : nothing}
                      ${model.ollamaName
                        ? html`
                            若本机已安装 Ollama，也可执行
                            <code>ollama pull ${model.ollamaName}</code>
                            ，并在「模型配置」中添加 Ollama Provider 使用。
                          `
                        : nothing}
                    </p>
                  `
                : html`
                    <button
                      class="btn primary"
                      type="button"
                      ?disabled=${props.loading || props.busyId === model.id}
                      @click=${() => props.onDownload(model.id)}
                    >下载内嵌模型</button>
                  `
              : model.running
                ? html`
                    ${model.kind !== "embedding"
                      ? html`
                          <button
                            class="btn primary"
                            type="button"
                            ?disabled=${props.busyId === model.id}
                            @click=${() => props.onChat(model)}
                          >测试对话</button>
                        `
                      : nothing}
                    <button
                      class="btn"
                      type="button"
                      ?disabled=${props.busyId === model.id}
                      @click=${() => props.onStop(model.id)}
                    >停止</button>
                  `
                : html`
                    <button
                      class="btn primary"
                      type="button"
                      ?disabled=${props.busyId === model.id}
                      @click=${() => props.onStart(model)}
                    >启动</button>
                    <button
                      class="btn btn--danger"
                      type="button"
                      ?disabled=${props.busyId === model.id}
                      @click=${() => props.onDelete(model)}
                    >删除</button>
                  `}
          </div>
        </div>
      </div>
    </div>
  `;
}

function renderPlazaChatThinking(thinking: string) {
  return html`
    <details class="chat-thinking chat-thinking--archive plaza-chat-msg__thinking">
      <summary class="chat-thinking__summary">
        <span class="chat-thinking__summary-icon" aria-hidden="true">${icons.brain}</span>
        <span class="chat-thinking__summary-label">思考过程</span>
      </summary>
      <div class="chat-thinking__content">${thinking}</div>
    </details>
  `;
}

function renderMultiModelPerfWarning(runningCount: number) {
  if (runningCount <= 0) {
    return nothing;
  }
  if (runningCount >= 2) {
    return html`
      <div class="plaza-perf-warning plaza-perf-warning--alert" role="status">
        当前同时运行 <strong>${runningCount}</strong> 个模型，会显著占用内存并降低推理速度。建议仅保留需要的模型运行。
      </div>
    `;
  }
  return html`
    <div class="plaza-perf-warning" role="status">
      已运行 1 个模型。同时启动多个模型会占用更多内存并降低性能，请按需启动。
    </div>
  `;
}

function renderPlazaChatTestModal(props: ModelPlazaProps) {
  const model = props.chatModel;
  if (!model) {
    return nothing;
  }
  const endpoint = resolveEmbeddedChatProxyURL(props.gatewayHost) ?? "—";
  const canSend = props.chatInput.trim().length > 0 && !props.chatLoading;

  return html`
    <div class="modal-overlay" @click=${props.onCloseChat}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large plaza-chat-test-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div class="emp-detail-title-wrap">
            <div class="emp-detail-logo emp-card__icon--default" aria-hidden="true">${icons.plazaChat}</div>
            <div>
              <h2 class="emp-detail-title">测试对话 · ${model.name}</h2>
              <p class="plaza-detail-sub plaza-chat-test-modal__model-id">
                模型 ID：<code>${model.id}</code>
              </p>
              <p class="plaza-detail-sub plaza-chat-test-modal__endpoint">${endpoint}</p>
            </div>
          </div>
          <button class="emp-detail-modal__close" type="button" aria-label="关闭" @click=${props.onCloseChat}>
            ${icons.x}
          </button>
        </div>
        <div class="emp-detail-modal__body plaza-chat-test-modal__body">
          <div class="plaza-chat-test-modal__messages" aria-live="polite">
            ${props.chatMessages.length === 0
              ? html`<p class="muted plaza-chat-test-modal__empty">发送消息测试内嵌模型推理接口（不进入主聊天）。</p>`
              : props.chatMessages.map(
                  (msg) => html`
                    <div class="plaza-chat-msg plaza-chat-msg--${msg.role}">
                      <span class="plaza-chat-msg__role">
                        ${msg.role === "user" ? "你" : "模型"}
                        ${msg.modelId
                          ? html`<span class="plaza-chat-msg__model-id">${msg.modelId}</span>`
                          : nothing}
                      </span>
                      ${msg.role === "assistant" && msg.thinking
                        ? renderPlazaChatThinking(msg.thinking)
                        : nothing}
                      <div class="plaza-chat-msg__content">${msg.content}</div>
                    </div>
                  `,
                )}
            ${props.chatLoading
              ? html`<div class="plaza-chat-msg plaza-chat-msg--assistant plaza-chat-msg--loading">
                  <span class="plaza-chat-msg__role">
                    模型
                    <span class="plaza-chat-msg__model-id">${model.id}</span>
                  </span>
                  <div class="plaza-chat-msg__content">${icons.loader2} 生成中…</div>
                </div>`
              : nothing}
          </div>
          ${props.chatError
            ? html`<p class="embedded-model-error plaza-chat-test-modal__error" role="alert">${props.chatError}</p>`
            : nothing}
          <form
            class="plaza-chat-test-modal__composer"
            @submit=${(e: Event) => {
              e.preventDefault();
              if (canSend) {
                props.onSendChat();
              }
            }}
          >
            <textarea
              class="plaza-chat-test-modal__input"
              rows="2"
              placeholder="输入测试消息…"
              .value=${props.chatInput}
              ?disabled=${props.chatLoading}
              @input=${(e: Event) => props.onChatInput((e.target as HTMLTextAreaElement).value)}
              @keydown=${(e: KeyboardEvent) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  if (canSend) {
                    props.onSendChat();
                  }
                }
              }}
            ></textarea>
            <button class="btn primary" type="submit" ?disabled=${!canSend}>发送</button>
          </form>
        </div>
      </div>
    </div>
  `;
}

export function renderModelPlaza(props: ModelPlazaProps) {
  const showPageHeader = props.showPageHeader ?? true;
  const searchQuery = props.searchQuery ?? "";
  const visibleModels = filterEmbeddedModelsByQuery(props.models, searchQuery);
  const hasInstalled = props.models.some((m) => m.installed);
  const hw = props.hardware ?? detectLocalHardware();
  const recommendations = resolveRecommendations(props, hw);
  const recById = new Map(recommendations.map((r) => [r.model.id, r]));
  const sortedModels = [...visibleModels].sort((a, b) => {
    const sa = recById.get(a.id)?.score ?? 0;
    const sb = recById.get(b.id)?.score ?? 0;
    if (sb !== sa) {
      return sb - sa;
    }
    const rank = (m: EmbeddedModelEntry) => (m.featured ? 2 : 0) + (m.builtin ? 1 : 0);
    return rank(b) - rank(a);
  });
  const installRecs = recommendations.filter((r) => r.model.downloadable !== false && r.tier !== "F");
  const bannerRecs = installRecs.length > 0 ? installRecs : recommendations.filter((r) => r.tier !== "F");
  const showRecommendBanner = !hasInstalled && !props.loading && bannerRecs.length > 0;
  const runningCount = props.models.filter((m) => m.running).length;
  const sortLabel = hw.serverSide ? "Gateway 服务器推荐得分" : "本机估算推荐得分";
  const pageTitle = props.pageTitle ?? "本机模型";

  const toolbarActions = html`
    <div class="${showPageHeader ? "emp-toolbar__actions" : "plaza-toolbar__actions"}">
      ${showPageHeader
        ? html`
            <div class="emp-search">
              <span class="input"><input
                class="emp-search__input"
                type="text"
                placeholder="搜索"
                .value=${searchQuery}
                ?disabled=${props.loading}
                @input=${(e: Event) => props.onSearchChange?.((e.target as HTMLInputElement).value)}
              /></span>
              <span class="emp-search__icon" aria-hidden="true">${icons.search}</span>
            </div>
          `
        : nothing}
      <button class="btn" type="button" ?disabled=${props.loading} @click=${props.onRefresh}>刷新</button>
      ${props.downloadingModelId
        ? html`
            <button class="btn btn--danger" type="button" @click=${props.onCancelDownload}>取消下载</button>
          `
        : nothing}
    </div>
  `;

  const introBlock = html`
    <div class="embedded-model-intro">
      <p class="embedded-model-intro__lead">
        共 ${visibleModels.length} 个模型，已按${sortLabel}排序，点击行查看详情。
      </p>
      <p class="embedded-model-intro__tips">
        优先选 S / A / B 级模型；详情见
        <button class="plaza-manual-import-link" type="button" @click=${props.onOpenRecommend}>
          推荐说明
        </button>，GGUF 手动导入见
        <button class="plaza-manual-import-link" type="button" @click=${props.onOpenManualImport}>
          手动导入说明
        </button>。
      </p>
    </div>
  `;

  return html`
    <main class="emp-page">
      <section class="emp-list-wrap">
        <div class="emp-content">
          <div class="emp-main">
            <div class="emp-main__body">
              ${showPageHeader
                ? toolbarActions
                : html`
                    <div class="plaza-toolbar">
                      ${introBlock}
                      ${toolbarActions}
                    </div>
                  `}
              ${showPageHeader
                ? html`
                    <div class="emp-sections">
                      <div class="emp-section">
                        <div class="emp-section__header">
                          <h3 class="emp-section__title">${pageTitle}</h3>
                        </div>
                        ${introBlock}
                        ${props.error ? html`<div class="emp-empty embedded-model-error">${props.error}</div>` : nothing}
                        ${renderMultiModelPerfWarning(runningCount)}
                        ${props.loading && props.models.length === 0
                          ? html`<div class="emp-loading">加载中…</div>`
                          : nothing}
                        ${!props.loading && visibleModels.length === 0
                          ? html`<div class="emp-empty">暂无匹配的模型</div>`
                          : nothing}
                        ${showRecommendBanner ? renderRecommendBanner(props, bannerRecs) : nothing}
                        ${visibleModels.length > 0
                          ? html`
                              <div class="plaza-list">
                                ${renderPlazaListHeader()}
                                ${sortedModels.map((m) => renderModelRow(props, m, recById.get(m.id)))}
                              </div>
                            `
                          : nothing}
                      </div>
                    </div>
                  `
                : html`
                    ${props.error ? html`<div class="emp-empty embedded-model-error">${props.error}</div>` : nothing}
                    ${renderMultiModelPerfWarning(runningCount)}
                    ${props.loading && props.models.length === 0
                      ? html`<div class="emp-loading">加载中…</div>`
                      : nothing}
                    ${!props.loading && visibleModels.length === 0
                      ? html`<div class="emp-empty">暂无匹配的模型</div>`
                      : nothing}
                    ${showRecommendBanner ? renderRecommendBanner(props, bannerRecs) : nothing}
                    ${visibleModels.length > 0
                      ? html`
                          <div class="plaza-list">
                            ${renderPlazaListHeader()}
                            ${sortedModels.map((m) => renderModelRow(props, m, recById.get(m.id)))}
                          </div>
                        `
                      : nothing}
                  `}
            </div>
          </div>
        </div>
      </section>
    </main>
    ${renderRecommendGuideModal(props)}
    ${renderPlazaManualImportModal({ open: props.manualImportOpen, onClose: props.onCloseManualImport })}
    ${renderDetailModal(props)}
    ${renderPlazaChatTestModal(props)}
  `;
}

/** Compact card for the local models section in model library. */
export function renderEmbeddedLocalCard(props: {
  model: EmbeddedModelEntry;
  busyId: string | null;
  onStart: (model: EmbeddedModelEntry) => void;
  onStop: (modelId: string) => void;
}) {
  const { model } = props;
  const isBusy = props.busyId === model.id;
  return html`
    <div class="emp-card-wrap ${model.running ? "active" : ""}">
      <div class="emp-card emp-card-btn">
        <div class="emp-card__icon emp-card__icon--default" aria-hidden="true">${icons.modelCube}</div>
        <div class="emp-card__actions models-provider-actions">
          ${model.running
            ? html`
                <button
                  class="btn btn--sm"
                  type="button"
                  ?disabled=${isBusy}
                  @click=${(e: Event) => {
                    e.stopPropagation();
                    props.onStop(model.id);
                  }}
                >停止</button>
                <span class="market-card-chip market-card-chip--state">运行中</span>
              `
            : html`
                <button
                  class="btn btn--sm primary"
                  type="button"
                  ?disabled=${isBusy}
                  @click=${(e: Event) => {
                    e.stopPropagation();
                    props.onStart(model);
                  }}
                >启动</button>
              `}
        </div>
        <h3 class="emp-card__title">${model.name}</h3>
        <p class="emp-card__desc">本地 GGUF · ${formatContextLength(model.contextLength)}</p>
        <div class="market-card-meta">
          ${model.running ? html`<span>运行中</span>` : html`<span>已安装</span>`}
        </div>
      </div>
    </div>
  `;
}

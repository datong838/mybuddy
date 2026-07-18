import { customElement, property, state } from "lit/decorators.js";
import { LitElement, html, nothing } from "lit";
import {
  renderBusyDesk,
  renderReadingProps,
  renderSvgDefs,
  renderTorsoCover,
} from "./openocta-mascot-scenes.ts";

const POKE_MESSAGES = [
  "我在呢～",
  "触手有点忙！",
  "再戳我就翻书啦",
  "八爪鱼式思考中…",
  "这页好看吗？",
  "书中自有黄金屋～",
];

type Point = { x: number; y: number };

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function mascotAsset(base: string, name: string) {
  const prefix = base ? base.replace(/\/$/, "") : "";
  return `${prefix}/${encodeURIComponent(name)}`;
}

/**
 * Floating OpenOcta octopus — idle: reading; busy: typing on laptop.
 */
@customElement("openocta-mascot")
export class OpenOctaMascot extends LitElement {
  createRenderRoot() {
    return this;
  }

  @property({ type: Boolean }) busy = false;
  @property({ attribute: "asset-base" }) assetBase = "";

  @state() private poke = false;
  @state() private bubble: string | null = null;
  @state() private dragging = false;
  @state() private pos: Point | null = null;

  #dragOffset: Point = { x: 0, y: 0 };
  #bubbleTimer: ReturnType<typeof setTimeout> | null = null;
  #dragMoved = false;

  disconnectedCallback() {
    super.disconnectedCallback();
    this.#clearBubbleTimer();
    this.#stopDragListeners();
  }

  #clearBubbleTimer() {
    if (this.#bubbleTimer) {
      clearTimeout(this.#bubbleTimer);
      this.#bubbleTimer = null;
    }
  }

  #showBubble(text: string) {
    this.#clearBubbleTimer();
    this.bubble = text;
    this.#bubbleTimer = setTimeout(() => {
      this.bubble = null;
      this.#bubbleTimer = null;
    }, 2200);
  }

  #onPointerDown(event: PointerEvent) {
    const target = event.currentTarget as HTMLElement;
    if (event.button !== 0) return;
    const rect = target.getBoundingClientRect();
    this.#dragOffset = { x: event.clientX - rect.left, y: event.clientY - rect.top };
    this.#dragMoved = false;
    this.dragging = true;
    target.setPointerCapture(event.pointerId);
    this.#startDragListeners();
    event.preventDefault();
  }

  #onPointerMove = (event: PointerEvent) => {
    if (!this.dragging) return;
    this.#dragMoved = true;
    const shell = this.closest(".shell") as HTMLElement | null;
    const bounds = shell?.getBoundingClientRect() ?? document.body.getBoundingClientRect();
    const size = 156;
    const margin = 12;
    const x = clamp(event.clientX - this.#dragOffset.x - bounds.left, margin, bounds.width - size - margin);
    const y = clamp(event.clientY - this.#dragOffset.y - bounds.top, margin, bounds.height - size - margin);
    this.pos = { x, y };
  };

  #onPointerUp = () => {
    if (!this.dragging) return;
    this.dragging = false;
    this.#stopDragListeners();
  };

  #startDragListeners() {
    window.addEventListener("pointermove", this.#onPointerMove);
    window.addEventListener("pointerup", this.#onPointerUp);
    window.addEventListener("pointercancel", this.#onPointerUp);
  }

  #stopDragListeners() {
    window.removeEventListener("pointermove", this.#onPointerMove);
    window.removeEventListener("pointerup", this.#onPointerUp);
    window.removeEventListener("pointercancel", this.#onPointerUp);
  }

  #onClick(event: MouseEvent) {
    if (this.#dragMoved) return;
    event.stopPropagation();
    this.poke = true;
    window.setTimeout(() => {
      this.poke = false;
    }, 520);

    if (this.busy) {
      this.#showBubble("敲键盘中，稍等～");
      return;
    }
    const msg = POKE_MESSAGES[Math.floor(Math.random() * POKE_MESSAGES.length)];
    this.#showBubble(msg);
  }

  render() {
    const pngSrc = mascotAsset(this.assetBase, "玩偶形象设计.png");
    const style = this.pos
      ? `left:${this.pos.x}px;top:${this.pos.y}px;right:auto;bottom:auto;`
      : "";

    const activityLabel = this.busy ? "敲键盘中" : "看书";

    return html`
      <div
        class="octo-mascot ${this.busy ? "octo-mascot--busy" : "octo-mascot--idle"} ${this.poke ? "octo-mascot--poke" : ""} ${this.dragging ? "octo-mascot--dragging" : ""}"
        style=${style}
        role="img"
        aria-label=${this.busy ? "OpenOcta 八爪鱼助手正在敲键盘" : "OpenOcta 八爪鱼助手正在看书"}
        @pointerdown=${this.#onPointerDown}
        @click=${this.#onClick}
      >
        ${this.bubble ? html`<div class="octo-mascot__bubble">${this.bubble}</div>` : nothing}

        <div class="octo-mascot__activity">${activityLabel}</div>

        <div class="octo-mascot__shadow" aria-hidden="true"></div>

        <div class="octo-mascot__figure">
          ${renderSvgDefs()}

          <div class="octo-mascot__art-wrap">
            <img class="octo-mascot__art" src=${pngSrc} alt="" draggable="false" />
          </div>

          ${this.busy ? renderTorsoCover() : nothing}

          ${
            this.busy
              ? html`<div class="octo-mascot__stage octo-mascot__busy">${renderBusyDesk()}</div>`
              : html`
                  <div class="octo-mascot__stage octo-mascot__idle-fx">${renderReadingProps()}</div>
                `
          }
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "openocta-mascot": OpenOctaMascot;
  }
}

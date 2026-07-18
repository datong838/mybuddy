import { html, type TemplateResult } from "lit";
import { icons } from "../icons.js";

function positionPlazaHint(hint: HTMLElement, tooltip: HTMLElement) {
  const rect = hint.getBoundingClientRect();
  const margin = 8;
  tooltip.classList.add("is-visible");

  const tooltipWidth = tooltip.offsetWidth;
  const tooltipHeight = tooltip.offsetHeight;

  let left = rect.left;
  if (left + tooltipWidth > window.innerWidth - margin) {
    left = rect.right - tooltipWidth;
  }
  if (left < margin) {
    left = margin;
  }

  let top = rect.bottom + 6;
  if (top + tooltipHeight > window.innerHeight - margin) {
    top = Math.max(margin, rect.top - tooltipHeight - 6);
  }

  tooltip.style.left = `${left}px`;
  tooltip.style.top = `${top}px`;
}

function showPlazaHint(e: Event) {
  const hint = (e.currentTarget as HTMLElement).closest(".plaza-help-hint");
  if (!hint) {
    return;
  }
  const tooltip = hint.querySelector(".plaza-help-hint__tooltip") as HTMLElement | null;
  if (!tooltip) {
    return;
  }
  positionPlazaHint(hint, tooltip);
}

function hidePlazaHint(e: Event) {
  const hint = (e.currentTarget as HTMLElement).closest(".plaza-help-hint");
  if (!hint) {
    return;
  }
  const tooltip = hint.querySelector(".plaza-help-hint__tooltip") as HTMLElement | null;
  if (!tooltip) {
    return;
  }
  window.setTimeout(() => {
    if (!hint.matches(":hover") && !tooltip.matches(":hover")) {
      tooltip.classList.remove("is-visible");
    }
  }, 100);
}

/** 带悬浮说明的问号图标，tooltip 使用 fixed 定位以跳出 modal 裁剪 */
export function renderPlazaHelpHint(
  text: string,
  label?: string,
  options?: { nowrap?: boolean },
): TemplateResult {
  const tooltipClass = options?.nowrap
    ? "plaza-help-hint__tooltip plaza-help-hint__tooltip--nowrap"
    : "plaza-help-hint__tooltip";
  return html`
    <span
      class="plaza-help-hint"
      aria-label=${label ?? "说明"}
      @mouseenter=${showPlazaHint}
      @mouseleave=${hidePlazaHint}
    >
      ${icons.helpCircle}
      <span
        class=${tooltipClass}
        @mouseenter=${showPlazaHint}
        @mouseleave=${hidePlazaHint}
      >${text}</span>
    </span>
  `;
}

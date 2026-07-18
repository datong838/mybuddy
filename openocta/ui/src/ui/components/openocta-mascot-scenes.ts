import { html, svg } from "lit";

/** SVG defs: white-bg knockout only */
export function renderSvgDefs() {
  return svg`
    <svg class="octo-mascot__filters" aria-hidden="true" width="0" height="0">
      <defs>
        <filter id="octo-knockout" color-interpolation-filters="sRGB">
          <feColorMatrix
            type="matrix"
            values="
              1 0 0 0 0
              0 1 0 0 0
              0 0 1 0 0
              -1.12 -1.12 -1.12 3.05 0"
          />
        </filter>
      </defs>
    </svg>
  `;
}

/** Opaque cover over book + gripping arms while typing */
export function renderTorsoCover() {
  return html`
    <div class="octo-mascot__torso-cover" aria-hidden="true">
      <div class="octo-mascot__torso-core"></div>
      <div class="octo-mascot__torso-shine"></div>
      <div class="octo-mascot__torso-arm octo-mascot__torso-arm--left"></div>
      <div class="octo-mascot__torso-arm octo-mascot__torso-arm--right"></div>
    </div>
  `;
}

const FINGER_SLOTS = [
  { left: "31%", delay: "0s" },
  { left: "41%", delay: "-0.03s" },
  { left: "51%", delay: "-0.06s" },
  { left: "61%", delay: "-0.02s" },
] as const;

export function renderBusyDesk() {
  const keys = Array.from({ length: 10 }, (_, i) => i);

  return html`
    <div class="octo-mascot__desk" aria-hidden="true">
      <div class="octo-mascot__laptop">
        <div class="octo-mascot__laptop-lid">
          <div class="octo-mascot__laptop-screen">
            <span class="octo-mascot__code-line"></span>
            <span class="octo-mascot__code-line octo-mascot__code-line--mid"></span>
            <span class="octo-mascot__code-line octo-mascot__code-line--short"></span>
            <span class="octo-mascot__cursor">▍</span>
          </div>
        </div>
        <div class="octo-mascot__laptop-base-plate">
          <div class="octo-mascot__keyboard">
            ${keys.map(
              (i) => html`
                <span class="octo-mascot__key" style="--ki:${i}"></span>
              `,
            )}
          </div>
          <div class="octo-mascot__trackpad"></div>
        </div>
      </div>
      <div class="octo-mascot__fingers">
        ${FINGER_SLOTS.map(
          (f, i) => html`
            <span
              class="octo-mascot__finger"
              style="--fi:${i};left:${f.left};animation-delay:${f.delay}"
            ></span>
          `,
        )}
      </div>
    </div>
  `;
}

export function renderReadingProps() {
  return html`
    <div class="octo-mascot__props octo-mascot__props--read" aria-hidden="true">
      <span class="octo-mascot__page-flip"></span>
    </div>
  `;
}

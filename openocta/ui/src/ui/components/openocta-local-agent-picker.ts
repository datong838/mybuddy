import { customElement, property, state } from "lit/decorators.js";
import { LitElement, html, nothing } from "lit";
import { icons } from "../icons.ts";
import {
  localAgentInitial,
  primaryMentionAlias,
  sortLocalAgents,
  type LocalAgentProbeResult,
} from "../local-agents.ts";
import { t } from "../strings.js";

/**
 * Compact dropdown for picking local CLI agents in the chat compose bar.
 */
@customElement("openocta-local-agent-picker")
export class OpenOctaLocalAgentPicker extends LitElement {
  createRenderRoot() {
    return this;
  }

  @property({ attribute: false }) agents: LocalAgentProbeResult[] = [];
  @property({ type: Boolean }) disabled = false;

  @state() private open = false;

  #docClickHandler = (event: Event) => {
    if (!this.open) {
      return;
    }
    const target = event.target;
    if (target instanceof Node && this.contains(target)) {
      return;
    }
    this.open = false;
  };

  connectedCallback(): void {
    super.connectedCallback();
    document.addEventListener("click", this.#docClickHandler, true);
  }

  disconnectedCallback(): void {
    document.removeEventListener("click", this.#docClickHandler, true);
    super.disconnectedCallback();
  }

  private installedAgents(): LocalAgentProbeResult[] {
    return sortLocalAgents(this.agents).filter((a) => a.installed);
  }

  private toggleOpen(event: Event) {
    event.stopPropagation();
    if (this.disabled || this.installedAgents().length === 0) {
      return;
    }
    this.open = !this.open;
  }

  private pick(agent: LocalAgentProbeResult) {
    this.open = false;
    this.dispatchEvent(
      new CustomEvent("agent-insert", {
        detail: { mention: `@${primaryMentionAlias(agent)} ` },
        bubbles: true,
        composed: true,
      }),
    );
  }

  render() {
    const installed = this.installedAgents();
    if (installed.length === 0) {
      return nothing;
    }

    return html`
      <div class="local-agent-picker">
        <button
          type="button"
          class="chat-compose__chip local-agent-picker__trigger ${this.open ? "chat-compose__chip--active" : ""}"
          aria-label=${t("chatLocalAgentsPicker")}
          aria-expanded=${this.open ? "true" : "false"}
          title=${t("chatLocalAgentsHint")}
          ?disabled=${this.disabled}
          @click=${this.toggleOpen}
        >
          <span class="chat-compose__chip-icon" aria-hidden="true">${icons.users}</span>
          <span class="chat-compose__chip-label">${t("chatLocalAgentsPicker")}</span>
          <span class="local-agent-picker__chevron" aria-hidden="true">${icons.arrowDown}</span>
        </button>
        ${
          this.open
            ? html`
                <div class="local-agent-picker__menu" role="listbox" @click=${(e: Event) => e.stopPropagation()}>
                  ${installed.map(
                    (agent) => html`
                      <button
                        type="button"
                        class="local-agent-picker__item"
                        role="option"
                        title=${agent.invokeHint}
                        @click=${() => this.pick(agent)}
                      >
                        <span class="local-agent-chip__avatar local-agent-chip__avatar--sm" aria-hidden="true"
                          >${localAgentInitial(agent.id)}</span
                        >
                        <span class="local-agent-picker__item-name">${agent.label}</span>
                        <span class="local-agent-picker__item-alias muted">@${primaryMentionAlias(agent)}</span>
                      </button>
                    `,
                  )}
                </div>
              `
            : nothing
        }
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "openocta-local-agent-picker": OpenOctaLocalAgentPicker;
  }
}

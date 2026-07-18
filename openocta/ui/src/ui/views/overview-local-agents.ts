import { html, nothing } from "lit";
import { t } from "../strings.js";
import {
  localAgentInitial,
  primaryMentionAlias,
  sortLocalAgents,
  type LocalAgentProbeResult,
} from "../local-agents.ts";

export type OverviewLocalAgentsProps = {
  loading: boolean;
  error: string | null;
  agents: LocalAgentProbeResult[];
};

export function renderOverviewLocalAgents(props: OverviewLocalAgentsProps) {
  const agents = sortLocalAgents(props.agents);

  return html`
    <section class="card" style="margin-top: 16px;">
      <div class="card-title">${t("overviewLocalAgents")}</div>
      <div class="card-sub">${t("overviewLocalAgentsSub")}</div>
      ${
        props.error
          ? html`<div class="callout danger" style="margin-top: 14px;">${props.error}</div>`
          : nothing
      }
      ${
        props.loading && agents.length === 0
          ? html`<div class="muted" style="margin-top: 14px; padding: 12px;">${t("overviewLocalAgentsLoading")}</div>`
          : html`
              <div class="local-agents-grid" style="margin-top: 14px;">
                ${agents.map((agent) => renderLocalAgentCard(agent))}
              </div>
            `
      }
    </section>
  `;
}

function renderLocalAgentCard(agent: LocalAgentProbeResult) {
  const missing = !agent.installed;
  return html`
    <div
      class="local-agent-card ${missing ? "local-agent-card--missing" : "local-agent-card--ok"}"
      title=${agent.probeMethod}
    >
      <div class="local-agent-card__head">
        <span class="local-agent-card__avatar" aria-hidden="true">${localAgentInitial(agent.id)}</span>
        <div>
          <div class="local-agent-card__title">${agent.label}</div>
          <div class="local-agent-card__status muted">
            ${missing ? t("overviewLocalAgentMissing") : t("overviewLocalAgentReady")}
          </div>
        </div>
      </div>
      ${
        !missing && agent.version
          ? html`<div class="local-agent-card__meta muted">${agent.version}</div>`
          : nothing
      }
      ${
        !missing && agent.path
          ? html`<div class="local-agent-card__path muted">${agent.path}</div>`
          : nothing
      }
      <div class="local-agent-card__probe muted">
        <span class="local-agent-card__probe-label">${t("overviewLocalAgentProbe")}</span>
        ${agent.probeMethod}
      </div>
      <div class="local-agent-card__invoke muted">
        <span class="local-agent-card__probe-label">${t("overviewLocalAgentInvoke")}</span>
        <code>${agent.invokeHint}</code>
      </div>
      <div class="local-agent-card__mention muted">
        ${t("overviewLocalAgentMention")}: @${primaryMentionAlias(agent)}
      </div>
    </div>
  `;
}

export function renderLocalAgentChips(
  agents: LocalAgentProbeResult[],
  opts: {
    onInsert: (mention: string) => void;
    compact?: boolean;
  },
) {
  const sorted = sortLocalAgents(agents);
  if (sorted.length === 0) {
    return nothing;
  }
  return html`
    <div class="local-agent-chips ${opts.compact ? "local-agent-chips--compact" : ""}">
      ${sorted.map(
        (agent) => html`
          <button
            type="button"
            class="local-agent-chip ${agent.installed ? "" : "local-agent-chip--missing"}"
            ?disabled=${!agent.installed}
            title=${agent.installed ? agent.invokeHint : agent.probeMethod}
            @click=${() => opts.onInsert(`@${primaryMentionAlias(agent)} `)}
          >
            <span class="local-agent-chip__avatar" aria-hidden="true">${localAgentInitial(agent.id)}</span>
            <span class="local-agent-chip__label">${agent.label}</span>
          </button>
        `,
      )}
    </div>
  `;
}

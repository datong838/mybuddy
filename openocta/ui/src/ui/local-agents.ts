export type LocalAgentProbeResult = {
  id: string;
  label: string;
  installed: boolean;
  path?: string;
  version?: string;
  probeMethod: string;
  invokeHint: string;
  aliases: string[];
};

export type LocalAgentsProbeReport = {
  agents: LocalAgentProbeResult[];
  probedAt: number;
};

export const LOCAL_AGENT_ORDER = [
  "openclaw",
  "hermes",
  "cursor",
  "codex",
  "opencode",
  "trae",
] as const;

export function localAgentInitial(id: string): string {
  return id.slice(0, 1).toUpperCase();
}

export function primaryMentionAlias(agent: LocalAgentProbeResult): string {
  return agent.aliases[0] ?? agent.id;
}

export function sortLocalAgents(agents: LocalAgentProbeResult[]): LocalAgentProbeResult[] {
  const order = new Map(LOCAL_AGENT_ORDER.map((id, i) => [id, i]));
  return [...agents].sort((a, b) => (order.get(a.id as typeof LOCAL_AGENT_ORDER[number]) ?? 99) - (order.get(b.id as typeof LOCAL_AGENT_ORDER[number]) ?? 99));
}

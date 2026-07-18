import type { LocalAgentsProbeReport } from "../local-agents.ts";

export type LocalAgentsState = {
  client: { request: (method: string, params?: unknown) => Promise<unknown> } | null;
  connected: boolean;
  localAgentsReport: LocalAgentsProbeReport | null;
  localAgentsLoading: boolean;
  localAgentsError: string | null;
};

export async function loadLocalAgents(state: LocalAgentsState, opts?: { force?: boolean }) {
  if (!state.client || !state.connected) {
    return;
  }
  state.localAgentsLoading = true;
  state.localAgentsError = null;
  try {
    const res = (await state.client.request("localAgents.probe", {
      force: Boolean(opts?.force),
    })) as LocalAgentsProbeReport;
    state.localAgentsReport = res;
  } catch (err) {
    state.localAgentsError = String(err);
  } finally {
    state.localAgentsLoading = false;
  }
}

export async function loadLocalAgentsStatus(state: LocalAgentsState) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    const res = (await state.client.request("localAgents.status", {})) as LocalAgentsProbeReport;
    state.localAgentsReport = res;
  } catch {
    // status is best-effort; probe on overview refresh
  }
}

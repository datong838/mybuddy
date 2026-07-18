import type { GatewayBrowserClient } from "../gateway.ts";
import type { CronJob, CronRunLogEntry, CronStatus } from "../types.ts";
import type { CronFormState } from "../ui-types.ts";
import { toNumber } from "../format.ts";
import { applyScheduleToForm, buildAtCronExpr } from "../cron/cron-schedule.ts";

export type CronState = {
  client: GatewayBrowserClient | null;
  connected: boolean;
  cronLoading: boolean;
  cronJobs: CronJob[];
  cronStatus: CronStatus | null;
  cronError: string | null;
  cronForm: CronFormState;
  cronRunsJobId: string | null;
  cronRuns: CronRunLogEntry[];
  cronBusy: boolean;
};

function normalizeEmployeeIdForCron(raw: string): string {
  let s = (raw ?? "").trim();
  if (!s) return "";
  if (s.toLowerCase().startsWith("local:")) {
    s = s.slice("local:".length);
  }
  s = s.replaceAll(":", "-");
  return s.trim().toLowerCase();
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

function toLocalDateTimeInputValue(ms: number): string {
  const d = new Date(ms);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

export async function loadCronStatus(state: CronState) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    const res = await state.client.request<CronStatus>("cron.status", {});
    state.cronStatus = res;
  } catch (err) {
    state.cronError = String(err);
  }
}

export async function loadCronJobs(state: CronState) {
  if (!state.client || !state.connected) {
    return;
  }
  if (state.cronLoading) {
    return;
  }
  state.cronLoading = true;
  state.cronError = null;
  try {
    const res = await state.client.request<{ jobs?: Array<CronJob> }>("cron.list", {
      includeDisabled: true,
    });
    state.cronJobs = Array.isArray(res.jobs) ? res.jobs : [];
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronLoading = false;
  }
}

export function buildCronSchedule(form: CronFormState) {
  if (form.scheduleKind === "every") {
    const amount = toNumber(form.everyAmount, 0);
    if (amount <= 0) {
      throw new Error("Invalid interval amount.");
    }
    const unit = form.everyUnit;
    const mult = unit === "minutes" ? 60_000 : unit === "hours" ? 3_600_000 : 86_400_000;
    return { kind: "every" as const, everyMs: amount * mult };
  }
  if (form.scheduleKind === "at") {
    const expr = buildAtCronExpr(form);
    return { kind: "cron" as const, expr, tz: form.cronTz.trim() || undefined };
  }
  const expr = form.cronExpr.trim();
  if (!expr) {
    throw new Error("Cron expression required.");
  }
  return { kind: "cron" as const, expr, tz: form.cronTz.trim() || undefined };
}

export function buildCronRunConfig(form: CronFormState) {
  const modelRef = form.modelRef.trim();
  const skillKeys = form.skillKeys.filter(Boolean);
  const mcpServers = form.mcpServers.filter(Boolean);
  let extraParams: Record<string, unknown> | undefined;
  const rawExtra = form.extraParamsJson.trim();
  if (rawExtra && rawExtra !== "{}") {
    try {
      const parsed = JSON.parse(rawExtra) as unknown;
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        extraParams = parsed as Record<string, unknown>;
      }
    } catch {
      throw new Error("附加参数 JSON 格式无效。");
    }
  }
  if (!modelRef && skillKeys.length === 0 && mcpServers.length === 0 && !extraParams) {
    return {};
  }
  return {
    modelRef: modelRef || undefined,
    skillKeys: skillKeys.length ? skillKeys : undefined,
    mcpServers: mcpServers.length ? mcpServers : undefined,
    extraParams,
  };
}

function buildCronDelivery(form: CronFormState) {
  const channel = form.channel.trim();
  if (!channel) {
    return undefined;
  }
  const to = form.deliveryTo.trim();
  if (!to) {
    throw new Error("选择 IM 通道后，请填写群聊/用户 ID（如飞书 oc_xxx、ou_xxx）。");
  }
  return {
    mode: "announce" as const,
    channel,
    to,
  };
}

function buildCronPayload(form: CronFormState) {
  const message = form.payloadText.trim();
  if (!message) {
    throw new Error("Agent message required.");
  }
  return { kind: "agentTurn" as const, message };
}

export function cronFormFromJob(job: CronJob, prev: CronFormState): CronFormState {
  const next: CronFormState = { ...prev };
  next.name = job.name ?? "";
  next.description = job.description ?? "";
  next.digitalEmployeeId = normalizeEmployeeIdForCron(job.digitalEmployeeId ?? "");
  next.enabled = !!job.enabled;
  next.payloadText =
    job.payload?.kind === "agentTurn"
      ? String(job.payload.message ?? "")
      : String((job.payload as { text?: string })?.text ?? "");
  next.channel = job.delivery?.channel && job.delivery.channel !== "last" ? job.delivery.channel : "";
  next.deliveryTo = job.delivery?.to ?? "";
  next.modelRef = job.runConfig?.modelRef ?? "";
  next.skillKeys = [...(job.runConfig?.skillKeys ?? [])];
  next.mcpServers = [...(job.runConfig?.mcpServers ?? [])];
  next.extraParamsJson =
    job.runConfig?.extraParams && Object.keys(job.runConfig.extraParams).length > 0
      ? JSON.stringify(job.runConfig.extraParams, null, 2)
      : "{}";
  return applyScheduleToForm(job.schedule, next);
}

function runConfigForSave(form: CronFormState) {
  const rc = buildCronRunConfig(form);
  if (!rc.modelRef && !(rc.skillKeys?.length ?? 0) && !(rc.mcpServers?.length ?? 0) && !rc.extraParams) {
    return undefined;
  }
  return rc;
}

function buildCronJobBody(form: CronFormState) {
  const schedule = buildCronSchedule(form);
  const payload = buildCronPayload(form);
  const delivery = buildCronDelivery(form);
  const runConfig = runConfigForSave(form);
  const digitalEmployeeId = normalizeEmployeeIdForCron(form.digitalEmployeeId.trim());
  const job = {
    name: form.name.trim(),
    description: form.description.trim() || undefined,
    digitalEmployeeId: digitalEmployeeId || undefined,
    enabled: form.enabled,
    schedule,
    sessionTarget: "isolated" as const,
    wakeMode: "now" as const,
    payload,
    delivery,
    runConfig,
  };
  if (!job.name) {
    throw new Error("Name required.");
  }
  return job;
}

export async function addCronJob(state: CronState) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    const job = buildCronJobBody(state.cronForm);
    await state.client.request("cron.add", job);
    state.cronForm = {
      ...state.cronForm,
      name: "",
      description: "",
      payloadText: "",
    };
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function updateCronJob(state: CronState, jobId: string) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    const body = buildCronJobBody(state.cronForm);
    const patch = {
      enabled: body.enabled,
      name: body.name,
      description: body.description ?? "",
      digitalEmployeeId: body.digitalEmployeeId ?? "",
      schedule: body.schedule,
      sessionTarget: body.sessionTarget,
      wakeMode: body.wakeMode,
      payload: body.payload,
      delivery: body.delivery,
      runConfig: buildCronRunConfig(state.cronForm),
    };
    await state.client.request("cron.update", { id: jobId, patch });
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function toggleCronJob(state: CronState, job: CronJob, enabled: boolean) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    await state.client.request("cron.update", { id: job.id, patch: { enabled } });
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function runCronJob(state: CronState, job: CronJob) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    await state.client.request("cron.run", { id: job.id, mode: "force" });
    await loadCronRuns(state, job.id);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function removeCronJob(state: CronState, job: CronJob) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    await state.client.request("cron.remove", { id: job.id });
    if (state.cronRunsJobId === job.id) {
      state.cronRunsJobId = null;
      state.cronRuns = [];
    }
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function loadCronRuns(state: CronState, jobId: string) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    const res = await state.client.request<{ entries?: Array<CronRunLogEntry> }>("cron.runs", {
      id: jobId,
      limit: 50,
    });
    state.cronRunsJobId = jobId;
    state.cronRuns = Array.isArray(res.entries) ? res.entries : [];
  } catch (err) {
    state.cronError = String(err);
  }
}

export { toLocalDateTimeInputValue };

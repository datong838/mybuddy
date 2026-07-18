import { html, nothing } from "lit";
import type { ChannelUiMetaEntry, CronJob, CronRunLogEntry, CronStatus, SkillStatusEntry } from "../types.ts";
import type { CronFormState } from "../ui-types.ts";
import { formatAgo, formatMs } from "../format.ts";
import { icons } from "../icons.js";
import { nativeConfirm } from "../native-dialog-bridge.ts";
import { pathForTab } from "../navigation.ts";
import { cronRunHistorySessionKey } from "../sessions/session-key-utils.js";
import { formatCronSchedule } from "../presenter.ts";
import { t } from "../strings.js";
import { quartzPreviewFromAtForm } from "../cron/cron-schedule.ts";

export type CronModelOption = { value: string; label: string };
export type CronMcpOption = { key: string; label: string };

export type CronProps = {
  basePath: string;
  loading: boolean;
  status: CronStatus | null;
  jobs: CronJob[];
  error: string | null;
  busy: boolean;
  form: CronFormState;
  addModalOpen: boolean;
  editModalOpen?: boolean;
  editJobId?: string | null;
  digitalEmployees?: Array<{ id: string; name?: string; enabled?: boolean }>;
  digitalEmployeesLoading?: boolean;
  modelOptions?: CronModelOption[];
  skillOptions?: SkillStatusEntry[];
  mcpOptions?: CronMcpOption[];
  channels: string[];
  channelLabels?: Record<string, string>;
  channelMeta?: ChannelUiMetaEntry[];
  runsJobId: string | null;
  runs: CronRunLogEntry[];
  onFormChange: (patch: Partial<CronFormState>) => void;
  onRefresh: () => void;
  onOpenAddModal: () => void;
  onCloseAddModal: () => void;
  onAdd: () => void;
  onOpenEditModal?: (job: CronJob) => void;
  onCloseEditModal?: () => void;
  onUpdate?: (jobId: string) => void;
  onToggle: (job: CronJob, enabled: boolean) => void;
  onRun: (job: CronJob) => void;
  onRemove: (job: CronJob) => void;
  confirmRemove?: boolean;
  onLoadRuns: (jobId: string) => void;
  onShowHistory?: (jobId: string) => void;
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

function toggleListItem(list: string[], value: string, checked: boolean): string[] {
  const set = new Set(list);
  if (checked) {
    set.add(value);
  } else {
    set.delete(value);
  }
  return [...set];
}

function buildChannelOptions(props: CronProps): string[] {
  const options = props.channels.filter(Boolean);
  const current = props.form.channel?.trim();
  if (current && !options.includes(current)) {
    options.push(current);
  }
  return [...new Set(options)];
}

function resolveChannelLabel(props: CronProps, channel: string): string {
  const meta = props.channelMeta?.find((entry) => entry.id === channel);
  if (meta?.label) {
    return meta.label;
  }
  return props.channelLabels?.[channel] ?? channel;
}

function formatCronStatDateTime(ms?: number | null): string {
  if (!ms) {
    return "n/a";
  }
  const date = new Date(ms);
  if (Number.isNaN(date.getTime())) {
    return "n/a";
  }
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

const WEEKDAY_LABELS = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];

export function renderCronConfig(props: CronProps) {
  return html`
    <section class="stack cron-config-stack">
      <div class="card">
        <div class="card-title">${t("cronScheduler")}</div>
        <div class="card-sub">${t("cronSchedulerSub")}</div>
        <div class="stat-grid">
          <div class="stat">
            <div class="stat-label">${t("cronEnabled")}</div>
            <div class="stat-value">
              ${props.status ? (props.status.enabled ? t("commonYes") : t("commonNo")) : t("commonNA")}
            </div>
          </div>
          <div class="stat">
            <div class="stat-label">${t("cronJobs")}</div>
            <div class="stat-value">${props.status?.jobs ?? t("commonNA")}</div>
          </div>
          <div class="stat">
            <div class="stat-label">${t("overviewCronNext")}</div>
            <div class="stat-value">${formatCronStatDateTime(props.status?.nextWakeAtMs ?? null)}</div>
          </div>
        </div>
        ${props.error ? html`<div class="muted" style="margin-top: 12px;">${props.error}</div>` : nothing}
      </div>
    </section>

    <section class="card cron-jobs-card">
      ${
        props.jobs.length === 0
          ? html`<div class="muted">${t("cronNoJobsYet")}</div>`
          : html`
              <div class="list">
                ${props.jobs.map((job) => renderJob(job, props, { mode: "config" }))}
              </div>
            `
      }
    </section>
    ${props.addModalOpen ? renderCronAddModal(props) : nothing}
    ${props.editModalOpen && props.editJobId ? renderCronEditModal(props, props.editJobId) : nothing}
  `;
}

function renderCronAddModal(props: CronProps) {
  return html`
    <div class="modal-overlay" @click=${props.onCloseAddModal}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large cron-config-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 class="emp-detail-title" style="margin: 0;">新建任务</h1>
            <div class="emp-detail-summary cron-config-modal__sub">
              配置周期、消息内容、通道与会话后，系统将按调度自动触发对话。
            </div>
          </div>
          <button
            class="emp-detail-modal__close"
            type="button"
            aria-label=${t("commonCancel")}
            @click=${props.onCloseAddModal}
          >
            ${icons.x}
          </button>
        </div>
        <div class="cron-config-modal__body">
          ${props.error ? html`<div class="callout danger">${props.error}</div>` : nothing}
          ${renderCronForm(props)}
        </div>
        <div class="modal__actions cron-config-modal__actions">
          <button class="btn" @click=${props.onCloseAddModal}>${t("commonCancel")}</button>
          <button
            class="btn primary"
            ?disabled=${props.busy || !props.form.payloadText.trim()}
            @click=${props.onAdd}
          >
            ${props.busy ? t("commonSaving") : t("cronAddJob")}
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderCronEditModal(props: CronProps, jobId: string) {
  return html`
    <div class="modal-overlay" @click=${props.onCloseEditModal ?? (() => {})}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large cron-config-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 class="emp-detail-title" style="margin: 0;">编辑定时任务</h1>
            <div class="emp-detail-summary cron-config-modal__sub">修改后将立即保存到本地任务配置。</div>
          </div>
          <button
            class="emp-detail-modal__close"
            type="button"
            aria-label=${t("commonCancel")}
            @click=${props.onCloseEditModal ?? (() => {})}
          >
            ${icons.x}
          </button>
        </div>
        <div class="cron-config-modal__body">
          ${props.error ? html`<div class="callout danger">${props.error}</div>` : nothing}
          ${renderCronForm(props)}
        </div>
        <div class="modal__actions cron-config-modal__actions">
          <button class="btn" @click=${props.onCloseEditModal ?? (() => {})}>${t("commonCancel")}</button>
          <button
            class="btn primary"
            ?disabled=${props.busy || !props.form.payloadText.trim()}
            @click=${() => props.onUpdate?.(jobId)}
          >
            ${props.busy ? t("commonSaving") : "保存修改"}
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderScheduleTabs(props: CronProps) {
  const kind = props.form.scheduleKind;
  const setKind = (next: CronFormState["scheduleKind"]) => props.onFormChange({ scheduleKind: next });
  return html`
    <div class="cron-schedule-tabs" role="tablist" aria-label="调度方式">
      <button
        type="button"
        role="tab"
        class="cron-schedule-tabs__tab ${kind === "every" ? "cron-schedule-tabs__tab--active" : ""}"
        aria-selected=${kind === "every"}
        @click=${() => setKind("every")}
      >
        每
      </button>
      <button
        type="button"
        role="tab"
        class="cron-schedule-tabs__tab ${kind === "at" ? "cron-schedule-tabs__tab--active" : ""}"
        aria-selected=${kind === "at"}
        @click=${() => setKind("at")}
      >
        在
      </button>
      <button
        type="button"
        role="tab"
        class="cron-schedule-tabs__tab ${kind === "cron" ? "cron-schedule-tabs__tab--active" : ""}"
        aria-selected=${kind === "cron"}
        @click=${() => setKind("cron")}
      >
        Cron
      </button>
    </div>
  `;
}

function renderHourMinuteSelects(
  props: CronProps,
  hour: string,
  minute: string,
  onHour: (v: string) => void,
  onMinute: (v: string) => void,
) {
  const hours = Array.from({ length: 24 }, (_, i) => String(i));
  const minutes = Array.from({ length: 60 }, (_, i) => String(i));
  return html`
    <label class="field">
      <span>时</span>
      <span class="select"><select .value=${hour} @change=${(e: Event) => onHour((e.target as HTMLSelectElement).value)}>
        ${hours.map((h) => html`<option value=${h} ?selected=${h === hour}>${h.padStart(2, "0")}</option>`)}
      </select></span>
    </label>
    <label class="field">
      <span>分</span>
      <span class="select"><select .value=${minute} @change=${(e: Event) => onMinute((e.target as HTMLSelectElement).value)}>
        ${minutes.map((m) => html`<option value=${m} ?selected=${m === minute}>${m.padStart(2, "0")}</option>`)}
      </select></span>
    </label>
  `;
}

function renderScheduleFields(props: CronProps) {
  const form = props.form;
  if (form.scheduleKind === "every") {
    return html`
      <div class="muted" style="margin: 8px 0 12px;">按固定间隔重复运行，例如每 30 分钟或每 2 小时。</div>
      <div class="form-grid">
        <label class="field">
          <span>${t("cronEvery")}</span>
          <span class="input"><input
            .value=${form.everyAmount}
            @input=${(e: Event) =>
              props.onFormChange({ everyAmount: (e.target as HTMLInputElement).value })}
          /></span>
        </label>
        <label class="field">
          <span>${t("cronUnit")}</span>
          <span class="select"><select
            .value=${form.everyUnit}
            @change=${(e: Event) =>
              props.onFormChange({
                everyUnit: (e.target as HTMLSelectElement).value as CronFormState["everyUnit"],
              })}
          >
            <option value="minutes">${t("cronMinutes")}</option>
            <option value="hours">${t("cronHours")}</option>
            <option value="days">${t("cronDays")}</option>
          </select></span>
        </label>
      </div>
    `;
  }
  if (form.scheduleKind === "at") {
    return html`
      <div class="muted" style="margin: 8px 0 12px;">
        在指定时刻运行，例如每天凌晨 1:00，或每周固定星期与时间。
      </div>
      <div class="form-grid">
        <label class="field">
          <span>重复方式</span>
          <span class="select"><select
            .value=${form.atRepeatMode}
            @change=${(e: Event) =>
              props.onFormChange({
                atRepeatMode: (e.target as HTMLSelectElement).value as CronFormState["atRepeatMode"],
              })}
          >
            <option value="daily">每天</option>
            <option value="weekly">每周</option>
          </select></span>
        </label>
        ${
          form.atRepeatMode === "weekly"
            ? html`
                <label class="field">
                  <span>星期</span>
                  <span class="select"><select
                    .value=${form.atWeekday}
                    @change=${(e: Event) =>
                      props.onFormChange({ atWeekday: (e.target as HTMLSelectElement).value })}
                  >
                    ${WEEKDAY_LABELS.map(
                      (label, idx) =>
                        html`<option value=${String(idx)} ?selected=${form.atWeekday === String(idx)}>${label}</option>`,
                    )}
                  </select></span>
                </label>
              `
            : nothing
        }
        ${renderHourMinuteSelects(
          props,
          form.atHour,
          form.atMinute,
          (v) => props.onFormChange({ atHour: v }),
          (v) => props.onFormChange({ atMinute: v }),
        )}
      </div>
      <div class="muted" style="margin-top: 8px; font-size: 12px;">
        对应 Quartz: ${quartzPreviewFromAtForm(form)}
      </div>
    `;
  }
  return html`
    <div class="muted" style="margin: 8px 0 12px;">直接填写标准 5 段 Cron 表达式（分 时 日 月 周）。</div>
    <div class="form-grid">
      <label class="field">
        <span>${t("cronExpression")}</span>
        <span class="input"><input
          .value=${form.cronExpr}
          @input=${(e: Event) =>
            props.onFormChange({ cronExpr: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
      <label class="field">
        <span>Timezone (optional)</span>
        <span class="input"><input
          .value=${form.cronTz}
          @input=${(e: Event) =>
            props.onFormChange({ cronTz: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
    </div>
  `;
}

function renderMultiSelectBox(
  items: Array<{ key: string; label: string; description?: string }>,
  selected: string[],
  emptyText: string,
  onToggle: (key: string, checked: boolean) => void,
) {
  if (!items.length) {
    return html`<div class="cron-resource-box muted">${emptyText}</div>`;
  }
  return html`
    <div class="cron-resource-box">
      ${items.map(
        (item) => html`
          <label class="cron-resource-item">
            <input
              type="checkbox"
              .checked=${selected.includes(item.key)}
              @change=${(e: Event) => onToggle(item.key, (e.target as HTMLInputElement).checked)}
            />
            <span class="cron-resource-item__text">
              <span class="cron-resource-item__name">${item.label}</span>
              ${item.description ? html`<span class="muted cron-resource-item__desc">${item.description}</span>` : nothing}
            </span>
          </label>
        `,
      )}
    </div>
  `;
}

function renderCronForm(props: CronProps) {
  const channelOptions = buildChannelOptions(props);
  const employeeOptions = (props.digitalEmployees ?? [])
    .filter((e) => e.enabled !== false)
    .slice()
    .sort((a, b) => (a.name ?? a.id ?? "").localeCompare(b.name ?? b.id ?? "", undefined, { sensitivity: "base" }));
  const selectedEmployeeId = normalizeEmployeeIdForCron(props.form.digitalEmployeeId ?? "");
  const isEditing = !!props.editModalOpen && !!props.editJobId;
  const selectedEmployee =
    selectedEmployeeId && employeeOptions.length
      ? employeeOptions.find(
          (e) => normalizeEmployeeIdForCron(e.id ?? "").toLowerCase() === selectedEmployeeId.toLowerCase(),
        )
      : undefined;
  const employeeMissing = selectedEmployeeId !== "" && !selectedEmployee && !props.digitalEmployeesLoading;

  const enabledSkills = (props.skillOptions ?? []).filter((s) => !s.disabled && s.eligible);
  const skillItems = enabledSkills.map((s) => ({
    key: s.skillKey,
    label: s.name || s.skillKey,
    description: s.description,
  }));
  const mcpItems = (props.mcpOptions ?? []).map((m) => ({ key: m.key, label: m.label }));

  return html`
    <div class="form-grid">
      <label class="field">
        <span>${t("cronName")}</span>
        <span class="input"><input
          .value=${props.form.name}
          @input=${(e: Event) => props.onFormChange({ name: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
      <label class="field">
        <span>${t("cronDescription")}</span>
        <span class="textarea"><textarea
          rows="2"
          .value=${props.form.description}
          @input=${(e: Event) =>
            props.onFormChange({ description: (e.target as HTMLTextAreaElement).value })}
        ></textarea></span>
      </label>
    </div>

    <div class="field" style="margin-top: 16px;">
      <span>${t("cronSchedule")}</span>
      ${renderScheduleTabs(props)}
      ${renderScheduleFields(props)}
    </div>

    <label class="field" style="margin-top: 16px;">
      <span>Agent 消息（用户侧提示词）<span style="color: var(--danger-color);"> *</span></span>
      <span class="textarea"><textarea
        .value=${props.form.payloadText}
        placeholder="定时触发时发送给智能体的内容，相当于会话中的用户消息"
        @input=${(e: Event) =>
          props.onFormChange({ payloadText: (e.target as HTMLTextAreaElement).value })}
        rows="4"
        required
      ></textarea></span>
    </label>

    <div class="form-grid" style="margin-top: 12px;">
      <label class="field">
        <span>通道（个人已开通实例）</span>
        <span class="select"><select
          .value=${props.form.channel}
          @change=${(e: Event) =>
            props.onFormChange({ channel: (e.target as HTMLSelectElement).value })}
        >
          <option value="" ?selected=${!props.form.channel}>不指定</option>
          ${channelOptions.map(
            (channel) =>
              html`<option value=${channel} ?selected=${props.form.channel === channel}>
                ${resolveChannelLabel(props, channel)}
              </option>`,
          )}
        </select></span>
        <div class="muted" style="font-size: 11px; margin-top: 4px;">
          仅列出你已开通的通道；如需更多通道，请先到「资源市场-通道」申请。
        </div>
      </label>
      ${
        props.form.channel
          ? html`
              <label class="field">
                <span>${t("cronTo")}<span style="color: var(--danger-color);"> *</span></span>
                <span class="input"><input
                  .value=${props.form.deliveryTo}
                  placeholder=${t("cronToPlaceholder")}
                  @input=${(e: Event) =>
                    props.onFormChange({ deliveryTo: (e.target as HTMLInputElement).value })}
                /></span>
                <div class="muted" style="font-size: 11px; margin-top: 4px;">${t("cronToHint")}</div>
              </label>
            `
          : nothing
      }
    </div>

    <label class="field" style="margin-top: 12px;">
      <span>数字员工（可选）</span>
      <span class="select"><select
        @change=${async (e: Event) => {
          const el = e.target as HTMLSelectElement;
          const next = normalizeEmployeeIdForCron(el.value ?? "");
          const prev = normalizeEmployeeIdForCron(props.form.digitalEmployeeId ?? "");
          if (isEditing && prev && !next) {
            const ok = await nativeConfirm(
              `你正在移除该定时任务绑定的数字员工（${prev}）。\n\n移除后：后续触发将不再使用该数字员工会话上下文。\n\n确认移除吗？`,
            );
            if (!ok) {
              el.value = prev;
              return;
            }
          }
          props.onFormChange({ digitalEmployeeId: next });
        }}
      >
        <option value="" ?selected=${!selectedEmployeeId}>不指定</option>
        ${employeeOptions.map((emp) => {
          const idNorm = normalizeEmployeeIdForCron(emp.id ?? "");
          return html`<option value=${idNorm} ?selected=${idNorm === selectedEmployeeId}>
            ${emp.name ? `${emp.name}（${idNorm}）` : idNorm}
          </option>`;
        })}
        ${
          employeeMissing
            ? html`<option value=${selectedEmployeeId} ?selected=${true}>—（已删除：${selectedEmployeeId}）</option>`
            : nothing
        }
      </select></span>
      <div class="muted" style="font-size: 11px; margin-top: 4px;">
        选择已授权且已启用的数字员工后，任务会按该数字员工的配置发起会话。
      </div>
    </label>
    ${
      employeeMissing
        ? html`<div class="callout danger" style="margin-top: 8px;">
            该定时任务已配置的数字员工（${selectedEmployeeId}）不存在（可能已被删除）。请改为有效项或清空后保存。
          </div>`
        : nothing
    }

    <label class="field" style="margin-top: 12px;">
      <span>授权模型（可选）</span>
      <span class="select"><select
        .value=${props.form.modelRef}
        @change=${(e: Event) =>
          props.onFormChange({ modelRef: (e.target as HTMLSelectElement).value })}
      >
        ${(props.modelOptions ?? [{ value: "", label: "不指定" }]).map(
          (opt) => html`<option value=${opt.value} ?selected=${props.form.modelRef === opt.value}>${opt.label}</option>`,
        )}
      </select></span>
      <div class="muted" style="font-size: 11px; margin-top: 4px;">
        仅列出你已授权的模型；不指定时由系统自动选择。
      </div>
    </label>

    <label class="field" style="margin-top: 12px;">
      <span>技能（多选，可选）</span>
      ${renderMultiSelectBox(
        skillItems,
        props.form.skillKeys,
        "暂无已授权且已启用的技能",
        (key, checked) =>
          props.onFormChange({
            skillKeys: toggleListItem(props.form.skillKeys, key, checked),
          }),
      )}
      <div class="muted" style="font-size: 11px; margin-top: 4px;">
        不选择时，默认使用所有已授权且已启用的技能和工具。
      </div>
    </label>

    <label class="field" style="margin-top: 12px;">
      <span>工具（多选，可选）</span>
      ${renderMultiSelectBox(
        mcpItems,
        props.form.mcpServers,
        "暂无已授权且已启用的工具",
        (key, checked) =>
          props.onFormChange({
            mcpServers: toggleListItem(props.form.mcpServers, key, checked),
          }),
      )}
    </label>

    <label class="field" style="margin-top: 12px;">
      <span>附加参数 (JSON)</span>
      <span class="textarea"><textarea
        rows="3"
        .value=${props.form.extraParamsJson}
        @input=${(e: Event) =>
          props.onFormChange({ extraParamsJson: (e.target as HTMLTextAreaElement).value })}
      ></textarea></span>
      <div class="muted" style="font-size: 11px; margin-top: 4px;">
        可填写高级参数；上方选择的技能、工具、模型会在保存时自动合并。
      </div>
    </label>
  `;
}

export function renderCronHistory(props: CronProps) {
  const orderedRuns = props.runs.toSorted((a, b) => b.ts - a.ts);

  return html`
    <section class="stack cron-config-stack cron-config-stack-history">
      <div class="card">
        ${
          props.jobs.length === 0
            ? nothing
            : html`
                <label class="field">
                  <span>${t("agentsTabCron")}</span>
                  <span class="select"><select @change=${(e: Event) => {
                    const value = (e.target as HTMLSelectElement).value;
                    if (value) props.onLoadRuns(value);
                  }}>
                    <option value="" ?selected=${props.runsJobId == null}>${t("cronSelectJob")}</option>
                    ${props.jobs.map(
                      (job) =>
                        html`<option value=${job.id} ?selected=${props.runsJobId === job.id}>
                          ${job.name}
                        </option>`,
                    )}
                  </select></span>
                </label>
              `
        }
        ${
          props.runsJobId == null
            ? props.jobs.length === 0
              ? html`<div class="muted" style="margin-top: 12px">${t("cronNoJobsYet")}</div>`
              : html`<div class="muted" style="margin-top: 12px">${t("cronSelectJobToInspect")}</div>`
            : orderedRuns.length === 0
              ? html`<div class="muted" style="margin-top: 12px">${t("cronNoRunsYet")}</div>`
              : html`
                  <div class="list" style="margin-top: 12px;">
                    ${orderedRuns.map((entry) => renderRun(entry, props.basePath))}
                  </div>
                `
        }
      </div>
    </section>
  `;
}

function renderJob(job: CronJob, props: CronProps, opts: { mode: "config" | "history" }) {
  const isSelected = props.runsJobId === job.id;
  const itemClass = `list-item list-item-clickable cron-job${isSelected ? " list-item-selected" : ""}`;
  return html`
    <div
      class=${itemClass}
      @click=${() => {
        if (opts.mode === "config") {
          props.onShowHistory?.(job.id);
          return;
        }
        props.onLoadRuns(job.id);
      }}
    >
      <div class="list-main">
        <div class="list-title">${job.name}</div>
        <div class="list-sub">${formatCronSchedule(job)}</div>
        ${renderJobPayload(job)}
        ${renderJobEmployee(job, props)}
        ${renderJobRunConfig(job)}
      </div>
      <div class="list-meta">
        ${renderJobState(job)}
      </div>
      <div class="cron-job-footer">
        <div class="chip-row cron-job-chips">
          <span class=${`chip ${job.enabled ? "chip-ok" : "chip-danger"}`}>
            ${job.enabled ? "enabled" : "disabled"}
          </span>
          ${job.delivery?.channel ? html`<span class="chip">${job.delivery.channel}</span>` : nothing}
        </div>
        <div class="row cron-job-actions">
          ${
            opts.mode === "config"
              ? html`
                  <button
                    class="btn"
                    ?disabled=${props.busy}
                    @click=${(event: Event) => {
                      event.stopPropagation();
                      props.onOpenEditModal?.(job);
                    }}
                  >
                    Edit
                  </button>
                `
              : nothing
          }
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${(event: Event) => {
              event.stopPropagation();
              props.onToggle(job, !job.enabled);
            }}
          >
            ${job.enabled ? "Disable" : "Enable"}
          </button>
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${(event: Event) => {
              event.stopPropagation();
              props.onRun(job);
            }}
          >
            Run
          </button>
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${(event: Event) => {
              event.stopPropagation();
              if (opts.mode === "config") {
                props.onShowHistory?.(job.id);
                return;
              }
              props.onLoadRuns(job.id);
            }}
          >
            History
          </button>
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${async (event: Event) => {
              event.stopPropagation();
              if (props.confirmRemove && !(await nativeConfirm(t("cronDeleteConfirm")))) {
                return;
              }
              props.onRemove(job);
            }}
          >
            Remove
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderJobEmployee(job: CronJob, props: CronProps) {
  const raw = normalizeEmployeeIdForCron(job.digitalEmployeeId ?? "");
  if (!raw) return nothing;
  const list = props.digitalEmployees ?? [];
  const hit = list.find((e) => normalizeEmployeeIdForCron(e.id ?? "").toLowerCase() === raw.toLowerCase());
  if (!hit) {
    return html`<div class="cron-job-detail">
      <span class="cron-job-detail-label">数字员工</span>
      <span class="muted cron-job-detail-value">—（已删除：${raw}）</span>
    </div>`;
  }
  const label = hit.name ? `${hit.name}（${raw}）` : raw;
  return html`<div class="cron-job-detail">
    <span class="cron-job-detail-label">数字员工</span>
    <span class="muted cron-job-detail-value">${label}</span>
  </div>`;
}

function renderJobRunConfig(job: CronJob) {
  const rc = job.runConfig;
  if (!rc) return nothing;
  const parts: string[] = [];
  if (rc.modelRef) parts.push(`模型: ${rc.modelRef}`);
  if (rc.skillKeys?.length) parts.push(`技能: ${rc.skillKeys.length}`);
  if (rc.mcpServers?.length) parts.push(`工具: ${rc.mcpServers.length}`);
  if (!parts.length) return nothing;
  return html`<div class="cron-job-detail">
    <span class="cron-job-detail-label">运行配置</span>
    <span class="muted cron-job-detail-value">${parts.join(" · ")}</span>
  </div>`;
}

function renderJobPayload(job: CronJob) {
  if (job.payload.kind === "systemEvent") {
    return html`<div class="cron-job-detail">
      <span class="cron-job-detail-label">System</span>
      <span class="muted cron-job-detail-value">${job.payload.text}</span>
    </div>`;
  }
  return html`
    <div class="cron-job-detail">
      <span class="cron-job-detail-label">Prompt</span>
      <span class="muted cron-job-detail-value">${job.payload.message}</span>
    </div>
    ${
      job.delivery?.channel
        ? html`<div class="cron-job-detail">
            <span class="cron-job-detail-label">通道</span>
            <span class="muted cron-job-detail-value">${job.delivery.channel}</span>
          </div>`
        : nothing
    }
  `;
}

function formatStateRelative(ms?: number) {
  if (typeof ms !== "number" || !Number.isFinite(ms)) {
    return "n/a";
  }
  return formatAgo(ms);
}

function renderJobState(job: CronJob) {
  const status = job.state?.lastStatus ?? "n/a";
  const statusClass =
    status === "ok"
      ? "cron-job-status-ok"
      : status === "error"
        ? "cron-job-status-error"
        : status === "skipped"
          ? "cron-job-status-skipped"
          : "cron-job-status-na";
  const nextRunAtMs = job.state?.nextRunAtMs;
  const lastRunAtMs = job.state?.lastRunAtMs;

  return html`
    <div class="cron-job-state">
      <div class="cron-job-state-row">
        <span class="cron-job-state-key">Status</span>
        <span class=${`cron-job-status-pill ${statusClass}`}>${status}</span>
      </div>
      <div class="cron-job-state-row">
        <span class="cron-job-state-key">Next</span>
        <span class="cron-job-state-value" title=${formatMs(nextRunAtMs)}>
          ${formatStateRelative(nextRunAtMs)}
        </span>
      </div>
      <div class="cron-job-state-row">
        <span class="cron-job-state-key">Last</span>
        <span class="cron-job-state-value" title=${formatMs(lastRunAtMs)}>
          ${formatStateRelative(lastRunAtMs)}
        </span>
      </div>
    </div>
  `;
}

function cronRunChatSessionKey(entry: CronRunLogEntry): string | null {
  const fromRun = cronRunHistorySessionKey(entry.sessionKey);
  if (fromRun) {
    return fromRun;
  }
  const jid = entry.jobId?.trim();
  return jid ? `agent:main:cron:${jid}` : null;
}

function renderRun(entry: CronRunLogEntry, basePath: string) {
  const chatSessionKey = cronRunChatSessionKey(entry);
  const chatUrl = chatSessionKey
    ? `${pathForTab("message", basePath)}?session=${encodeURIComponent(chatSessionKey)}`
    : null;
  return html`
    <div class="list-item">
      <div class="list-main">
        <div class="list-title">${entry.status}</div>
        <div class="list-sub">${entry.summary ?? ""}</div>
      </div>
      <div class="list-meta">
        <div>${formatMs(entry.ts)}</div>
        <div class="muted">${entry.durationMs ?? 0}ms</div>
        ${
          chatUrl
            ? html`<div><a class="session-link" href=${chatUrl}>Open run chat</a></div>`
            : nothing
        }
        ${entry.error ? html`<div class="muted">${entry.error}</div>` : nothing}
      </div>
    </div>
  `;
}

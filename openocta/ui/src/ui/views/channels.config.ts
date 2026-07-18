import { html, nothing } from "lit";
import { icons } from "../icons.js";
import type { ConfigUiHints } from "../types.ts";
import type { ChannelsProps } from "./channels.types.ts";
import { t } from "../strings.js";
import {
  getChannelFormDef,
  getValueAtPath,
  type ChannelFieldDef,
} from "./channels-config-fields.ts";

type ChannelConfigFormProps = {
  channelId: string;
  configValue: Record<string, unknown> | null;
  schema: unknown;
  uiHints: ConfigUiHints;
  digitalEmployees?: Array<{
    id: string;
    name?: string;
    enabled?: boolean;
    builtin?: boolean;
    from?: string;
    type?: string;
  }>;
  digitalEmployeesLoading?: boolean;
  disabled: boolean;
  onPatch: (path: Array<string | number>, value: unknown) => void;
};

function normalizeEmployeeId(raw: string): string {
  const v = (raw ?? "").trim();
  if (!v) return "";
  // 兼容 /api/v1/employees 返回的本地前缀：local:<id>
  if (v.toLowerCase().startsWith("local:")) {
    return v.slice("local:".length).trim();
  }
  return v;
}

function employeeSourceLabel(emp: { builtin?: boolean; from?: string }): string {
  if (emp?.builtin) return "内置";
  const from = (emp?.from ?? "").trim().toLowerCase();
  if (from === "remote") return "远程";
  if (from === "local") return "本地";
  return "本地";
}

function resolveChannelValue(
  config: Record<string, unknown>,
  channelId: string,
): Record<string, unknown> {
  const channels = (config.channels ?? {}) as Record<string, unknown>;
  const fromChannels = channels[channelId];
  const fallback = config[channelId];
  const resolved =
    (fromChannels && typeof fromChannels === "object"
      ? (fromChannels as Record<string, unknown>)
      : null) ??
    (fallback && typeof fallback === "object" ? (fallback as Record<string, unknown>) : null);
  return resolved ?? {};
}

function formatFieldValue(raw: unknown, field: ChannelFieldDef): string {
  if (raw == null) {
    return "";
  }
  if (field.type === "boolean") {
    return raw ? "true" : "false";
  }
  if (field.type === "string[]") {
    return Array.isArray(raw) ? raw.join(", ") : typeof raw === "string" ? raw : "";
  }
  return String(raw);
}

function parseFieldValue(value: string, field: ChannelFieldDef): unknown {
  if (field.type === "boolean") {
    return value === "true" || value === "1" || value.toLowerCase() === "yes";
  }
  if (field.type === "number") {
    const n = parseInt(value, 10);
    return isNaN(n) ? undefined : n;
  }
  if (field.type === "string[]") {
    return value.trim() ? value.split(/,\s*/).map((s) => s.trim()).filter(Boolean) : [];
  }
  return value;
}

export function renderChannelConfigForm(props: ChannelConfigFormProps) {
  const formDef = getChannelFormDef(props.channelId);
  const configValue = props.configValue ?? {};
  const value = resolveChannelValue(configValue, props.channelId) as Record<string, unknown>;

  if (!formDef) {
    return html`
      <div class="callout danger">${t("channelsConfigSchemaUnavailable")}</div>
    `;
  }

  const employeeOptions = (props.digitalEmployees ?? []).slice().sort((a, b) => {
    const an = (a.name ?? a.id ?? "").toLowerCase();
    const bn = (b.name ?? b.id ?? "").toLowerCase();
    return an.localeCompare(bn);
  });
  const selectedEmployeeIdRaw = String((value as Record<string, unknown>)["digitalEmployeeId"] ?? "").trim();
  const selectedEmployeeId = normalizeEmployeeId(selectedEmployeeIdRaw);
  const selectedEmployee =
    selectedEmployeeId && employeeOptions.length
      ? employeeOptions.find(
          (e) => normalizeEmployeeId(String(e.id ?? "")).toLowerCase() === selectedEmployeeId.toLowerCase(),
        )
      : undefined;
  // 兼容：历史/手工配置可能把“名称”写进 digitalEmployeeId；这里做一次按 name 的兜底匹配。
  const selectedEmployeeByName =
    !selectedEmployee && selectedEmployeeId && employeeOptions.length
      ? employeeOptions.find(
          (e) => (e.name ?? "").trim().toLowerCase() === selectedEmployeeId.toLowerCase(),
        )
      : undefined;
  const resolvedEmployee = selectedEmployee ?? selectedEmployeeByName;
  // 避免在数字员工列表仍在加载时误判为“已删除”
  const employeeMissing =
    selectedEmployeeId !== "" && !resolvedEmployee && props.digitalEmployeesLoading !== true;
  const currentEmployeeValue = normalizeEmployeeId(String(resolvedEmployee?.id ?? selectedEmployeeIdRaw));

  return html`
    <div class="config-form">
      <div class="field">
        <span>数字员工</span>
        <span class="select"><select
          .value=${currentEmployeeValue}
          ?disabled=${props.disabled}
          @change=${(e: Event) =>
            props.onPatch(
              ["channels", props.channelId, "digitalEmployeeId"],
              (e.target as HTMLSelectElement).value,
            )}
        >
          <option value="" ?selected=${currentEmployeeValue === ""}>（不选择）</option>
          ${employeeOptions.map((emp) => {
            const id = normalizeEmployeeId(String(emp.id ?? ""));
            const src = employeeSourceLabel(emp);
            const label = emp.name ? `${emp.name}（${id} · ${src}）` : `${id}（${src}）`;
            return html`<option value=${id} ?selected=${id === currentEmployeeValue}>${label}</option>`;
          })}
          ${
            employeeMissing
              ? html`<option value=${selectedEmployeeId} ?selected=${selectedEmployeeId === currentEmployeeValue}>—（已删除：${selectedEmployeeId}）</option>`
              : nothing
          }
        </select></span>
        <div class="muted" style="font-size: 12px; margin-top: 6px; line-height: 1.35;">
          选择后，该渠道进入的对话会使用该数字员工的人设与技能；若数字员工被删除，会回退为普通会话并提示你修改配置。
        </div>
        ${
          resolvedEmployee
            ? html`<div class="row" style="margin-top: 8px; gap: 8px; flex-wrap: wrap;">
                <span class="chip">${employeeSourceLabel(resolvedEmployee)}</span>
                ${resolvedEmployee.type
                  ? html`<span class="chip">${String(resolvedEmployee.type).trim() || "其它"}</span>`
                  : html`<span class="chip">其它</span>`}
              </div>`
            : nothing
        }
      </div>
      ${
        employeeMissing
          ? html`<div class="callout danger" style="margin-top: 12px;">
              当前渠道已配置的数字员工（${selectedEmployeeId}）不存在（可能已被删除）。请将“数字员工”改为有效项或清空后保存。
            </div>`
          : nothing
      }
      ${formDef.fields.map((field) => {
        const raw = getValueAtPath(value, field.path);
        const displayValue = formatFieldValue(raw, field);
        const basePath = ["channels", props.channelId, ...field.path];
        return html`
          <div class="field">
            <span>
              ${field.label}
              ${field.required ? html`<span style="color: var(--danger-color);">*</span>` : ""}
            </span>
            ${field.type === "boolean"
              ? html`
                  <div class="row" style="align-items: center; gap: 8px;">
                    <span class="checkbox"><input
                      type="checkbox"
                      ?checked=${raw === true}
                      ?disabled=${props.disabled}
                      @change=${(e: Event) =>
                        props.onPatch(basePath, (e.target as HTMLInputElement).checked)}
                    /></span>
                  </div>
                `
              : html`
                  <span class="input"><input
                    type="${field.type === "number" ? "number" : "text"}"
                    .value=${displayValue}
                    placeholder=${field.placeholder ?? ""}
                    ?disabled=${props.disabled}
                    @input=${(e: Event) => {
                      const v = (e.target as HTMLInputElement).value;
                      props.onPatch(basePath, parseFieldValue(v, field));
                    }}
                  /></span>
                `}
            ${field.help
              ? html`<div class="muted" style="font-size: 12px; margin-top: 4px; line-height: 1.35;">
                  ${field.help}
                </div>`
              : nothing}
          </div>
        `;
      })}
    </div>
  `;
}

export function renderChannelConfigSection(params: { channelId: string; props: ChannelsProps }) {
  const { channelId, props } = params;
  const disabled = props.configSaving;
  return html`
    <div>
      ${renderChannelConfigForm({
        channelId,
        configValue: props.configForm,
        schema: props.configSchema,
        uiHints: props.configUiHints,
        digitalEmployees: props.digitalEmployees,
        digitalEmployeesLoading: props.digitalEmployeesLoading,
        disabled,
        onPatch: props.onConfigPatch,
      })}
      <div class="row" style="margin-top: 12px;">
        <button
          class="btn primary"
          ?disabled=${disabled || !props.configFormDirty}
          @click=${() => props.onConfigSave()}
        >
          ${props.configSaving ? t("commonSaving") : t("commonSave")}
        </button>
        <button
          class="btn"
          ?disabled=${disabled}
          @click=${() => props.onConfigReload()}
        >
          ${t("commonReload")}
        </button>
      </div>
    </div>
  `;
}

const CHANNEL_LABELS: Record<string, string> = {
  wework: "微信",
  weixin: "个人微信",
  dingtalk: "钉钉",
  feishu: "飞书",
  qq: "QQ",
};

/** Renders the slide-out panel for channel config (CoPaw-style). Shown when selectedChannelId is set. */
export function renderChannelConfigPanel(props: ChannelsProps) {
  const channelId = props.selectedChannelId;
  if (!channelId) {
    return nothing;
  }
  const label = CHANNEL_LABELS[channelId.toLowerCase()] ?? channelId.charAt(0).toUpperCase() + channelId.slice(1);
  return html`
    <div
      class="channel-panel-overlay"
      @click=${(e: Event) => {
        if ((e.target as HTMLElement).classList.contains("channel-panel-overlay")) {
          props.onChannelSelect(null);
        }
      }}
    >
      <div class="channel-panel card" @click=${(e: Event) => e.stopPropagation()}>
        <div class="channel-panel-header row" style="justify-content: space-between; align-items: center;">
          <div class="card-title">${label} ${t("configSettingsTitle")}</div>
          <button class="btn btn--icon" type="button" aria-label="关闭" @click=${() => props.onChannelSelect(null)}>
            ${icons.x}
          </button>
        </div>
        <div class="channel-panel-content">
          ${renderChannelConfigSection({ channelId, props })}
        </div>
      </div>
    </div>
  `;
}

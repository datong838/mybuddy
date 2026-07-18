import type { LogLevel } from "./types.ts";
import type { CronFormState } from "./ui-types.ts";
import type { ApiKeyFormState } from "./controllers/api-keys.ts";

export const DEFAULT_API_KEY_FORM: ApiKeyFormState = {
  editId: "",
  name: "",
  allowedPaths: ["/openocta/open/v1/ping", "/openocta/open/v1/completion"],
  bindingMode: "resources",
  allowedModels: [],
  skillKeys: [],
  mcpServers: [],
  digitalEmployeeId: "",
  monthlyTokenLimit: "",
};

export const DEFAULT_LOG_LEVEL_FILTERS: Record<LogLevel, boolean> = {
  trace: true,
  debug: true,
  info: true,
  warn: true,
  error: true,
  fatal: true,
};

export const DEFAULT_CRON_FORM: CronFormState = {
  name: "",
  description: "",
  digitalEmployeeId: "",
  enabled: true,
  scheduleKind: "at",
  scheduleAt: "",
  everyAmount: "30",
  everyUnit: "minutes",
  atRepeatMode: "daily",
  atWeekday: "1",
  atHour: "9",
  atMinute: "0",
  cronExpr: "0 9 * * *",
  cronTz: "",
  payloadText: "",
  channel: "",
  deliveryTo: "",
  modelRef: "",
  skillKeys: [],
  mcpServers: [],
  extraParamsJson: "{}",
};

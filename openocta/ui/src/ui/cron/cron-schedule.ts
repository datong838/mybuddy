import type { CronSchedule } from "../types.ts";
import type { CronFormState } from "../ui-types.ts";

export function buildAtCronExpr(form: CronFormState): string {
  const minute = Math.min(59, Math.max(0, Number.parseInt(form.atMinute, 10) || 0));
  const hour = Math.min(23, Math.max(0, Number.parseInt(form.atHour, 10) || 0));
  if (form.atRepeatMode === "weekly") {
    const dow = Math.min(6, Math.max(0, Number.parseInt(form.atWeekday, 10) || 0));
    return `${minute} ${hour} * * ${dow}`;
  }
  return `${minute} ${hour} * * *`;
}

export function quartzPreviewFromAtForm(form: CronFormState): string {
  const minute = Math.min(59, Math.max(0, Number.parseInt(form.atMinute, 10) || 0));
  const hour = Math.min(23, Math.max(0, Number.parseInt(form.atHour, 10) || 0));
  if (form.atRepeatMode === "weekly") {
    const dow = Math.min(6, Math.max(0, Number.parseInt(form.atWeekday, 10) || 0));
    return `0 ${minute} ${hour} ? * ${dow + 1}`;
  }
  return `0 ${minute} ${hour} * * ?`;
}

export function applyScheduleToForm(schedule: CronSchedule | undefined, form: CronFormState): CronFormState {
  const next = { ...form };
  const kind = schedule?.kind ?? form.scheduleKind;
  if (kind === "every" && schedule?.kind === "every") {
    next.scheduleKind = "every";
    const everyMs = Number(schedule.everyMs ?? 0);
    const minute = 60_000;
    const hour = 3_600_000;
    const day = 86_400_000;
    if (everyMs > 0 && everyMs % day === 0) {
      next.everyUnit = "days";
      next.everyAmount = String(Math.max(1, Math.round(everyMs / day)));
    } else if (everyMs > 0 && everyMs % hour === 0) {
      next.everyUnit = "hours";
      next.everyAmount = String(Math.max(1, Math.round(everyMs / hour)));
    } else {
      next.everyUnit = "minutes";
      next.everyAmount = String(Math.max(1, Math.round((everyMs || minute) / minute)));
    }
    return next;
  }
  if (kind === "at" && schedule?.kind === "at") {
    next.scheduleKind = "at";
    const atMs = Date.parse(schedule.at ?? "");
    if (Number.isFinite(atMs)) {
      next.scheduleAt = toLocalDateTimeInputValue(atMs);
    }
    return next;
  }
  if (schedule?.kind === "cron") {
    const expr = String(schedule.expr ?? "").trim();
    const parsed = parseDailyOrWeeklyCron(expr);
    if (parsed) {
      next.scheduleKind = "at";
      next.atRepeatMode = parsed.repeatMode;
      next.atWeekday = String(parsed.weekday);
      next.atHour = String(parsed.hour);
      next.atMinute = String(parsed.minute);
    } else {
      next.scheduleKind = "cron";
      next.cronExpr = expr;
      next.cronTz = String(schedule.tz ?? "").trim();
    }
  }
  return next;
}

function parseDailyOrWeeklyCron(expr: string):
  | { repeatMode: "daily" | "weekly"; hour: number; minute: number; weekday: number }
  | null {
  const parts = expr.trim().split(/\s+/);
  if (parts.length !== 5) {
    return null;
  }
  const minute = Number.parseInt(parts[0], 10);
  const hour = Number.parseInt(parts[1], 10);
  const dom = parts[2];
  const month = parts[3];
  const dow = parts[4];
  if (!Number.isFinite(minute) || !Number.isFinite(hour)) {
    return null;
  }
  if (dom === "*" && month === "*" && dow === "*") {
    return { repeatMode: "daily", hour, minute, weekday: 0 };
  }
  if (dom === "*" && month === "*" && dow !== "*") {
    const weekday = Number.parseInt(dow, 10);
    if (!Number.isFinite(weekday)) {
      return null;
    }
    return { repeatMode: "weekly", hour, minute, weekday };
  }
  return null;
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

function toLocalDateTimeInputValue(ms: number): string {
  const d = new Date(ms);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

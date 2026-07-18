export type ThemeMode = "system" | "light" | "dark";
export type ResolvedTheme = "light" | "dark";

export function getSystemTheme(): ResolvedTheme {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return "dark";
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function resolveTheme(mode: ThemeMode): ResolvedTheme {
  if (mode === "system") {
    return getSystemTheme();
  }
  return mode;
}

const THEME_CYCLE: ThemeMode[] = ["light", "dark", "system"];

export function cycleTheme(mode: ThemeMode): ThemeMode {
  const index = THEME_CYCLE.indexOf(mode);
  return THEME_CYCLE[(index + 1) % THEME_CYCLE.length] ?? "light";
}

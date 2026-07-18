/** 网关自动生成的自定义会话标签 */
const AUTO_CUSTOM_SESSION_LABEL = /^自定义(会话|对话)\d+$/;

const SIDEBAR_TITLE_MAX_LEN = 32;

/** derivedTitle 在无消息时的 sessionId+日期 占位，不作为副标题展示 */
const SESSION_ID_DERIVED_FALLBACK = /^[0-9a-f]{8}(?:\s*\(\d{4}-\d{2}-\d{2}\))?$/i;

export function isSessionIdDerivedFallback(text: string | undefined | null): boolean {
  if (!text) return false;
  return SESSION_ID_DERIVED_FALLBACK.test(text.trim());
}

export function isAutoCustomSessionLabel(label: string | undefined | null): boolean {
  if (!label) return false;
  return AUTO_CUSTOM_SESSION_LABEL.test(label.trim());
}

export function truncateSessionSidebarTitle(text: string, maxLen = SIDEBAR_TITLE_MAX_LEN): string {
  const normalized = text.replace(/\s+/g, " ").trim();
  if (!normalized) return "";
  if (normalized.length <= maxLen) return normalized;
  return `${normalized.slice(0, maxLen - 1)}…`;
}

export type SessionSidebarTitleInput = {
  key: string;
  derivedTitle?: string | null;
  lastMessagePreview?: string | null;
  label?: string | null;
  displayName?: string | null;
  sessionId?: string | null;
  origin?: Record<string, string> | null;
  employeeName?: string | null;
};

/** 侧栏会话标题：自定义会话保持 label（如「自定义会话 N」），空会话显示「新对话」。 */
export function resolveSessionSidebarTitle(input: SessionSidebarTitleInput): string {
  const isCustom = input.key.toLowerCase().startsWith("custom:");
  const derived = input.derivedTitle?.trim() ?? "";
  const preview = input.lastMessagePreview?.trim() ?? "";
  const label = input.label?.trim() ?? "";
  const displayName = input.displayName?.trim() ?? "";

  if (input.employeeName?.trim()) {
    return input.employeeName.trim();
  }

  if (isCustom) {
    if (label) return truncateSessionSidebarTitle(label);
    return "新对话";
  }

  const origin = input.origin;
  const originLabel =
    (origin?.label || origin?.from || origin?.to)?.trim() ?? "";
  if (originLabel) return truncateSessionSidebarTitle(originLabel);

  if (derived) return truncateSessionSidebarTitle(derived);
  if (displayName && !isAutoCustomSessionLabel(displayName)) {
    return truncateSessionSidebarTitle(displayName);
  }
  if (label && !isAutoCustomSessionLabel(label)) return truncateSessionSidebarTitle(label);
  if (preview) return truncateSessionSidebarTitle(preview);

  const sessionId = input.sessionId?.trim();
  if (sessionId) return truncateSessionSidebarTitle(sessionId.slice(0, 8));
  return "会话";
}

export function resolveSessionSidebarSubtitle(
  title: string,
  lastMessagePreview?: string | null,
  derivedTitle?: string | null,
): string {
  const derived = derivedTitle?.trim() ?? "";
  const derivedIsMessage = derived && !isSessionIdDerivedFallback(derived);
  // 优先使用第一条用户消息（derivedTitle）作为副标题
  const conversation = derivedIsMessage ? derived : (lastMessagePreview?.trim() ?? "");
  if (!conversation) return "";
  if (conversation === title) return "";
  return truncateSessionSidebarTitle(conversation, 48);
}

export function compareSessionSidebarRows(
  a: { pinnedAt?: number | null; updatedAt?: number | null },
  b: { pinnedAt?: number | null; updatedAt?: number | null },
): number {
  const pinA = a.pinnedAt ?? 0;
  const pinB = b.pinnedAt ?? 0;
  if (pinA !== pinB) return pinB - pinA;
  return (b.updatedAt ?? 0) - (a.updatedAt ?? 0);
}

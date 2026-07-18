import type { ChatItem, MessageGroup } from "../types/chat-types.ts";
import { CHAT_HISTORY_LIMIT } from "../controllers/chat.ts";
import { normalizeMessage, normalizeRoleForGrouping, isToolResultMessage } from "./message-normalizer.ts";

export type ChatItemsInput = {
  sessionKey: string;
  messages: unknown[];
  toolMessages: unknown[];
  conversationOnly?: boolean;
  canAbort?: boolean;
  stream?: string | null;
  reasoningStream?: string | null;
  streamStartedAt?: number | null;
  runPhase?: "idle" | "thinking" | "tool" | "streaming";
  a2uiMessages?: unknown[];
};

function groupMessages(items: ChatItem[]): Array<ChatItem | MessageGroup> {
  const result: Array<ChatItem | MessageGroup> = [];
  let currentGroup: MessageGroup | null = null;

  for (const item of items) {
    if (item.kind !== "message") {
      if (currentGroup) {
        result.push(currentGroup);
        currentGroup = null;
      }
      result.push(item);
      continue;
    }

    if (isToolResultMessage(item.message)) {
      if (currentGroup && currentGroup.role === "assistant") {
        currentGroup.messages.push({ message: item.message, key: item.key });
        continue;
      }
    }

    const normalized = normalizeMessage(item.message);
    const normalizedRole = normalizeRoleForGrouping(normalized.role);
    const role = normalizedRole === "tool" ? "assistant" : normalizedRole;
    const timestamp = normalized.timestamp || Date.now();

    if (!currentGroup || currentGroup.role !== role) {
      if (currentGroup) {
        result.push(currentGroup);
      }
      currentGroup = {
        kind: "group",
        key: `group:${role}:${item.key}`,
        role,
        messages: [{ message: item.message, key: item.key }],
        timestamp,
        isStreaming: false,
      };
    } else {
      currentGroup.messages.push({ message: item.message, key: item.key });
    }
  }

  if (currentGroup) {
    result.push(currentGroup);
  }

  return result;
}

export function messageKey(message: unknown, index: number): string {
  const m = message as Record<string, unknown>;
  const toolCallId = typeof m.toolCallId === "string" ? m.toolCallId : "";
  if (toolCallId) {
    return `tool:${toolCallId}`;
  }
  const id = typeof m.id === "string" ? m.id : "";
  if (id) {
    return `msg:${id}`;
  }
  const messageId = typeof m.messageId === "string" ? m.messageId : "";
  if (messageId) {
    return `msg:${messageId}`;
  }
  const timestamp = typeof m.timestamp === "number" ? m.timestamp : null;
  const role = typeof m.role === "string" ? m.role : "unknown";
  if (timestamp != null) {
    return `msg:${role}:${timestamp}:${index}`;
  }
  return `msg:${role}:${index}`;
}

export function buildChatItems(props: ChatItemsInput): Array<ChatItem | MessageGroup> {
  const items: ChatItem[] = [];
  const history = Array.isArray(props.messages) ? props.messages : [];
  const tools = Array.isArray(props.toolMessages) ? props.toolMessages : [];
  const conversationOnly = props.conversationOnly ?? false;
  const historyStart = Math.max(0, history.length - CHAT_HISTORY_LIMIT);
  if (historyStart > 0) {
    items.push({
      kind: "message",
      key: "chat:history:notice",
      message: {
        role: "system",
        content: `Showing last ${CHAT_HISTORY_LIMIT} messages (${historyStart} hidden).`,
        timestamp: 0,
      },
    });
  }
  for (let i = historyStart; i < history.length; i++) {
    const msg = history[i];
    const normalized = normalizeMessage(msg);

    if (conversationOnly && normalized.role === "toolResult") {
      continue;
    }

    items.push({
      kind: "message",
      key: messageKey(msg, i),
      message: msg,
    });
  }
  const runActive = Boolean(props.canAbort);
  const liveA2UI = props.a2uiMessages ?? [];
  const streamText = props.stream ?? "";
  const reasoningText = props.reasoningStream?.trim() ?? "";

  if (!conversationOnly && !runActive) {
    for (let i = 0; i < tools.length; i++) {
      items.push({
        kind: "message",
        key: messageKey(tools[i], i + history.length),
        message: tools[i],
      });
    }
  }

  if (runActive) {
    const key = `stream:${props.sessionKey}:${props.streamStartedAt ?? "live"}`;
    const phase =
      props.runPhase === "tool"
        ? "tool"
        : props.runPhase === "streaming"
          ? "streaming"
          : "thinking";
    if (liveA2UI.length > 0) {
      items.push({
        kind: "a2ui",
        key: `${key}:a2ui`,
        messages: liveA2UI,
        runPhase: phase,
        reasoningText: reasoningText || undefined,
      });
    } else if (streamText.length > 0) {
      items.push({
        kind: "stream",
        key,
        text: streamText,
        reasoningText: reasoningText || undefined,
        startedAt: props.streamStartedAt ?? Date.now(),
      });
    }
    // Keep a live status row while the run is in-flight (tool/thinking between turns).
    if (liveA2UI.length === 0 && streamText.length === 0) {
      items.push({
        kind: "reading-indicator",
        key,
        startedAt: props.streamStartedAt ?? Date.now(),
        phase,
        reasoningText: reasoningText || undefined,
      });
    }
  }

  return groupMessages(items);
}

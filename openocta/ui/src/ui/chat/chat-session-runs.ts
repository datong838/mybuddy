import type { ChatState } from "../controllers/chat.ts";
import type { ChatEventPayload } from "../controllers/chat.ts";
import { dedupeA2UIMessages } from "./a2ui-bridge.ts";
import { extractThinking } from "./message-extract.ts";
import { canonicalGatewaySessionKey, gatewaySessionKeysEqual } from "../sessions/session-key-utils.js";

export type SessionChatRunState = {
  runId: string | null;
  runPhase: "idle" | "thinking" | "tool" | "streaming";
  stream: string | null;
  reasoningStream: string | null;
  streamStartedAt: number | null;
  a2uiMessages: unknown[];
  terminalRunIds: string[];
  errorRunId: string | null;
  sending: boolean;
};

const TERMINAL_RUN_LIMIT = 24;

function defaultRunState(): SessionChatRunState {
  return {
    runId: null,
    runPhase: "idle",
    stream: null,
    reasoningStream: null,
    streamStartedAt: null,
    a2uiMessages: [],
    terminalRunIds: [],
    errorRunId: null,
    sending: false,
  };
}

const runStore = new Map<string, SessionChatRunState>();

function storeKey(sessionKey: string): string {
  return canonicalGatewaySessionKey(sessionKey);
}

export function getSessionChatRunState(sessionKey: string): SessionChatRunState {
  const key = storeKey(sessionKey);
  const existing = runStore.get(key);
  if (!existing) {
    return defaultRunState();
  }
  return {
    ...existing,
    a2uiMessages: [...existing.a2uiMessages],
    terminalRunIds: [...existing.terminalRunIds],
  };
}

function putSessionChatRunState(sessionKey: string, next: SessionChatRunState) {
  runStore.set(storeKey(sessionKey), {
    ...next,
    a2uiMessages: [...next.a2uiMessages],
    terminalRunIds: [...next.terminalRunIds],
  });
}

export function captureChatRunFields(state: ChatState): SessionChatRunState {
  return {
    runId: state.chatRunId,
    runPhase: state.chatRunPhase,
    stream: state.chatStream,
    reasoningStream: state.chatReasoningStream,
    streamStartedAt: state.chatStreamStartedAt,
    a2uiMessages: [...state.chatA2UIMessages],
    terminalRunIds: [...(state.chatTerminalRunIds ?? [])],
    errorRunId: state.chatErrorRunId ?? null,
    sending: state.chatSending,
  };
}

function normalizeRunSnapshot(snap: SessionChatRunState): SessionChatRunState {
  let next = { ...snap, a2uiMessages: [...snap.a2uiMessages], terminalRunIds: [...snap.terminalRunIds] };
  if (next.runId && next.terminalRunIds.includes(next.runId)) {
    next = {
      ...next,
    runId: null,
    runPhase: "idle",
    stream: null,
    reasoningStream: null,
    streamStartedAt: null,
      a2uiMessages: [],
      sending: false,
    };
  }
  if (!next.runId && next.stream === "") {
    next = { ...next, stream: null };
  }
  return next;
}

export function applyChatRunFields(state: ChatState, snap: SessionChatRunState) {
  const normalized = normalizeRunSnapshot(snap);
  state.chatRunId = normalized.runId;
  state.chatRunPhase = normalized.runPhase;
  state.chatStream = normalized.stream;
  state.chatReasoningStream = normalized.reasoningStream;
  state.chatStreamStartedAt = normalized.streamStartedAt;
  state.chatA2UIMessages = [...normalized.a2uiMessages];
  state.chatTerminalRunIds = [...normalized.terminalRunIds];
  state.chatErrorRunId = normalized.errorRunId;
  state.chatSending = normalized.sending;
}

export function persistSessionRunState(sessionKey: string, snap: SessionChatRunState) {
  const key = sessionKey?.trim();
  if (!key) {
    return;
  }
  putSessionChatRunState(key, snap);
}

export function persistActiveSessionRunState(state: ChatState) {
  const key = state.sessionKey?.trim();
  if (!key) {
    return;
  }
  putSessionChatRunState(key, captureChatRunFields(state));
}

/** Merge in-memory run fields when sessionKey is the active UI session. */
export function resolveSessionChatRunState(
  sessionKey: string,
  live?: Pick<
    ChatState,
    | "sessionKey"
    | "chatRunId"
    | "chatRunPhase"
    | "chatStream"
    | "chatReasoningStream"
    | "chatStreamStartedAt"
    | "chatA2UIMessages"
    | "chatTerminalRunIds"
    | "chatErrorRunId"
    | "chatSending"
  > | null,
): SessionChatRunState {
  const snap = getSessionChatRunState(sessionKey);
  if (!live?.sessionKey || !gatewaySessionKeysEqual(live.sessionKey, sessionKey)) {
    return snap;
  }
  return captureChatRunFields(live as ChatState);
}

export function restoreSessionRunState(state: ChatState, sessionKey: string) {
  applyChatRunFields(state, getSessionChatRunState(sessionKey));
}

/** Save current session run state, switch key, restore target session run state. */
export function switchChatSessionRunState(state: ChatState, nextSessionKey: string) {
  persistActiveSessionRunState(state);
  state.sessionKey = nextSessionKey;
  restoreSessionRunState(state, nextSessionKey);
}

export function isSessionChatBusy(
  sessionKey: string,
  live?: Pick<ChatState, "sessionKey" | "chatRunId" | "chatSending" | "chatTerminalRunIds"> | null,
): boolean {
  const fullLive = live
    ? ({
        ...live,
        chatRunPhase: "idle" as const,
        chatStream: null,
        chatReasoningStream: null,
        chatStreamStartedAt: null,
        chatA2UIMessages: [],
        chatErrorRunId: null,
      } satisfies Parameters<typeof resolveSessionChatRunState>[1])
    : null;
  const snap = normalizeRunSnapshot(resolveSessionChatRunState(sessionKey, fullLive));
  const runActive = Boolean(snap.runId) && snap.runId
    ? !snap.terminalRunIds.includes(snap.runId.trim())
    : false;
  return snap.sending || runActive;
}

function markTerminal(snap: SessionChatRunState, runId: string): SessionChatRunState {
  const id = runId.trim();
  if (!id || snap.terminalRunIds.includes(id)) {
    return snap;
  }
  return {
    ...snap,
    terminalRunIds: [...snap.terminalRunIds, id].slice(-TERMINAL_RUN_LIMIT),
  };
}

function isTerminal(snap: SessionChatRunState, runId: string): boolean {
  const id = runId.trim();
  return id !== "" && snap.terminalRunIds.includes(id);
}

function clearActiveRun(snap: SessionChatRunState, runId?: string): SessionChatRunState {
  let next: SessionChatRunState = {
    ...snap,
    runId: null,
    runPhase: "idle",
    stream: null,
    reasoningStream: null,
    streamStartedAt: null,
    a2uiMessages: [],
    sending: false,
  };
  if (runId?.trim()) {
    next = markTerminal(next, runId);
  }
  return next;
}

/**
 * Apply chat WS events for a session that is not currently visible so its run
 * state stays accurate when the user switches back.
 */
export function applyBackgroundChatEvent(sessionKey: string, payload: ChatEventPayload) {
  const snap = getSessionChatRunState(sessionKey);
  const runId = typeof payload.runId === "string" ? payload.runId.trim() : "";

  if (payload.state === "a2ui" && payload.a2ui != null) {
    if (
      !runId ||
      !snap.runId ||
      snap.runId === runId ||
      isTerminal(snap, runId)
    ) {
      putSessionChatRunState(sessionKey, {
        ...snap,
        runId: snap.runId ?? (runId || null),
        a2uiMessages: dedupeA2UIMessages([...snap.a2uiMessages, payload.a2ui]),
        runPhase: "streaming",
      });
    }
    return;
  }

  if (runId && isTerminal(snap, runId)) {
    return;
  }

  if (runId && snap.runId && runId !== snap.runId) {
    if (payload.state === "final" || payload.state === "complete") {
      putSessionChatRunState(sessionKey, clearActiveRun(snap, runId));
    }
    return;
  }

  if (payload.state === "delta") {
    const reasoningChunk = typeof payload.reasoning === "string" ? payload.reasoning : "";
    if (reasoningChunk) {
      const current = snap.reasoningStream ?? "";
      putSessionChatRunState(sessionKey, {
        ...snap,
        runId: snap.runId ?? (runId || null),
        reasoningStream: current + reasoningChunk,
        runPhase: "thinking",
      });
      return;
    }
    const chunk = typeof payload.text === "string" ? payload.text : "";
    if (!chunk) {
      return;
    }
    const current = snap.stream ?? "";
    const stream =
      current && chunk.length >= current.length && chunk.startsWith(current)
        ? chunk
        : current + chunk;
    putSessionChatRunState(sessionKey, {
      ...snap,
      runId: snap.runId ?? (runId || null),
      stream,
      runPhase: "streaming",
    });
    return;
  }

  if (payload.state === "turn") {
    const turnThinking = extractThinking(payload.message)?.trim() ?? "";
    putSessionChatRunState(sessionKey, {
      ...snap,
      runId: snap.runId ?? (runId || null),
      reasoningStream: turnThinking ? null : snap.reasoningStream,
      a2uiMessages: [],
      runPhase: "tool",
    });
    return;
  }

  if (
    payload.state === "final" ||
    payload.state === "complete" ||
    payload.state === "aborted" ||
    payload.state === "error"
  ) {
    putSessionChatRunState(sessionKey, clearActiveRun(snap, runId));
  }
}

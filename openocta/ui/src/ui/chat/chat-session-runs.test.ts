import { describe, expect, it } from "vitest";
import {
  applyBackgroundChatEvent,
  getSessionChatRunState,
  isSessionChatBusy,
  persistSessionRunState,
} from "./chat-session-runs.ts";

describe("chat-session-runs", () => {
  it("tracks background runId and stream for another session", () => {
    const sessionKey = "custom:abc";
    persistSessionRunState(sessionKey, {
      runId: null,
      runPhase: "idle",
      stream: null,
      streamStartedAt: null,
      a2uiMessages: [],
      terminalRunIds: [],
      errorRunId: null,
      sending: false,
    });

    applyBackgroundChatEvent(sessionKey, {
      runId: "run-bg",
      sessionKey,
      state: "delta",
      text: "Hello",
    });

    const snap = getSessionChatRunState(sessionKey);
    expect(snap.runId).toBe("run-bg");
    expect(snap.stream).toBe("Hello");
    expect(isSessionChatBusy(sessionKey)).toBe(true);
  });

  it("clears background run on complete", () => {
    const sessionKey = "custom:def";
    persistSessionRunState(sessionKey, {
      runId: "run-bg",
      runPhase: "streaming",
      stream: "Hi",
      streamStartedAt: Date.now(),
      a2uiMessages: [],
      terminalRunIds: [],
      errorRunId: null,
      sending: false,
    });

    applyBackgroundChatEvent(sessionKey, {
      runId: "run-bg",
      sessionKey,
      state: "complete",
    });

    const snap = getSessionChatRunState(sessionKey);
    expect(snap.runId).toBeNull();
    expect(snap.stream).toBeNull();
    expect(isSessionChatBusy(sessionKey)).toBe(false);
  });
});

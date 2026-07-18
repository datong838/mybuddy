import { describe, expect, it } from "vitest";
import { cronJobSessionKeyFromRunSessionKey, cronRunHistorySessionKey, isCronRunSessionKey } from "./session-key-utils.js";

describe("isCronRunSessionKey", () => {
  it("matches current agent:cron:…:run:… keys", () => {
    expect(isCronRunSessionKey("agent:main:cron:job-1:run:abc")).toBe(true);
    expect(isCronRunSessionKey("AGENT:MAIN:cron:job-1:run:abc")).toBe(true);
  });

  it("matches legacy cron:…:run:… keys", () => {
    expect(isCronRunSessionKey("cron:job-1:run:sess-1")).toBe(true);
  });

  it("does not match main or persistent cron job keys", () => {
    expect(isCronRunSessionKey("main")).toBe(false);
    expect(isCronRunSessionKey("agent:main:main")).toBe(false);
    expect(isCronRunSessionKey("agent:main:cron:job-1")).toBe(false);
  });

  it("does not confuse employee run keys", () => {
    expect(isCronRunSessionKey("agent:main:employee:e1:run:uuid")).toBe(false);
  });
});

describe("cronJobSessionKeyFromRunSessionKey", () => {
  it("strips :run:… from agent cron run keys", () => {
    expect(cronJobSessionKeyFromRunSessionKey("agent:main:cron:job-1:run:abc")).toBe(
      "agent:main:cron:job-1",
    );
    expect(
      cronJobSessionKeyFromRunSessionKey(
        "agent:main:cron:e11d9767-9b27-45bf-97ee-94339b6e0898:run:sess-id",
      ),
    ).toBe("agent:main:cron:e11d9767-9b27-45bf-97ee-94339b6e0898");
  });

  it("returns persistent cron keys unchanged", () => {
    expect(cronJobSessionKeyFromRunSessionKey("agent:main:cron:e11d9767-9b27-45bf-97ee-94339b6e0898")).toBe(
      "agent:main:cron:e11d9767-9b27-45bf-97ee-94339b6e0898",
    );
  });

  it("normalizes legacy cron:job:run:…", () => {
    expect(cronJobSessionKeyFromRunSessionKey("cron:j1:run:s1")).toBe("agent:main:cron:j1");
  });
});

describe("cronRunHistorySessionKey", () => {
  it("keeps full agent cron run keys for history navigation", () => {
    expect(cronRunHistorySessionKey("agent:main:cron:job-1:run:abc")).toBe(
      "agent:main:cron:job-1:run:abc",
    );
  });

  it("keeps persistent cron job keys", () => {
    expect(cronRunHistorySessionKey("agent:main:cron:job-1")).toBe("agent:main:cron:job-1");
  });

  it("normalizes legacy cron run keys", () => {
    expect(cronRunHistorySessionKey("cron:j1:run:s1")).toBe("agent:main:cron:j1:run:s1");
  });

  it("rejects non-cron keys", () => {
    expect(cronRunHistorySessionKey("agent:main:oc_abc")).toBeNull();
  });
});

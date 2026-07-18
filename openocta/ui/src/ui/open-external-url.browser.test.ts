import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  DESKTOP_SHELL_QUERY_VALUE,
  SHELL_MODE_QUERY_PARAM,
  SHELL_MODE_SESSION_KEY,
  bootstrapShellModeFromUrl,
  getExternalUrlOpenMode,
  getShellMode,
  handleExternalAnchorClick,
  openExternalUrl,
} from "./open-external-url.ts";

describe("external URL helpers", () => {
  beforeEach(() => {
    sessionStorage.clear();
    document.body.innerHTML = "";
    window.history.replaceState({}, "", "/overview");
  });

  afterEach(() => {
    sessionStorage.clear();
    document.body.innerHTML = "";
    vi.restoreAllMocks();
  });

  it("stores desktop shell mode and removes the bootstrap query param", () => {
    window.history.replaceState(
      {},
      "",
      `/overview?${SHELL_MODE_QUERY_PARAM}=${DESKTOP_SHELL_QUERY_VALUE}&session=abc&onboarding=1#hash`,
    );

    bootstrapShellModeFromUrl();

    expect(sessionStorage.getItem(SHELL_MODE_SESSION_KEY)).toBe("desktop");
    expect(window.location.search).toBe("?session=abc&onboarding=1");
    expect(window.location.hash).toBe("#hash");
  });

  it("derives open mode from session storage", () => {
    expect(getShellMode()).toBe("browser");
    expect(getExternalUrlOpenMode()).toBe("new-window");

    sessionStorage.setItem(SHELL_MODE_SESSION_KEY, "desktop");

    expect(getShellMode()).toBe("desktop");
    expect(getExternalUrlOpenMode()).toBe("current-window");
  });

  it("intercepts plain external anchor clicks", () => {
    document.body.innerHTML = `<a id="docs" href="https://docs.openclaw.ai/web/dashboard" target="_blank">Docs</a>`;
    const link = document.getElementById("docs");
    const openUrl = vi.fn().mockResolvedValue(undefined);
    let intercepted = false;

    document.body.addEventListener("click", (event) => {
      intercepted = handleExternalAnchorClick(event as MouseEvent, { openUrl });
    });

    link?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 }));

    expect(intercepted).toBe(true);
    expect(openUrl).toHaveBeenCalledWith("https://docs.openclaw.ai/web/dashboard", {
      gatewayHost: undefined,
      gatewayToken: undefined,
    });
  });

  it("does not intercept same-origin or modified clicks", () => {
    document.body.innerHTML = `
      <a id="internal" href="/channels">Channels</a>
      <a id="external" href="https://docs.openclaw.ai/web/dashboard">Docs</a>
    `;
    const internal = document.getElementById("internal");
    const external = document.getElementById("external");
    const openUrl = vi.fn().mockResolvedValue(undefined);
    const seen: boolean[] = [];

    document.body.addEventListener("click", (event) => {
      seen.push(handleExternalAnchorClick(event as MouseEvent, { openUrl }));
    });

    internal?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 }));
    external?.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true, button: 0, ctrlKey: true }),
    );

    expect(seen).toEqual([false, false]);
    expect(openUrl).not.toHaveBeenCalled();
  });

  it("opens external URLs in a new window for browser mode", async () => {
    const popup = { opener: window } as Window;
    const openSpy = vi.spyOn(window, "open").mockReturnValue(popup);

    await openExternalUrl("https://community.databuff.com/c/10-category/10");

    expect(openSpy).toHaveBeenCalledWith("https://community.databuff.com/c/10-category/10", "_blank");
    expect(popup.opener).toBeNull();
    expect(window.location.pathname).toBe("/overview");
  });
});

import { describe, expect, it } from "vitest";
import {
  compareSessionSidebarRows,
  isAutoCustomSessionLabel,
  isSessionIdDerivedFallback,
  resolveSessionSidebarSubtitle,
  resolveSessionSidebarTitle,
} from "./session-sidebar.ts";

describe("session-sidebar", () => {
  it("detects auto custom session labels", () => {
    expect(isAutoCustomSessionLabel("自定义会话1")).toBe(true);
    expect(isAutoCustomSessionLabel("自定义对话2")).toBe(true);
    expect(isAutoCustomSessionLabel("我的会话")).toBe(false);
  });

  it("keeps auto label as title for custom sessions", () => {
    const title = resolveSessionSidebarTitle({
      key: "custom:abc",
      derivedTitle: "帮我写一份周报",
      label: "自定义会话3",
    });
    expect(title).toBe("自定义会话3");
    expect(
      resolveSessionSidebarSubtitle(title, "最后一条消息", "帮我写一份周报"),
    ).toBe("帮我写一份周报");
  });

  it("falls back to new chat for empty custom sessions", () => {
    const title = resolveSessionSidebarTitle({
      key: "custom:abc",
      label: "",
    });
    expect(title).toBe("新对话");
  });

  it("sorts pinned sessions first", () => {
    const rows = [
      { pinnedAt: 0, updatedAt: 300 },
      { pinnedAt: 100, updatedAt: 100 },
      { pinnedAt: 0, updatedAt: 200 },
    ];
    rows.sort(compareSessionSidebarRows);
    expect(rows[0].pinnedAt).toBe(100);
    expect(rows[1].updatedAt).toBe(300);
  });

  it("hides duplicate subtitle", () => {
    expect(resolveSessionSidebarSubtitle("你好", "你好")).toBe("");
    expect(resolveSessionSidebarSubtitle("你好", "世界")).toBe("世界");
    expect(resolveSessionSidebarSubtitle("自定义会话1", null, "帮我写周报")).toBe("帮我写周报");
    expect(resolveSessionSidebarSubtitle("自定义会话1", "最新回复", "帮我写周报")).toBe("帮我写周报");
    expect(isSessionIdDerivedFallback("a1b2c3d4 (2026-06-17)")).toBe(true);
    expect(
      resolveSessionSidebarSubtitle("自定义会话1", "", "a1b2c3d4 (2026-06-17)"),
    ).toBe("");
  });
});

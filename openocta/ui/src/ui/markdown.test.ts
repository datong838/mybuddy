import { describe, expect, it } from "vitest";
import { normalizeMarkdownForChat, toSanitizedMarkdownHtml } from "./markdown";

describe("toSanitizedMarkdownHtml", () => {
  it("renders basic markdown", () => {
    const html = toSanitizedMarkdownHtml("Hello **world**");
    expect(html).toContain("<strong>world</strong>");
  });

  it("strips scripts and unsafe links", () => {
    const html = toSanitizedMarkdownHtml(
      [
        "<script>alert(1)</script>",
        "",
        "[x](javascript:alert(1))",
        "",
        "[ok](https://example.com)",
      ].join("\n"),
    );
    expect(html).not.toContain("<script");
    expect(html).not.toContain("javascript:");
    expect(html).toContain("https://example.com");
  });

  it("renders fenced code blocks", () => {
    const html = toSanitizedMarkdownHtml(["```ts", "console.log(1)", "```"].join("\n"));
    expect(html).toContain("<pre>");
    expect(html).toContain("<code");
    expect(html).toContain("console.log(1)");
  });

  it("preserves markdown list line breaks", () => {
    const html = toSanitizedMarkdownHtml("files:\n\n- CHANGELOG.md\n- UPLOAD_FORMAT.md");
    expect(html).toContain("<li>");
    expect(html).toContain("CHANGELOG.md");
  });

  it("preserves single newlines as line breaks", () => {
    const html = toSanitizedMarkdownHtml("line one\nline two\ndrwxr-xr-x");
    expect(html).toContain("<br>");
    expect(html).toContain("line one");
    expect(html).toContain("line two");
  });

  it("renders GFM tables with scroll wrapper", () => {
    const html = toSanitizedMarkdownHtml(
      [
        "| 时段 | 天气 | 气温 |",
        "|------|------|------|",
        "| 早上 | 有霾 | 28°C |",
      ].join("\n"),
    );
    expect(html).toContain("chat-md-table-wrap");
    expect(html).toContain("<table");
    expect(html).toContain("<thead>");
    expect(html).toContain("早上");
    expect(html).toContain("有霾");
  });

  it("normalizes heading merged with table row", () => {
    const normalized = normalizeMarkdownForChat(
      "## 📊大盘整体表现|指数 |收盘 |涨跌 |\n|------|------|------|\n|上证指数 |4073.90 | +1.16% |",
    );
    expect(normalized).toContain("## 📊大盘整体表现\n\n|指数");
    const html = toSanitizedMarkdownHtml(normalized);
    expect(html).toContain("<table");
    expect(html).toContain("上证指数");
  });

  it("converts csv fences into markdown tables", () => {
    const normalized = normalizeMarkdownForChat(
      ["```csv", "代码,名称,涨跌幅", "159819,人工智能ETF,+1.30%", "```"].join("\n"),
    );
    expect(normalized).toContain("| 代码 | 名称 | 涨跌幅 |");
    const html = toSanitizedMarkdownHtml(normalized);
    expect(html).toContain("<table");
    expect(html).toContain("159819");
  });

  it("does not split standalone Chinese headings", () => {
    const normalized = normalizeMarkdownForChat("## 📊大盘整体表现\n\n正文");
    expect(normalized).toContain("## 📊大盘整体表现");
    expect(normalized).not.toMatch(/## 📊大\n\n盘整体表现/);
  });

  it("splits glued list items on one line", () => {
    const normalized = normalizeMarkdownForChat(
      "-今日成交额 3.52万亿-上涨2469家-市场整体评分 6.9分",
    );
    expect(normalized).toContain("-今日成交额");
    expect(normalized).toContain("\n-上涨2469家");
  });
});

import DOMPurify from "dompurify";
import { marked } from "marked";
import { truncateText } from "./format.ts";

marked.setOptions({
  gfm: true,
  breaks: true,
});

/** Repair common model output glitches before markdown parse (merged headings/tables/lists). */
export function normalizeMarkdownForChat(markdown: string): string {
  if (!markdown) {
    return "";
  }
  let text = markdown.replace(/\r\n/g, "\n");

  // ## Title| col | col |  -> heading + table header on separate lines
  text = text.replace(/^(#{1,6}\s+[^\n|]+)(\|[^\n]+\|[^\n]*\|)\s*$/gm, "$1\n\n$2");

  // Section heading glued to body when a keyword runs into the next sentence (same line).
  text = text.replace(
    /^(#{1,6}\s+.*?(?:小结|热点|龙头|明显))([\u4e00-\u9fff][^\n|]+)$/gm,
    "$1\n\n$2",
  );

  // Horizontal rule glued to text: 状态---\n## Next
  text = text.replace(/([^\s-\n])---(?=\s*(?:#{1,6}|\n|$))/g, "$1\n\n---\n\n");

  // Multiple list items on one line: -foo-bar (split before - + CJK, not negative numbers)
  text = text.replace(/^(-[^\n]+)$/gm, (line) =>
    line.replace(/([^\d\s])-(?=[\u4e00-\u9fff])/g, "$1\n-"),
  );

  // Numbered list glued to heading or prior sentence: 热点1. ** -> 热点\n\n1. **
  text = text.replace(/([^\n\d])(\d+\.\s+\*\*)/g, "$1\n\n$2");
  text = text.replace(/([^\n\d])(\d+\.\s+[\u4e00-\u9fff])/g, "$1\n\n$2");

  // Convert ```csv fences into GFM tables for chat readability
  text = convertCsvFencesToMarkdownTables(text);

  // Ensure blank line before table blocks
  text = text.replace(/([^\n|])\n(\|[^|\n]+\|)/g, "$1\n\n$2");

  return text;
}

function convertCsvFencesToMarkdownTables(text: string): string {
  return text.replace(/```csv\n([\s\S]*?)```/gi, (_match, body: string) => {
    const lines = body
      .trim()
      .split(/\n/)
      .map((line) => line.trim())
      .filter(Boolean);
    if (lines.length < 2) {
      return `\`\`\`csv\n${body}\`\`\``;
    }
    const headers = lines[0].split(",").map((cell) => cell.trim());
    if (headers.length < 2) {
      return `\`\`\`csv\n${body}\`\`\``;
    }
    const maxRows = 15;
    const dataRows = lines.slice(1, 1 + maxRows);
    const overflow = lines.length - 1 - maxRows;
    const headerRow = `| ${headers.join(" | ")} |`;
    const separator = `| ${headers.map(() => "---").join(" | ")} |`;
    const bodyRows = dataRows.map((row) => {
      const cells = row.split(",").map((cell) => cell.trim());
      while (cells.length < headers.length) {
        cells.push("");
      }
      return `| ${cells.slice(0, headers.length).join(" | ")} |`;
    });
    const parts = [headerRow, separator, ...bodyRows];
    if (overflow > 0) {
      parts.push("", `> 另有 ${overflow} 行数据未展示`);
    }
    return parts.join("\n");
  });
}

const allowedTags = [
  "a",
  "b",
  "blockquote",
  "br",
  "code",
  "del",
  "div",
  "em",
  "h1",
  "h2",
  "h3",
  "h4",
  "hr",
  "i",
  "img",
  "li",
  "ol",
  "p",
  "pre",
  "strong",
  "table",
  "tbody",
  "td",
  "th",
  "thead",
  "tr",
  "ul",
];

const allowedAttrs = ["class", "href", "rel", "target", "title", "start", "src", "alt"];

let hooksInstalled = false;
const MARKDOWN_CHAR_LIMIT = 140_000;
const MARKDOWN_PARSE_LIMIT = 40_000;
const MARKDOWN_CACHE_LIMIT = 200;
const MARKDOWN_CACHE_MAX_CHARS = 50_000;
const markdownCache = new Map<string, string>();

function getCachedMarkdown(key: string): string | null {
  const cached = markdownCache.get(key);
  if (cached === undefined) {
    return null;
  }
  markdownCache.delete(key);
  markdownCache.set(key, cached);
  return cached;
}

function setCachedMarkdown(key: string, value: string) {
  markdownCache.set(key, value);
  if (markdownCache.size <= MARKDOWN_CACHE_LIMIT) {
    return;
  }
  const oldest = markdownCache.keys().next().value;
  if (oldest) {
    markdownCache.delete(oldest);
  }
}

function installHooks() {
  if (hooksInstalled) {
    return;
  }
  hooksInstalled = true;

  DOMPurify.addHook("afterSanitizeAttributes", (node) => {
    if (node instanceof HTMLAnchorElement) {
      const href = node.getAttribute("href");
      if (!href) {
        return;
      }
      if (!href.includes("://") && /^attachments\//i.test(href)) {
        node.setAttribute("data-chat-attachment", href);
        node.setAttribute("href", "#");
        node.classList.add("chat-attachment-link");
        return;
      }
      node.setAttribute("rel", "noreferrer noopener");
      node.setAttribute("target", "_blank");
      return;
    }
    if (node instanceof HTMLImageElement) {
      const src = node.getAttribute("src");
      if (!src || !isAllowedImageSrc(src)) {
        node.removeAttribute("src");
      }
    }
  });
}

export function toSanitizedMarkdownHtml(markdown: string): string {
  if (!markdown) {
    return "";
  }
  if (!markdown.trim()) {
    return "";
  }
  installHooks();
  const input = markdown;
  if (input.length <= MARKDOWN_CACHE_MAX_CHARS) {
    const cached = getCachedMarkdown(input);
    if (cached !== null) {
      return cached;
    }
  }
  const truncated = truncateText(input, MARKDOWN_CHAR_LIMIT);
  const suffix = truncated.truncated
    ? `\n\n… truncated (${truncated.total} chars, showing first ${truncated.text.length}).`
    : "";
  if (truncated.text.length > MARKDOWN_PARSE_LIMIT) {
    const escaped = escapeHtml(`${truncated.text}${suffix}`);
    const html = `<pre class="code-block">${escaped}</pre>`;
    const sanitized = sanitizeMarkdownHtml(html);
    if (input.length <= MARKDOWN_CACHE_MAX_CHARS) {
      setCachedMarkdown(input, sanitized);
    }
    return sanitized;
  }
  const rendered = marked.parse(`${normalizeMarkdownForChat(truncated.text)}${suffix}`) as string;
  const sanitized = sanitizeMarkdownHtml(rendered);
  if (input.length <= MARKDOWN_CACHE_MAX_CHARS) {
    setCachedMarkdown(input, sanitized);
  }
  return sanitized;
}

/** Wrap GFM tables so horizontal scroll works without breaking table layout. */
function wrapMarkdownTables(html: string): string {
  return html.replace(/<table\b[\s\S]*?<\/table>/gi, (table) => {
    return `<div class="chat-md-table-wrap">${table}</div>`;
  });
}

function sanitizeMarkdownHtml(html: string): string {
  const wrapped = wrapMarkdownTables(html);
  return DOMPurify.sanitize(wrapped, {
    ALLOWED_TAGS: allowedTags,
    ALLOWED_ATTR: allowedAttrs,
  });
}

function isAllowedImageSrc(src: string): boolean {
  const trimmed = src.trim().toLowerCase();
  return (
    trimmed.startsWith("data:image/") ||
    trimmed.startsWith("https://") ||
    trimmed.startsWith("http://")
  );
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

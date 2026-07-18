import { html, nothing, type TemplateResult } from "lit";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { toSanitizedMarkdownHtml } from "../markdown.ts";

export type FilePreviewKind =
  | "markdown"
  | "csv"
  | "json"
  | "yaml"
  | "xml"
  | "code"
  | "text";

export type FilePreviewVariant = "inline" | "modal";

function extFromFilename(filename: string): string {
  const idx = filename.lastIndexOf(".");
  return idx >= 0 ? filename.slice(idx + 1).toLowerCase() : "";
}

export function resolveTextFilePreviewKind(filename: string, mimeType: string): FilePreviewKind {
  const ext = extFromFilename(filename);
  const mime = mimeType.toLowerCase();
  if (ext === "md" || ext === "markdown") {
    return "markdown";
  }
  if (ext === "csv" || mime.includes("csv")) {
    return "csv";
  }
  if (ext === "json" || mime.includes("json")) {
    return "json";
  }
  if (ext === "yaml" || ext === "yml" || mime.includes("yaml")) {
    return "yaml";
  }
  if (ext === "xml" || mime.includes("xml")) {
    return "xml";
  }
  if (["log", "txt", "ini", "conf", "cfg", "env"].includes(ext)) {
    return "text";
  }
  if (["ts", "tsx", "js", "jsx", "go", "py", "rs", "java", "sql", "sh", "bash"].includes(ext)) {
    return "code";
  }
  return "text";
}

export function truncatePreviewText(text: string, maxChars: number): string {
  if (text.length <= maxChars) {
    return text;
  }
  return `${text.slice(0, maxChars)}…`;
}

/** Parse a single CSV line respecting quoted fields. */
function parseCsvLine(line: string): string[] {
  const out: string[] = [];
  let cur = "";
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const c = line[i];
    if (c === '"') {
      if (inQuotes && line[i + 1] === '"') {
        cur += '"';
        i++;
      } else {
        inQuotes = !inQuotes;
      }
    } else if (c === "," && !inQuotes) {
      out.push(cur);
      cur = "";
    } else {
      cur += c;
    }
  }
  out.push(cur);
  return out;
}

export function parseCsvTable(text: string, maxDataRows = Number.POSITIVE_INFINITY): {
  headers: string[];
  rows: string[][];
  hasMore: boolean;
} {
  const lines = text.split(/\r?\n/).filter((line) => line.trim().length > 0);
  if (lines.length === 0) {
    return { headers: [], rows: [], hasMore: false };
  }
  const headers = parseCsvLine(lines[0] ?? "");
  const dataLines = lines.slice(1);
  const rows = dataLines.slice(0, maxDataRows).map(parseCsvLine);
  return { headers, rows, hasMore: dataLines.length > maxDataRows };
}

function renderCsvTable(
  text: string,
  variant: FilePreviewVariant,
): TemplateResult {
  const maxRows = variant === "inline" ? 4 : Number.POSITIVE_INFINITY;
  const { headers, rows, hasMore } = parseCsvTable(text, maxRows);
  if (headers.length === 0 && rows.length === 0) {
    return html`<pre class="file-preview-plain">${truncatePreviewText(text, variant === "inline" ? 480 : 200_000)}</pre>`;
  }
  const colCount = Math.max(headers.length, ...rows.map((r) => r.length));
  const normalizedHeaders =
    headers.length >= colCount
      ? headers
      : [...headers, ...Array.from({ length: colCount - headers.length }, () => "")];
  return html`
    <div class="file-preview-table-wrap file-preview-table-wrap--${variant}">
      <table class="file-preview-table">
        <thead>
          <tr>
            ${normalizedHeaders.map((cell) => html`<th>${cell}</th>`)}
          </tr>
        </thead>
        <tbody>
          ${rows.map(
            (row) => html`
              <tr>
                ${Array.from({ length: colCount }, (_, i) => html`<td>${row[i] ?? ""}</td>`)}
              </tr>
            `,
          )}
        </tbody>
      </table>
      ${variant === "inline" && hasMore
        ? html`<div class="file-preview-table-more muted">…</div>`
        : nothing}
    </div>
  `;
}

function renderJsonBody(text: string, variant: FilePreviewVariant): TemplateResult {
  const limit = variant === "inline" ? 480 : 500_000;
  try {
    const pretty = JSON.stringify(JSON.parse(text), null, 2);
    return html`<pre class="file-preview-code">${truncatePreviewText(pretty, limit)}</pre>`;
  } catch {
    return html`<pre class="file-preview-plain">${truncatePreviewText(text, limit)}</pre>`;
  }
}

function renderMarkdownBody(text: string, variant: FilePreviewVariant): TemplateResult {
  if (variant === "inline") {
    return html`<div class="file-preview-markdown file-preview-markdown--inline chat-text">${unsafeHTML(toSanitizedMarkdownHtml(truncatePreviewText(text, 600)))}</div>`;
  }
  return html`<div class="file-preview-markdown chat-text">${unsafeHTML(toSanitizedMarkdownHtml(text))}</div>`;
}

export function renderTextFilePreviewBody(
  text: string,
  filename: string,
  mimeType: string,
  variant: FilePreviewVariant,
): TemplateResult {
  const kind = resolveTextFilePreviewKind(filename, mimeType);
  const charLimit = variant === "inline" ? 480 : 500_000;
  switch (kind) {
    case "csv":
      return renderCsvTable(text, variant);
    case "json":
      return renderJsonBody(text, variant);
    case "markdown":
      return renderMarkdownBody(text, variant);
    case "yaml":
    case "xml":
    case "code":
      return html`<pre class="file-preview-code file-preview-code--${kind}">${truncatePreviewText(text, charLimit)}</pre>`;
    case "text":
    default:
      return html`<pre class="file-preview-plain">${truncatePreviewText(text, charLimit)}</pre>`;
  }
}

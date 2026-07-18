import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { extractA2UIDisplayTextFromBlocks, normalizeA2UIMessages } from "./a2ui-bridge.ts";
import {
  componentsArrayFromRaw,
  normalizeComponentMap,
  type ComponentRecord,
} from "./a2ui-components.ts";
import { renderTextFilePreviewBody } from "./file-preview-content.ts";
import { parseOpenOctaFileAttachmentsFromText } from "./openocta-attachments.ts";
import { extractReferencedImagePaths } from "./attachment-images.ts";

export type FileBlock = {
  filename: string;
  mimeType: string;
  url: string;
  sizeBytes?: number;
  isPreviewable: boolean;
};

const PREVIEWABLE_MIME_PREFIXES = ["text/", "application/json", "application/yaml", "application/x-yaml"];
const PREVIEWABLE_EXTENSIONS = new Set([
  "md",
  "markdown",
  "txt",
  "json",
  "yaml",
  "yml",
  "xml",
  "csv",
  "log",
  "pdf",
  "html",
  "htm",
]);

function extFromName(name: string): string {
  const idx = name.lastIndexOf(".");
  return idx >= 0 ? name.slice(idx + 1).toLowerCase() : "";
}

function isPreviewableMime(mimeType: string, filename: string): boolean {
  const mime = mimeType.toLowerCase();
  if (mime === "application/pdf") {
    return true;
  }
  if (PREVIEWABLE_MIME_PREFIXES.some((p) => mime.startsWith(p))) {
    return true;
  }
  return PREVIEWABLE_EXTENSIONS.has(extFromName(filename));
}

function resolveBlockUrl(
  data: string | undefined,
  mimeType: string,
  url?: string,
): string | null {
  if (url?.trim()) {
    return url.trim();
  }
  if (!data?.trim()) {
    return null;
  }
  if (data.startsWith("data:")) {
    return data;
  }
  const media = mimeType || "application/octet-stream";
  return `data:${media};base64,${data}`;
}

function readSource(block: Record<string, unknown>): { data?: string; mimeType?: string } {
  const source = block.source as Record<string, unknown> | undefined;
  if (!source) {
    return {};
  }
  const data = typeof source.data === "string" ? source.data : undefined;
  const mimeType =
    (typeof source.media_type === "string" && source.media_type) ||
    (typeof source.mimeType === "string" && source.mimeType) ||
    undefined;
  return { data, mimeType };
}

function estimateSizeFromUrl(url: string): number | undefined {
  if (!url.startsWith("data:")) {
    return undefined;
  }
  const comma = url.indexOf(",");
  if (comma < 0) {
    return undefined;
  }
  const meta = url.slice(5, comma);
  const payload = url.slice(comma + 1);
  if (meta.includes(";base64")) {
    const padding = payload.endsWith("==") ? 2 : payload.endsWith("=") ? 1 : 0;
    return Math.max(0, Math.floor((payload.length * 3) / 4) - padding);
  }
  try {
    return decodeURIComponent(payload).length;
  } catch {
    return payload.length;
  }
}

function pushFileBlock(files: FileBlock[], next: FileBlock) {
  const key = `${next.filename}::${next.url}`;
  if (files.some((f) => `${f.filename}::${f.url}` === key)) {
    return;
  }
  files.push(next);
}

function readComponentString(comp: ComponentRecord, key: string): string | undefined {
  const direct = comp[key];
  if (typeof direct === "string" && direct.trim()) {
    return direct.trim();
  }
  const props = comp.properties;
  if (props != null && typeof props === "object" && !Array.isArray(props)) {
    const nested = (props as ComponentRecord)[key];
    if (typeof nested === "string" && nested.trim()) {
      return nested.trim();
    }
  }
  return undefined;
}

function filenameFromUrl(url: string, fallback: string): string {
  if (url.startsWith("data:")) {
    const mime = url.slice(5, url.indexOf(";")) || "application/octet-stream";
    if (mime.startsWith("image/")) {
      const ext = mime.split("/")[1]?.split("+")[0] || "png";
      return `${fallback}.${ext}`;
    }
    return fallback;
  }
  try {
    const parsed = new URL(url, "http://local");
    const base = parsed.pathname.split("/").pop();
    if (base?.trim()) {
      return base.trim();
    }
  } catch {
    // ignore
  }
  return fallback;
}

function mimeTypeFromUrl(url: string, fallback: string): string {
  if (url.startsWith("data:")) {
    const semi = url.indexOf(";");
    const comma = url.indexOf(",");
    const end = semi >= 0 ? semi : comma >= 0 ? comma : url.length;
    const mime = url.slice(5, end).trim();
    if (mime) {
      return mime;
    }
  }
  const ext = extFromName(filenameFromUrl(url, fallback));
  if (ext === "png") return "image/png";
  if (ext === "jpg" || ext === "jpeg") return "image/jpeg";
  if (ext === "gif") return "image/gif";
  if (ext === "webp") return "image/webp";
  if (ext === "pdf") return "application/pdf";
  if (ext === "html" || ext === "htm") return "text/html";
  return fallback;
}

function fileBlockFromA2UIImageComponent(comp: ComponentRecord): FileBlock | null {
  const url = readComponentString(comp, "url");
  if (!url) {
    return null;
  }
  const label = readComponentString(comp, "description") || "image";
  const filename = filenameFromUrl(url, label);
  const mimeType = mimeTypeFromUrl(url, "image/png");
  return {
    filename,
    mimeType,
    url,
    sizeBytes: estimateSizeFromUrl(url),
    isPreviewable: true,
  };
}

/** Extract downloadable file blocks embedded in A2UI server messages (Text markers, Image URLs). */
export function extractFileBlocksFromA2UIBlocks(blocks: unknown[]): FileBlock[] {
  if (!Array.isArray(blocks) || blocks.length === 0) {
    return [];
  }
  const files: FileBlock[] = [];
  const repaired = normalizeA2UIMessages(blocks);
  for (const msg of repaired) {
    const rawComponents = msg.updateComponents?.components;
    if (rawComponents == null) {
      continue;
    }
    for (const raw of componentsArrayFromRaw(rawComponents)) {
      const comp = normalizeComponentMap(raw);
      const type = typeof comp.component === "string" ? comp.component : "";
      if (type === "Text") {
        const text = readComponentString(comp, "text");
        if (text) {
          for (const parsed of parseOpenOctaFileAttachmentsFromText(text)) {
            pushFileBlock(files, {
              filename: parsed.filename,
              mimeType: parsed.mimeType,
              url: parsed.url,
              sizeBytes: parsed.sizeBytes ?? estimateSizeFromUrl(parsed.url),
              isPreviewable: isPreviewableMime(parsed.mimeType, parsed.filename),
            });
          }
        }
        continue;
      }
      if (type === "Image") {
        const imageBlock = fileBlockFromA2UIImageComponent(comp);
        if (imageBlock) {
          pushFileBlock(files, imageBlock);
        }
      }
    }
  }
  return files;
}

export function dedupeFileBlocks(files: FileBlock[]): FileBlock[] {
  const out: FileBlock[] = [];
  for (const file of files) {
    pushFileBlock(out, file);
  }
  return out;
}

export function extractFileBlocks(message: unknown): FileBlock[] {
  const m = message as Record<string, unknown>;
  const content = m.content;
  const files: FileBlock[] = [];

  if (typeof content === "string") {
    for (const parsed of parseOpenOctaFileAttachmentsFromText(content)) {
      pushFileBlock(files, {
        filename: parsed.filename,
        mimeType: parsed.mimeType,
        url: parsed.url,
        sizeBytes: parsed.sizeBytes ?? estimateSizeFromUrl(parsed.url),
        isPreviewable: isPreviewableMime(parsed.mimeType, parsed.filename),
      });
    }
  }

  if (!Array.isArray(content)) {
    return files;
  }

  for (const part of content) {
    if (typeof part !== "object" || part === null) {
      continue;
    }
    const b = part as Record<string, unknown>;
    const kind = (typeof b.type === "string" ? b.type : "").toLowerCase();

    if (kind === "text" && typeof b.text === "string") {
      for (const parsed of parseOpenOctaFileAttachmentsFromText(b.text)) {
        pushFileBlock(files, {
          filename: parsed.filename,
          mimeType: parsed.mimeType,
          url: parsed.url,
          sizeBytes: parsed.sizeBytes ?? estimateSizeFromUrl(parsed.url),
          isPreviewable: isPreviewableMime(parsed.mimeType, parsed.filename),
        });
      }
      continue;
    }

    if (kind === "a2ui" && b.a2ui != null) {
      for (const parsed of extractFileBlocksFromA2UIBlocks([b.a2ui])) {
        pushFileBlock(files, parsed);
      }
      continue;
    }

    if (kind !== "file" && kind !== "document" && kind !== "attachment") {
      continue;
    }
    const mimeType =
      (typeof b.mimeType === "string" && b.mimeType) ||
      (typeof b.media_type === "string" && b.media_type) ||
      "application/octet-stream";
    if (mimeType.toLowerCase().startsWith("image/")) {
      continue;
    }
    const filename =
      (typeof b.filename === "string" && b.filename) ||
      (typeof b.name === "string" && b.name) ||
      "download";
    const { data: sourceData, mimeType: sourceMime } = readSource(b);
    const data =
      sourceData ||
      (typeof b.data === "string" ? b.data : undefined);
    const url = resolveBlockUrl(data, sourceMime || mimeType, typeof b.url === "string" ? b.url : undefined);
    if (!url) {
      continue;
    }
    pushFileBlock(files, {
      filename,
      mimeType: sourceMime || mimeType,
      url,
      sizeBytes:
        (typeof b.sizeBytes === "number" ? b.sizeBytes : undefined) ?? estimateSizeFromUrl(url),
      isPreviewable: isPreviewableMime(sourceMime || mimeType, filename),
    });
  }
  return files;
}

const referencedAttachmentPathPattern = /attachments\/[\w./-]+\.html?/gi;
const PREVIEWABLE_FILE_EXTENSIONS = "md|markdown|txt|json|ya?ml|csv|log|xml|pdf|html?|htm";
const referencedPathDelimiter = String.raw`[\s"'(\[\x60\uFF1A:]`;
const referencedPreviewablePathPattern = new RegExp(
  `(?:^|${referencedPathDelimiter})([A-Za-z0-9._-]+(?:[\\\\/][A-Za-z0-9._-]+)*\\.(?:${PREVIEWABLE_FILE_EXTENSIONS}))`,
  "gi",
);
const referencedAbsolutePreviewablePathPattern = new RegExp(
  `(?:^|${referencedPathDelimiter})(/(?:[A-Za-z0-9._-]+(?:[\\\\/][A-Za-z0-9._-]+)*)\\.(?:${PREVIEWABLE_FILE_EXTENSIONS}))`,
  "gi",
);
const referencedHomePreviewablePathPattern = new RegExp(
  `(?:^|${referencedPathDelimiter})(~(?:[A-Za-z0-9._-]+(?:[\\\\/][A-Za-z0-9._-]+)*)\\.(?:${PREVIEWABLE_FILE_EXTENSIONS}))`,
  "gi",
);

function normalizeReferencedPath(raw: string): string {
  return raw.trim().replace(/^[`'"]+|[`'"]+$/g, "");
}

function pushReferencedPath(paths: string[], seen: Set<string>, raw: string) {
  const path = normalizeReferencedPath(raw);
  if (!path || seen.has(path)) {
    return;
  }
  seen.add(path);
  paths.push(path);
}

export function extractReferencedAttachmentPaths(text: string): string[] {
  const seen = new Set<string>();
  const paths: string[] = [];
  for (const raw of text.match(referencedAttachmentPathPattern) ?? []) {
    pushReferencedPath(paths, seen, raw);
  }
  for (const match of text.matchAll(referencedPreviewablePathPattern)) {
    pushReferencedPath(paths, seen, match[1] ?? "");
  }
  for (const match of text.matchAll(referencedAbsolutePreviewablePathPattern)) {
    pushReferencedPath(paths, seen, match[1] ?? "");
  }
  for (const match of text.matchAll(referencedHomePreviewablePathPattern)) {
    pushReferencedPath(paths, seen, match[1] ?? "");
  }
  for (const path of extractReferencedImagePaths(text)) {
    pushReferencedPath(paths, seen, path);
  }
  return paths;
}

export function renderImageFileBlocks(
  files: FileBlock[],
  onPreview?: (req: FilePreviewRequest) => void,
) {
  const images = files.filter((file) => file.mimeType.toLowerCase().startsWith("image/"));
  if (images.length === 0) {
    return nothing;
  }
  return html`
    <div class="chat-message-images">
      ${images.map((file) => {
        const label = file.filename;
        return html`
          <div class="chat-message-image-wrap">
            <img
              src=${file.url}
              alt=${label}
              class="chat-message-image"
              @click=${() => {
                if (onPreview) {
                  onPreview({
                    filename: file.filename,
                    mimeType: file.mimeType,
                    url: file.url,
                  });
                  return;
                }
                window.open(file.url, "_blank");
              }}
            />
            <div class="chat-message-image-actions">
              ${renderFileIconButtons(file, onPreview)}
            </div>
          </div>
        `;
      })}
    </div>
  `;
}

export function fileBlockFromGatewayPayload(payload: Record<string, unknown>): FileBlock | null {
  const filename =
    (typeof payload.filename === "string" && payload.filename) ||
    (typeof payload.name === "string" && payload.name) ||
    "download";
  const mimeType =
    (typeof payload.mimeType === "string" && payload.mimeType) ||
    (typeof payload.media_type === "string" && payload.media_type) ||
    "application/octet-stream";
  const source = payload.source as Record<string, unknown> | undefined;
  const data =
    (typeof source?.data === "string" && source.data) ||
    (typeof payload.data === "string" && payload.data) ||
    undefined;
  const url = resolveBlockUrl(data, mimeType, typeof payload.url === "string" ? payload.url : undefined);
  if (!url) {
    return null;
  }
  return {
    filename,
    mimeType,
    url,
    sizeBytes:
      (typeof payload.sizeBytes === "number" ? payload.sizeBytes : undefined) ?? estimateSizeFromUrl(url),
    isPreviewable: isPreviewableMime(mimeType, filename),
  };
}

export function extractReferencedPathsFromGroup(messages: Array<{ message: unknown }>): string[] {
  const paths: string[] = [];
  for (const item of messages) {
    const m = item.message as Record<string, unknown>;
    const content = m.content;
    if (typeof content === "string") {
      paths.push(...extractReferencedAttachmentPaths(content));
      continue;
    }
    if (!Array.isArray(content)) {
      const text = typeof m.text === "string" ? m.text : "";
      if (text) {
        paths.push(...extractReferencedAttachmentPaths(text));
      }
      continue;
    }
    for (const part of content) {
      if (typeof part !== "object" || part === null) {
        continue;
      }
      const block = part as Record<string, unknown>;
      if (block.type === "text" && typeof block.text === "string") {
        paths.push(...extractReferencedAttachmentPaths(block.text));
      }
      if (block.type === "a2ui" && block.a2ui != null) {
        for (const file of extractFileBlocksFromA2UIBlocks([block.a2ui])) {
          paths.push(file.filename);
        }
        const repaired = normalizeA2UIMessages([block.a2ui]);
        const displayText = extractA2UIDisplayTextFromBlocks(repaired);
        if (displayText) {
          paths.push(...extractReferencedAttachmentPaths(displayText));
        }
        for (const msg of repaired) {
          const rawComponents = msg.updateComponents?.components;
          if (rawComponents == null) {
            continue;
          }
          for (const raw of componentsArrayFromRaw(rawComponents)) {
            const comp = normalizeComponentMap(raw);
            if (comp.component === "Text") {
              const text = readComponentString(comp, "text");
              if (text) {
                paths.push(...extractReferencedAttachmentPaths(text));
              }
            }
          }
        }
      }
    }
  }
  const seen = new Set<string>();
  return paths.filter((path) => {
    if (seen.has(path)) {
      return false;
    }
    seen.add(path);
    return true;
  });
}

export function extractGroupFileBlocks(messages: Array<{ message: unknown }>): FileBlock[] {
  const files: FileBlock[] = [];
  for (const item of messages) {
    for (const block of extractFileBlocks(item.message)) {
      pushFileBlock(files, block);
    }
  }
  return files;
}

export type FilePreviewRequest = {
  filename: string;
  mimeType: string;
  url: string;
};

function charsetFromDataUrlMeta(meta: string): string {
  const match = /;\s*charset=([^;]+)/i.exec(meta);
  if (match?.[1]?.trim()) {
    return match[1].trim();
  }
  return "utf-8";
}

function decodeBase64Text(payload: string, charset: string): string {
  const binary = atob(payload);
  const bytes = Uint8Array.from(binary, (c) => c.charCodeAt(0));
  try {
    return new TextDecoder(charset).decode(bytes);
  } catch {
    return new TextDecoder("utf-8").decode(bytes);
  }
}

export function decodeFileText(url: string): string | null {
  if (!url.startsWith("data:")) {
    return null;
  }
  const comma = url.indexOf(",");
  if (comma < 0) {
    return null;
  }
  const meta = url.slice(5, comma);
  const payload = url.slice(comma + 1);
  try {
    if (meta.includes(";base64")) {
      return decodeBase64Text(payload, charsetFromDataUrlMeta(meta));
    }
    return decodeURIComponent(payload);
  } catch {
    return null;
  }
}

function formatFileSize(sizeBytes?: number): string {
  if (sizeBytes == null || !Number.isFinite(sizeBytes)) {
    return "";
  }
  if (sizeBytes < 1024) {
    return `${sizeBytes} B`;
  }
  if (sizeBytes < 1024 * 1024) {
    return `${(sizeBytes / 1024).toFixed(sizeBytes < 10240 ? 1 : 0)} KB`;
  }
  return `${(sizeBytes / (1024 * 1024)).toFixed(1)} MB`;
}

function fileTypeLabel(filename: string, mimeType: string): string {
  const ext = extFromName(filename);
  if (ext === "html" || ext === "htm" || mimeType.includes("html")) {
    return "HTML";
  }
  if (ext === "markdown") {
    return "MD";
  }
  if (ext) {
    return ext.length <= 4 ? ext.toUpperCase() : ext.slice(0, 4).toUpperCase();
  }
  return "FILE";
}

function fileIconTone(filename: string, mimeType: string): string {
  const ext = extFromName(filename);
  if (ext === "html" || ext === "htm" || mimeType.includes("html")) {
    return "html";
  }
  if (ext === "md" || ext === "markdown") {
    return "md";
  }
  if (ext === "pdf" || mimeType.includes("pdf")) {
    return "pdf";
  }
  if (ext === "json" || mimeType.includes("json")) {
    return "json";
  }
  if (mimeType.toLowerCase().startsWith("image/")) {
    return "image";
  }
  return "default";
}

function filePreviewMode(file: FileBlock): "html" | "pdf" | "text" | null {
  if (!file.isPreviewable) {
    return null;
  }
  const ext = extFromName(file.filename);
  const mime = file.mimeType.toLowerCase();
  if (mime.includes("html") || ext === "html" || ext === "htm") {
    return "html";
  }
  if (mime.includes("pdf") || ext === "pdf") {
    return "pdf";
  }
  if (decodeFileText(file.url)) {
    return "text";
  }
  return null;
}

function toPreviewRequest(file: FileBlock): FilePreviewRequest {
  return {
    filename: file.filename,
    mimeType: file.mimeType,
    url: file.url,
  };
}

function renderFileEmbed(file: FileBlock, mode: "html" | "pdf" | "text") {
  if (mode === "html" || mode === "pdf") {
    return html`
      <iframe
        class="chat-file-embed"
        title=${file.filename}
        sandbox=""
        src=${file.url}
        loading="lazy"
      ></iframe>
    `;
  }
  const text = decodeFileText(file.url);
  if (!text) {
    return nothing;
  }
  return renderTextFilePreviewBody(text, file.filename, file.mimeType, "inline");
}

function renderFileIconCompact(filename: string, mimeType: string) {
  const label = fileTypeLabel(filename, mimeType);
  const tone = fileIconTone(filename, mimeType);
  return html`
    <div class="chat-file-card__icon chat-file-card__icon--compact chat-file-card__icon--${tone}" aria-hidden="true">
      <span class="chat-file-card__badge">${label}</span>
    </div>
  `;
}

function renderFileIconButtons(file: FileBlock, onPreview?: (req: FilePreviewRequest) => void) {
  return html`
    <div class="chat-file-icon-actions">
      ${
        file.isPreviewable && onPreview
          ? html`<button
              type="button"
              class="chat-file-icon-btn"
              title="预览"
              aria-label="预览 ${file.filename}"
              @click=${(e: Event) => {
                e.stopPropagation();
                onPreview(toPreviewRequest(file));
              }}
            >
              ${icons.eye}
            </button>`
          : nothing
      }
      <a
        class="chat-file-icon-btn"
        title="下载"
        aria-label="下载 ${file.filename}"
        href=${file.url}
        download=${file.filename}
        target="_blank"
        rel="noopener"
        @click=${(e: Event) => e.stopPropagation()}
      >
        ${icons.download}
      </a>
    </div>
  `;
}

function renderPreviewableFileCard(file: FileBlock, onPreview?: (req: FilePreviewRequest) => void) {
  const mode = filePreviewMode(file);
  if (!mode) {
    return null;
  }
  const openPreview = () => {
    if (onPreview) {
      onPreview(toPreviewRequest(file));
    }
  };
  return html`
    <div class="chat-file-preview-wrap">
      <div
        class="chat-file-preview-surface"
        role="button"
        tabindex="0"
        title="点击预览 ${file.filename}"
        @click=${openPreview}
        @keydown=${(e: KeyboardEvent) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            openPreview();
          }
        }}
      >
        ${renderFileEmbed(file, mode)}
      </div>
      <div class="chat-file-preview-bar">
        <div class="chat-file-preview-bar__info">
          <span class="chat-file-preview-bar__name">${file.filename}</span>
          ${
            file.sizeBytes
              ? html`<span class="chat-file-preview-bar__size muted">${formatFileSize(file.sizeBytes)}</span>`
              : nothing
          }
        </div>
        ${renderFileIconButtons(file, onPreview)}
      </div>
    </div>
  `;
}

function renderCompactFileRow(file: FileBlock, onPreview?: (req: FilePreviewRequest) => void) {
  return html`
    <div class="chat-file-card chat-file-card--compact">
      ${renderFileIconCompact(file.filename, file.mimeType)}
      <div class="chat-file-card__meta">
        <div class="chat-file-card__name">${file.filename}</div>
        <div class="chat-file-card__sub muted">
          ${fileTypeLabel(file.filename, file.mimeType)}${file.sizeBytes ? ` · ${formatFileSize(file.sizeBytes)}` : ""}
        </div>
      </div>
      ${renderFileIconButtons(file, onPreview)}
    </div>
  `;
}

export function renderFileAttachments(
  files: FileBlock[],
  onPreview?: (req: FilePreviewRequest) => void,
) {
  if (files.length === 0) {
    return nothing;
  }
  return html`
    <div class="chat-message-files">
      ${files.map((file) => renderCompactFileRow(file, onPreview))}
    </div>
  `;
}

import { stripThinkingTags } from "../format.ts";
import * as v0_9 from "@a2ui/web_core/v0_9";
import { basicCatalog } from "@a2ui/lit/v0_9";
import type { LitComponentApi } from "@a2ui/lit/v0_9";
import type { GatewayBrowserClient } from "../gateway.ts";
import { canonicalGatewaySessionKey } from "../sessions/session-key-utils.ts";

type Processor = v0_9.MessageProcessor<LitComponentApi>;

/** Canonical basic catalog id — must match @a2ui/lit basicCatalog.id */
export const BASIC_CATALOG_ID = basicCatalog.id;

function normalizeCatalogId(catalogId: unknown): string {
  if (typeof catalogId !== "string" || catalogId.trim() === "") {
    return BASIC_CATALOG_ID;
  }
  switch (catalogId.trim()) {
    case "basic":
    case "basic_catalog":
    case "basicCatalog":
    case "standard":
    case "default":
      return BASIC_CATALOG_ID;
    default:
      return catalogId;
  }
}

function cloneMessage(raw: unknown): v0_9.A2uiMessage {
  if (raw == null || typeof raw !== "object") {
    return raw as v0_9.A2uiMessage;
  }
  const msg = structuredClone(raw) as Record<string, unknown>;
  const createSurface = msg.createSurface;
  if (createSurface != null && typeof createSurface === "object") {
    (createSurface as Record<string, unknown>).catalogId = normalizeCatalogId(
      (createSurface as Record<string, unknown>).catalogId,
    );
  }
  const beginRendering = msg.beginRendering;
  if (beginRendering != null && typeof beginRendering === "object") {
    (beginRendering as Record<string, unknown>).catalogId = normalizeCatalogId(
      (beginRendering as Record<string, unknown>).catalogId,
    );
  }
  return msg as v0_9.A2uiMessage;
}

/** Normalize gateway/agent shorthand before handing messages to @a2ui/web_core. */
export function normalizeA2UIMessages(messages: unknown[]): v0_9.A2uiMessage[] {
  return messages.map(cloneMessage);
}

export function createChatA2UIProcessor(onAction: v0_9.ActionListener): Processor {
  return new v0_9.MessageProcessor([basicCatalog], onAction);
}

export function processA2UIMessages(processor: Processor, messages: unknown[]): void {
  if (messages.length === 0) {
    return;
  }
  processor.processMessages(normalizeA2UIMessages(messages));
}

export function createFreshChatA2UIProcessor(onAction: v0_9.ActionListener): Processor {
  return createChatA2UIProcessor(onAction);
}

/** @deprecated Streaming reset is handled by clearing panel messages. */
export function resetChatA2UISurfaces(): void {
  // No-op: each chat-a2ui-panel owns its processor instance.
}

export async function dispatchChatA2UIAction(
  client: GatewayBrowserClient | null,
  sessionKey: string,
  action: v0_9.A2uiClientAction,
): Promise<void> {
  if (!client) {
    return;
  }
  const userAction = {
    name: action.name,
    surfaceId: action.surfaceId,
    sourceComponentId: action.sourceComponentId,
    timestamp: new Date().toISOString(),
    context: { ...action.context },
  };
  await client.request("chat.a2ui.action", {
    sessionKey: canonicalGatewaySessionKey(sessionKey),
    userAction,
  });
}

/** Surface id referenced by a single A2UI server message block. */
export function a2uiMessageSurfaceId(block: unknown): string | null {
  if (block == null || typeof block !== "object") {
    return null;
  }
  const record = block as Record<string, unknown>;
  for (const key of [
    "createSurface",
    "updateComponents",
    "updateDataModel",
    "deleteSurface",
    "beginRendering",
    "surfaceUpdate",
    "dataModelUpdate",
  ] as const) {
    const payload = record[key];
    if (payload == null || typeof payload !== "object") {
      continue;
    }
    const surfaceId = (payload as Record<string, unknown>).surfaceId;
    if (typeof surfaceId === "string" && surfaceId.trim() !== "") {
      return surfaceId.trim();
    }
  }
  return null;
}

/** Drop persisted A2UI blocks for a surface after the user submits a button action. */
export function removeA2UISurfaceFromMessages(messages: unknown[], surfaceId: string): unknown[] {
  const sid = surfaceId.trim();
  if (!sid) {
    return messages;
  }
  const out: unknown[] = [];
  for (const msg of messages) {
    if (msg == null || typeof msg !== "object") {
      out.push(msg);
      continue;
    }
    const record = msg as Record<string, unknown>;
    const content = record.content;
    if (!Array.isArray(content)) {
      out.push(msg);
      continue;
    }
    const nextContent = content.filter((part) => {
      if (part == null || typeof part !== "object") {
        return true;
      }
      const block = part as Record<string, unknown>;
      if (block.type !== "a2ui") {
        return true;
      }
      return a2uiMessageSurfaceId(block.a2ui) !== sid;
    });
    if (nextContent.length === 0) {
      continue;
    }
    if (nextContent.length === content.length) {
      out.push(msg);
      continue;
    }
    out.push({ ...record, content: nextContent });
  }
  return out;
}

export function filterA2UIMessagesForSurface(messages: unknown[], surfaceId: string): unknown[] {
  const sid = surfaceId.trim();
  if (!sid) {
    return messages;
  }
  return messages.filter((block) => a2uiMessageSurfaceId(block) !== sid);
}

export function extractA2UIBlocks(message: unknown): unknown[] {
  const m = message as Record<string, unknown>;
  const content = m.content;
  if (!Array.isArray(content)) {
    return [];
  }
  const blocks: unknown[] = [];
  for (const part of content) {
    const item = part as Record<string, unknown>;
    if (item.type === "a2ui" && item.a2ui != null) {
      blocks.push(item.a2ui);
    }
  }
  return dedupeA2UIMessages(blocks);
}

/** Plain-text fallback from persisted A2UI updateDataModel payloads (e.g. path /content). */
export function extractA2UIDisplayTextFromBlocks(blocks: unknown[]): string | null {
  let latest: string | null = null;
  for (const raw of blocks) {
    if (raw == null || typeof raw !== "object") {
      continue;
    }
    const udm = (raw as Record<string, unknown>).updateDataModel;
    if (udm == null || typeof udm !== "object") {
      continue;
    }
    const value = (udm as Record<string, unknown>).value;
    if (typeof value === "string") {
      const trimmed = value.trim();
      if (trimmed) {
        latest = trimmed;
      }
    }
  }
  return latest;
}

const PERSISTED_OUTPUT_RE = /<persisted-output>[\s\S]*?<\/persisted-output>/gi;

/** Remove large tool stdout embedded in assistant A2UI payloads. */
export function stripPersistedOutputBlocks(text: string): string {
  return text.replace(PERSISTED_OUTPUT_RE, "").trim();
}

/** True when A2UI text is shell/tool output rather than a user-facing reply. */
export function isToolLikeDisplayText(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed) {
    return false;
  }
  if (PERSISTED_OUTPUT_RE.test(trimmed)) {
    PERSISTED_OUTPUT_RE.lastIndex = 0;
    const without = stripPersistedOutputBlocks(trimmed);
    if (!without) {
      return true;
    }
    return isToolLikeDisplayText(without);
  }
  const lower = trimmed.toLowerCase();
  const prefixes = [
    "command exited with non-zero code",
    "[stderr]:",
    "[command failed with exit code",
    "output too large (",
    "full output saved to:",
    "launching skill:",
    "launch chromium:",
  ];
  if (prefixes.some((p) => lower.startsWith(p))) {
    return true;
  }
  if (trimmed.startsWith('{"matches":')) {
    return true;
  }
  return false;
}

/** User-visible A2UI markdown after stripping tool noise. */
export function sanitizeA2UIDisplayText(text: string | null | undefined): string | null {
  if (text == null) {
    return null;
  }
  const cleaned = stripThinkingTags(stripPersistedOutputBlocks(text.trim()));
  if (!cleaned || isToolLikeDisplayText(cleaned)) {
    return null;
  }
  return cleaned;
}

export function extractA2UIDisplayText(message: unknown): string | null {
  return sanitizeA2UIDisplayText(extractA2UIDisplayTextFromBlocks(extractA2UIBlocks(message)));
}

/** Raw A2UI payload text (includes tool output) for tool trace panels. */
export function extractRawA2UIDisplayText(message: unknown): string | null {
  return extractA2UIDisplayTextFromBlocks(extractA2UIBlocks(message));
}

const TEXT_ONLY_A2UI_COMPONENTS = new Set(["Text", "Column", "Row"]);

/** True when A2UI blocks only render static text (no buttons, inputs, etc.). */
export function isTextOnlyA2UIDisplay(blocks: unknown[]): boolean {
  if (!sanitizeA2UIDisplayText(extractA2UIDisplayTextFromBlocks(blocks))) {
    return false;
  }
  const normalized = dedupeA2UIMessages(blocks);
  for (const block of normalized) {
    if (block == null || typeof block !== "object") {
      continue;
    }
    const updateComponents = (block as Record<string, unknown>).updateComponents;
    if (updateComponents == null || typeof updateComponents !== "object") {
      continue;
    }
    const components = (updateComponents as Record<string, unknown>).components;
    if (!Array.isArray(components)) {
      continue;
    }
    for (const comp of components) {
      if (comp == null || typeof comp !== "object") {
        continue;
      }
      const name = (comp as Record<string, unknown>).component;
      if (typeof name === "string" && !TEXT_ONLY_A2UI_COMPONENTS.has(name)) {
        return false;
      }
    }
  }
  return true;
}

function normalizeDataModelPath(path: unknown): string {
  if (typeof path !== "string" || path.trim() === "") {
    return "/";
  }
  return path.trim();
}

function a2uiUpdateDataModelKey(block: Record<string, unknown>): string | null {
  const udm = block.updateDataModel;
  if (udm == null || typeof udm !== "object") {
    return null;
  }
  const payload = udm as Record<string, unknown>;
  const sid = a2uiMessageSurfaceId(block) ?? "";
  return `${sid}\0${normalizeDataModelPath(payload.path)}`;
}

/** Coalesce streaming A2UI messages: keep latest updateDataModel per surface/path. */
export function dedupeA2UIMessages(messages: unknown[]): unknown[] {
  if (messages.length <= 1) {
    return messages;
  }
  const out: unknown[] = [];
  const updateDataModelIdx = new Map<string, number>();
  const updateComponentsIdx = new Map<string, number>();
  const createSurfaceSeen = new Set<string>();

  for (const raw of messages) {
    if (raw == null || typeof raw !== "object") {
      continue;
    }
    const block = raw as Record<string, unknown>;
    const fingerprint = JSON.stringify(block);
    const last = out[out.length - 1];
    if (last != null && JSON.stringify(last) === fingerprint) {
      continue;
    }

    const udmKey = a2uiUpdateDataModelKey(block);
    if (udmKey != null) {
      const idx = updateDataModelIdx.get(udmKey);
      if (idx !== undefined) {
        out[idx] = raw;
        continue;
      }
      updateDataModelIdx.set(udmKey, out.length);
      out.push(raw);
      continue;
    }

    if (block.updateComponents != null && typeof block.updateComponents === "object") {
      const sid = a2uiMessageSurfaceId(block) ?? "";
      const idx = updateComponentsIdx.get(sid);
      if (idx !== undefined) {
        out[idx] = raw;
        continue;
      }
      updateComponentsIdx.set(sid, out.length);
      out.push(raw);
      continue;
    }

    if (block.createSurface != null && typeof block.createSurface === "object") {
      const sid = a2uiMessageSurfaceId(block) ?? "";
      if (sid && createSurfaceSeen.has(sid)) {
        continue;
      }
      if (sid) {
        createSurfaceSeen.add(sid);
      }
      out.push(raw);
      continue;
    }

    out.push(raw);
  }
  return out;
}

/** Merge A2UI blocks into assistant message content without duplicate server messages. */
export function mergeA2UIIntoMessageContent(content: unknown[], blocks: unknown[]): unknown[] {
  if (blocks.length === 0) {
    return content;
  }
  const nonA2UI = content.filter((part) => {
    if (part == null || typeof part !== "object") {
      return true;
    }
    return (part as Record<string, unknown>).type !== "a2ui";
  });
  const existing = content
    .filter((part) => part != null && typeof part === "object" && (part as Record<string, unknown>).type === "a2ui")
    .map((part) => (part as Record<string, unknown>).a2ui);
  const merged = dedupeA2UIMessages([...existing, ...blocks]);
  return [...nonA2UI, ...merged.map((a2ui) => ({ type: "a2ui", a2ui }))];
}

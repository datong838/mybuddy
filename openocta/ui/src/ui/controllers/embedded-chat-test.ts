import { gatewayHttpBase } from "../gateway-url.ts";
import { splitReasoningFromText } from "../shared/text/reasoning-tags.ts";
import type { EmbeddedModelEntry } from "./embedded-models.ts";

export type PlazaChatMessage = {
  role: "user" | "assistant";
  content: string;
  modelId?: string;
  thinking?: string | null;
};

export type ChatCompletionResponse = {
  choices?: Array<{
    message?: { role?: string; content?: string };
    finish_reason?: string;
  }>;
  error?: { message?: string };
  message?: string;
  ok?: boolean;
};

function authHeaders(token: string): Record<string, string> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    "Content-Type": "application/json",
  };
  const tok = (token ?? "").trim();
  if (tok) {
    headers.Authorization = `Bearer ${tok}`;
    headers["X-Gateway-Token"] = tok;
  }
  return headers;
}

function apiBase(gatewayHost: string): string | null {
  const base = gatewayHttpBase(gatewayHost.trim());
  if (!base) {
    return null;
  }
  return base.replace(/\/$/, "");
}

/** Gateway proxy URL shown in the test chat modal (avoids browser CORS to model port). */
export function resolveEmbeddedChatProxyURL(gatewayHost: string): string | null {
  const base = apiBase(gatewayHost);
  if (!base) {
    return null;
  }
  return `${base}/api/embedded-models/chat/completions`;
}

/** Legacy direct runtime URL (server-side only; browser cannot call cross-origin). */
export function resolveEmbeddedChatURL(model: EmbeddedModelEntry): string | null {
  const endpoint = (model.endpoint ?? "").trim();
  if (endpoint) {
    return `${endpoint.replace(/\/+$/, "")}/chat/completions`;
  }
  if (model.port && model.port > 0) {
    return `http://127.0.0.1:${model.port}/v1/chat/completions`;
  }
  return null;
}

export async function postEmbeddedChatCompletion(opts: {
  gatewayHost: string;
  token: string;
  model: EmbeddedModelEntry;
  messages: PlazaChatMessage[];
  maxTokens?: number;
}): Promise<{ ok: boolean; content?: string; thinking?: string | null; error?: string }> {
  const url = resolveEmbeddedChatProxyURL(opts.gatewayHost);
  if (!url) {
    return { ok: false, error: "未配置网关地址（Gateway URL）" };
  }
  if (!opts.model.running) {
    return { ok: false, error: "模型未运行或缺少 endpoint" };
  }

  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers: authHeaders(opts.token),
      body: JSON.stringify({
        model: opts.model.id,
        messages: opts.messages.map((m) => ({ role: m.role, content: m.content })),
        max_tokens: opts.maxTokens ?? 512,
        stream: false,
        thinking: true,
      }),
    });
  } catch (err) {
    return { ok: false, error: err instanceof Error ? err.message : "网络请求失败" };
  }

  let data: ChatCompletionResponse = {};
  try {
    data = (await res.json()) as ChatCompletionResponse;
  } catch {
    return { ok: false, error: `响应解析失败 (HTTP ${res.status})` };
  }

  if (!res.ok) {
    return { ok: false, error: data.message ?? data.error?.message ?? `HTTP ${res.status}` };
  }

  const raw = data.choices?.[0]?.message?.content ?? "";
  const { thinking, text } = splitReasoningFromText(raw);
  if (!text && !thinking) {
    return { ok: false, error: "模型返回空内容" };
  }
  return { ok: true, content: text || "（无可见回复）", thinking };
}

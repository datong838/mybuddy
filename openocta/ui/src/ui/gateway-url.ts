/**
 * Gateway URL utilities.
 * Storage stores gatewayHost as "IP:Port" (e.g. "127.0.0.1:18900").
 * Use these helpers to build WebSocket and HTTP URLs.
 */

/**
 * Parse input to host:port format.
 * Accepts: "127.0.0.1:18900", "ws://127.0.0.1:18900", "wss://100.x.y.z:18900"
 */
export function parseGatewayHost(input: string): string {
  const v = (input ?? "").trim();
  if (!v) return "";
  const m = v.match(/^(?:wss?:\/\/)?([^/]+?)(?:\/|$)/);
  if (m) return m[1];
  return v;
}

/**
 * Build WebSocket URL from host:port.
 * Uses ws:// or wss:// based on current page protocol.
 */
export function gatewayWebSocketUrl(host: string): string {
  const h = parseGatewayHost(host);
  if (!h) return "";
  const proto = typeof location !== "undefined" && location.protocol === "https:" ? "wss" : "ws";
  return `${proto}://${h}`;
}

/**
 * Build HTTP base URL from host:port.
 * Uses http:// or https:// based on current page protocol.
 */
export function gatewayHttpBase(host: string): string {
  const h = parseGatewayHost(host);
  if (!h) return "";
  const proto = typeof location !== "undefined" && location.protocol === "https:" ? "https" : "http";
  return `${proto}://${h}`;
}

/** True when the UI talks to a gateway on this machine (desktop shell or local service). */
export function isLocalGatewayHost(host: string): boolean {
  const h = parseGatewayHost(host);
  if (!h) return false;
  const hostname = h.split(":")[0]?.trim().toLowerCase() ?? "";
  if (!hostname) return false;
  if (hostname === "127.0.0.1" || hostname === "localhost" || hostname === "::1") {
    return true;
  }
  if (typeof location !== "undefined") {
    return hostname === location.hostname.toLowerCase();
  }
  return false;
}

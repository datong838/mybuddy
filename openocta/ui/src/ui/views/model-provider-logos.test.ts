import { describe, expect, it } from "vitest";
import { BUILTIN_PROVIDERS } from "./models-builtin.ts";
import { resolveModelProviderLogo } from "./model-provider-logos.ts";

describe("resolveModelProviderLogo", () => {
  it("resolves logos for builtin providers that use non-obvious ids", () => {
    const xai = BUILTIN_PROVIDERS.find((provider) => provider.id === "xai");
    const zai = BUILTIN_PROVIDERS.find((provider) => provider.id === "zai");
    const vercel = BUILTIN_PROVIDERS.find((provider) => provider.id === "vercel-ai-gateway");

    expect(resolveModelProviderLogo("xai", xai?.label, xai?.baseUrl, xai)).toBeTruthy();
    expect(resolveModelProviderLogo("zai", zai?.label, zai?.baseUrl, zai)).toBeTruthy();
    expect(resolveModelProviderLogo("vercel-ai-gateway", vercel?.label, vercel?.baseUrl, vercel)).toBeTruthy();
  });

  it("reuses the same kimi logo across moonshot variants", () => {
    const moonshot = BUILTIN_PROVIDERS.find((provider) => provider.id === "moonshot");
    const moonshotCn = BUILTIN_PROVIDERS.find((provider) => provider.id === "moonshot-cn");
    const kimiCoding = BUILTIN_PROVIDERS.find((provider) => provider.id === "kimi-coding");

    expect(resolveModelProviderLogo("moonshot", moonshot?.label, moonshot?.baseUrl, moonshot)).toBe(
      resolveModelProviderLogo("moonshot-cn", moonshotCn?.label, moonshotCn?.baseUrl, moonshotCn),
    );
    expect(resolveModelProviderLogo("moonshot", moonshot?.label, moonshot?.baseUrl, moonshot)).toBe(
      resolveModelProviderLogo("kimi-coding", kimiCoding?.label, kimiCoding?.baseUrl, kimiCoding),
    );
  });

  it("resolves custom providers from stable public hostnames", () => {
    expect(resolveModelProviderLogo("custom-openrouter", "My Router", "https://openrouter.ai/api/v1")).toBeTruthy();
    expect(resolveModelProviderLogo("custom-deepseek", "My DeepSeek", "https://api.deepseek.com/v1")).toBeTruthy();
  });

  it("does not infer by IP address unless the provider name matches a known local engine", () => {
    expect(resolveModelProviderLogo("custom-provider", "Unknown Provider", "http://192.168.1.10:8000/v1")).toBeNull();
    expect(resolveModelProviderLogo("ollama-local", "Local Ollama", "http://127.0.0.1:11434/v1")).toBeTruthy();
    expect(resolveModelProviderLogo("vllm-node", "vLLM Node", "http://[::1]:8000/v1")).toBeTruthy();
  });

  it("matches normalized display names with Chinese aliases", () => {
    expect(resolveModelProviderLogo("custom-bailian", "百炼 (阿里云)", "http://10.0.0.8:9000/v1")).toBeTruthy();
    expect(resolveModelProviderLogo("custom-zhipu", "Z.ai (智谱)", "https://gateway.example.com/v1")).toBeTruthy();
  });
});

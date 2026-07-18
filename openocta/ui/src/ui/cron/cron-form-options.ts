type ModelEntry = { id: string; name?: string };

export function buildCronModelOptions(
  configValue: unknown,
): Array<{ value: string; label: string }> {
  const cfg = configValue as
    | {
        agents?: { defaults?: { model?: { primary?: string }; models?: Record<string, { alias?: string }> } };
        models?: { providers?: Record<string, { models?: ModelEntry[] }> };
      }
    | null;
  const defaultRef =
    cfg?.agents?.defaults?.model && typeof cfg.agents.defaults.model === "object"
      ? String((cfg.agents.defaults.model as { primary?: string }).primary ?? "").trim()
      : typeof cfg?.agents?.defaults?.model === "string"
        ? String(cfg.agents.defaults.model).trim()
        : "";
  const seen = new Set<string>();
  const opts: Array<{ value: string; label: string }> = [{ value: "", label: "不指定" }];

  const agentModels = cfg?.agents?.defaults?.models;
  if (agentModels && typeof agentModels === "object") {
    for (const [id, raw] of Object.entries(agentModels)) {
      const value = id.trim();
      if (!value || seen.has(value)) continue;
      seen.add(value);
      const alias =
        raw && typeof raw === "object" && "alias" in raw && typeof raw.alias === "string"
          ? raw.alias.trim()
          : "";
      const label = alias && alias !== value ? `${alias} (${value})` : value;
      opts.push({ value, label });
    }
  }

  const providers = cfg?.models?.providers;
  if (providers && typeof providers === "object") {
    for (const [providerKey, prov] of Object.entries(providers)) {
      const models =
        prov && typeof prov === "object" ? (prov as { models?: ModelEntry[] }).models : undefined;
      if (!Array.isArray(models)) continue;
      for (const m of models) {
        const modelId = m?.id?.trim();
        if (!modelId) continue;
        const value = `${providerKey}/${modelId}`;
        if (seen.has(value)) continue;
        seen.add(value);
        const label = m.name && m.name !== modelId ? `${m.name} (${value})` : value;
        opts.push({ value, label });
      }
    }
  }

  if (defaultRef && !seen.has(defaultRef)) {
    opts.push({ value: defaultRef, label: `默认 (${defaultRef})` });
  }

  return opts;
}

export function buildCronMcpOptions(
  configValue: unknown,
): Array<{ key: string; label: string }> {
  const cfg = configValue as { mcp?: { servers?: Record<string, { enabled?: boolean }> } } | null;
  const servers = cfg?.mcp?.servers ?? {};
  return Object.keys(servers)
    .filter((key) => servers[key]?.enabled !== false)
    .map((key) => ({ key, label: key }));
}

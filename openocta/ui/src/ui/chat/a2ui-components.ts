import type { ComponentRecord } from "./a2ui-components.ts";

/** Accept array, id-keyed map, {"item":[...]} wrapper, or single component object. */
export function componentsArrayFromRaw(raw: unknown): ComponentRecord[] {
  if (raw == null) {
    return [];
  }
  if (Array.isArray(raw)) {
    return raw.map((item) => (item ?? {}) as ComponentRecord);
  }
  if (typeof raw !== "object") {
    return [];
  }
  const obj = raw as ComponentRecord;
  const itemWrapped = obj.item;
  if (Array.isArray(itemWrapped)) {
    return itemWrapped.map((item) => (item ?? {}) as ComponentRecord);
  }
  if ("id" in obj || "component" in obj) {
    return [obj];
  }
  const out: ComponentRecord[] = [];
  for (const [id, value] of Object.entries(obj)) {
    if (value == null || typeof value !== "object" || Array.isArray(value)) {
      continue;
    }
    const comp = { ...(value as ComponentRecord) };
    if (comp.id == null) {
      comp.id = id;
    }
    out.push(comp);
  }
  return out;
}

/** Flatten nested {"Text": {...}} component maps from older payloads. */
export function normalizeComponentMap(comp: ComponentRecord): ComponentRecord {
  const out: ComponentRecord = { ...comp };
  const rawComponent = comp.component;
  if (rawComponent != null && typeof rawComponent === "object" && !Array.isArray(rawComponent)) {
    const nested = rawComponent as ComponentRecord;
    const typeName = Object.keys(nested)[0];
    if (typeName) {
      out.component = typeName;
      const props = nested[typeName];
      if (props != null && typeof props === "object" && !Array.isArray(props)) {
        for (const [key, value] of Object.entries(props as ComponentRecord)) {
          if (!(key in out)) {
            out[key] = value;
          }
        }
      }
    }
  }
  if (typeof out.component !== "string") {
    out.component = "Text";
  }
  return out;
}

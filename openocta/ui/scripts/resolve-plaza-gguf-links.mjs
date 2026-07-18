#!/usr/bin/env node
/**
 * Resolve GGUF HuggingFace repo URLs for plaza catalog models.
 * Updates hfUrl + downloadable for models whose CanIRun url points at base weights.
 *
 * Usage: node ui/scripts/resolve-plaza-gguf-links.mjs [--dry-run]
 */
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_UI_PATH = join(__dirname, "../src/ui/data/plaza-catalog.json");
const OUT_GO_PATH = join(__dirname, "../../src/pkg/embeddedmodels/plaza-catalog.json");
const DRY_RUN = process.argv.includes("--dry-run");
const RATE_LIMIT_MS = 180;

/** Too large or only multi-shard GGUF — keep HF source URL, no embedded download. */
const SKIP_IDS = new Set(["deepseek-v3.2", "kimi-k2"]);

const HF_BASES = ["https://hf-mirror.com", "https://huggingface.co"];
const PREFERRED_ORGS = [
  "bartowski",
  "lmstudio-community",
  "QuantFactory",
  "ggml-org",
  "Qwen",
  "unsloth",
  "hugging-quants",
  "TheBloke",
];

const BLOCK_TOKENS = [
  "mmproj",
  "clip",
  "projector",
  "embed",
  "embedding",
  "rerank",
  "asr",
  "whisper",
  "vocoder",
  "lora",
  "adapter",
];

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function isGgufRepo(url) {
  return /huggingface\.co/i.test(url ?? "") && /gguf/i.test(url ?? "");
}

function extractRepo(url) {
  const m = (url ?? "").match(/huggingface\.co\/([^/]+\/[^/?#]+)/i);
  return m ? m[1] : null;
}

function repoUrl(repo) {
  return `https://huggingface.co/${repo}`;
}

function normalizeToken(s) {
  return (s ?? "").toLowerCase().replace(/[^a-z0-9]+/g, "");
}

async function hfFetch(path) {
  for (const base of HF_BASES) {
    try {
      const res = await fetch(`${base}${path}`, {
        headers: { Accept: "application/json", "User-Agent": "openocta-plaza-resolver/1.0" },
        signal: AbortSignal.timeout(20000),
      });
      if (!res.ok) {
        continue;
      }
      return res.json();
    } catch {
      // try next mirror
    }
  }
  return null;
}

function isMainGguf(path) {
  const lower = path.toLowerCase();
  if (!lower.endsWith(".gguf")) {
    return false;
  }
  return !BLOCK_TOKENS.some((t) => lower.includes(t));
}

async function repoHasMainGguf(repo) {
  const tree = await hfFetch(`/api/models/${repo}/tree/main`);
  if (!Array.isArray(tree)) {
    return false;
  }
  return tree.some((e) => e.type === "file" && isMainGguf(e.path));
}

function candidateRepos(model) {
  const out = [];
  const seen = new Set();
  const add = (repo) => {
    const key = repo.trim();
    if (!key || seen.has(key)) {
      return;
    }
    seen.add(key);
    out.push(key);
  };

  const baseRepo = extractRepo(model.hfUrl);
  if (baseRepo) {
    const [org, repoName] = baseRepo.split("/");
    add(`${org}/${repoName}-GGUF`);
    add(`bartowski/${repoName}-GGUF`);
    add(`bartowski/${repoName}-Instruct-GGUF`);
    if (/-Instruct/i.test(repoName)) {
      add(`bartowski/${repoName}-GGUF`);
    }
    add(`lmstudio-community/${repoName}-GGUF`);
    add(`QuantFactory/${repoName}-GGUF`);
  }

  const compactName = (model.name ?? "").replace(/\s+/g, "");
  if (compactName) {
    add(`lmstudio-community/${compactName}-GGUF`);
    add(`bartowski/${compactName}-GGUF`);
    add(`bartowski/${compactName}-Instruct-GGUF`);
  }

  const idCompact = (model.id ?? "").replace(/-/g, "");
  if (idCompact) {
    add(`bartowski/${idCompact}-GGUF`);
  }

  return out;
}

function scoreSearchRepo(repoId, model) {
  const lower = repoId.toLowerCase();
  let score = 0;
  if (!lower.includes("gguf")) {
    return -999;
  }
  if (BLOCK_TOKENS.some((t) => lower.includes(t))) {
    return -999;
  }
  const org = repoId.split("/")[0];
  const prefIdx = PREFERRED_ORGS.indexOf(org);
  if (prefIdx >= 0) {
    score += 40 - prefIdx;
  }
  const idNorm = normalizeToken(model.id);
  const nameNorm = normalizeToken(model.name);
  const repoNorm = normalizeToken(repoId.split("/")[1] ?? "");
  if (idNorm && repoNorm.includes(idNorm.replace(/gguf/g, ""))) {
    score += 25;
  }
  if (nameNorm && repoNorm.includes(nameNorm.slice(0, Math.min(12, nameNorm.length)))) {
    score += 15;
  }
  if (/instruct|chat|it/i.test(repoId) && model.kind === "chat") {
    score += 5;
  }
  if (/embedding/i.test(repoId) && model.kind === "embedding") {
    score += 10;
  }
  if (/abliterated|nsfw|rp-|roleplay|tiny-orchestrator|neo-si/i.test(lower)) {
    score -= 20;
  }
  return score;
}

async function searchGgufRepos(model) {
  const queries = [
    `${model.name} GGUF`,
    `${model.id.replace(/-/g, " ")} GGUF`,
    extractRepo(model.hfUrl)?.split("/")[1]
      ? `${extractRepo(model.hfUrl).split("/")[1]} GGUF`
      : null,
  ].filter(Boolean);

  const scored = new Map();
  for (const q of queries) {
    const data = await hfFetch(`/api/models?search=${encodeURIComponent(q)}&limit=12`);
    if (!Array.isArray(data)) {
      continue;
    }
    for (const item of data) {
      const id = item?.id ?? item?.modelId;
      if (typeof id !== "string") {
        continue;
      }
      const score = scoreSearchRepo(id, model);
      if (score <= 0) {
        continue;
      }
      scored.set(id, Math.max(scored.get(id) ?? 0, score));
    }
    await sleep(80);
  }
  return [...scored.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([id]) => id);
}

async function resolveGgufRepo(model) {
  if (isGgufRepo(model.hfUrl)) {
    return { repo: extractRepo(model.hfUrl), source: "existing" };
  }

  for (const repo of candidateRepos(model)) {
    if (await repoHasMainGguf(repo)) {
      return { repo, source: "candidate" };
    }
    await sleep(60);
  }

  const searched = await searchGgufRepos(model);
  for (const repo of searched.slice(0, 8)) {
    if (await repoHasMainGguf(repo)) {
      return { repo, source: "search" };
    }
    await sleep(60);
  }

  return null;
}

async function main() {
  const catalog = JSON.parse(readFileSync(OUT_UI_PATH, "utf8"));
  const pending = catalog.filter((m) => !isGgufRepo(m.hfUrl) && !SKIP_IDS.has(m.id));
  console.log(`Resolving GGUF repos for ${pending.length}/${catalog.length} models…`);

  const resolved = [];
  const failed = [];

  for (let i = 0; i < pending.length; i++) {
    const model = pending[i];
    process.stdout.write(`[${i + 1}/${pending.length}] ${model.id} … `);
    const hit = await resolveGgufRepo(model);
    if (hit?.repo) {
      const oldRepo = extractRepo(model.hfUrl);
      model.hfUrl = repoUrl(hit.repo);
      model.downloadable = true;
      resolved.push({ id: model.id, repo: hit.repo, via: hit.source, old: oldRepo });
      console.log(`OK ${hit.repo} (${hit.source})`);
    } else {
      failed.push(model.id);
      console.log("SKIP");
    }
    await sleep(RATE_LIMIT_MS);
  }

  console.log(`\nResolved: ${resolved.length}, still missing: ${failed.length}`);
  if (failed.length) {
    console.log("Missing:", failed.join(", "));
  }

  if (DRY_RUN) {
    console.log("\nDry run — no files written.");
    console.log(JSON.stringify(resolved.slice(0, 10), null, 2));
    return;
  }

  const payload = `${JSON.stringify(catalog, null, 2)}\n`;
  writeFileSync(OUT_UI_PATH, payload, "utf8");
  writeFileSync(OUT_GO_PATH, payload, "utf8");
  console.log(`Updated ${OUT_UI_PATH} and ${OUT_GO_PATH}`);
  console.log(`Downloadable count: ${catalog.filter((m) => m.downloadable).length}/${catalog.length}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});

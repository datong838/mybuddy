const fs = require("fs");
const path = require("path");
const roots = ["/app/targets/next/web/.next", "/app/targets/vinext/dist"];
const repls = [
  ["WEB APPS", "网页应用"],
  ["未找到 snippets", "未找到提示片段"],
  ["搜索 snippets...", "搜索提示片段..."],
  ["探索 Marketplace", "探索应用市场"],
  ['"Snippets"', '"提示片段"'],
  ["Snippets", "提示片段"],
  ['"Marketplace"', '"应用市场"'],
  ["Marketplace", "应用市场"],
  ["Chatflow", "对话流"],
  ["CHATFLOW", "对话流"],
  ['label:"Agents"', 'label:"智能体"'],
  ["label:'Agents'", "label:'智能体'"],
  ['"Agents"', '"智能体"'],
];
function walk(dir, out = []) {
  let entries;
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const e of entries) {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) walk(p, out);
    else out.push(p);
  }
  return out;
}
let count = 0;
for (const root of roots) {
  if (!fs.existsSync(root)) continue;
  for (const p of walk(root)) {
    const base = path.basename(p);
    const ok =
      p.includes("zh-Hans") ||
      base.endsWith(".js") ||
      base.endsWith(".mjs") ||
      base.endsWith(".json");
    if (!ok) continue;
    let text;
    try {
      text = fs.readFileSync(p, "utf8");
    } catch {
      continue;
    }
    let next = text;
    for (const [a, b] of repls) {
      if (next.includes(a)) next = next.split(a).join(b);
    }
    if (next !== text) {
      fs.writeFileSync(p, next, "utf8");
      count += 1;
      console.log(p);
    }
  }
}
console.log("patched_files=" + count);

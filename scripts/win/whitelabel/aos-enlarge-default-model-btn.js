const fs = require("fs");
const path = require("path");
const roots = [
  "/app/targets/next/web/.next",
  "/app/targets/vinext/dist",
];
const needle = "i-ri-brain-2-line";
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
    else if (/\.(js|mjs)$/.test(e.name)) out.push(p);
  }
  return out;
}
let count = 0;
for (const root of roots) {
  if (!fs.existsSync(root)) continue;
  for (const p of walk(root)) {
    let text;
    try {
      text = fs.readFileSync(p, "utf8");
    } catch {
      continue;
    }
    if (!text.includes(needle)) continue;
    let next = text;
    // Enlarge default-model settings button (unique icon)
    next = next.split("i-ri-brain-2-line size-3.5").join("i-ri-brain-2-line size-4");
    next = next.split("i-ri-loader-2-line size-3.5").join("i-ri-loader-2-line size-4");
    // Common minified class combinations for this button
    next = next.replace(
      /i-ri-brain-2-line([^"']{0,40})size-3\.5/g,
      "i-ri-brain-2-line$1size-4",
    );
    // If button size prop is adjacent in same object literal near brain icon, bump small->medium carefully:
    // only within a short window around the icon
    const idx = next.indexOf(needle);
    if (idx >= 0) {
      const start = Math.max(0, idx - 400);
      const end = Math.min(next.length, idx + 400);
      const window = next.slice(start, end);
      const patchedWindow = window
        .replace(/size:"small"/g, 'size:"medium"')
        .replace(/size:'small'/g, "size:'medium'")
        .replace(/mr-0\.5/g, "mr-1");
      if (patchedWindow !== window) {
        next = next.slice(0, start) + patchedWindow + next.slice(end);
      }
    }
    // Ensure readable text size on trigger
    if (!next.includes("text-[14px]") && next.includes(needle)) {
      next = next.replace(
        'className:e()("relative"',
        'className:e()("relative text-[14px] leading-5"',
      );
      next = next.replace(
        'className:(0,q.cn)("relative"',
        'className:(0,q.cn)("relative text-[14px] leading-5"',
      );
      next = next.replace(
        'cn("relative"',
        'cn("relative text-[14px] leading-5"',
      );
    }
    if (next !== text) {
      fs.writeFileSync(p, next, "utf8");
      count += 1;
      console.log(p);
    }
  }
}
console.log("patched_files=" + count);

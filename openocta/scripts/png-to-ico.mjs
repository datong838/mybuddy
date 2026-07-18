#!/usr/bin/env node
/**
 * 将 PNG 转为单尺寸 ICO（Vista+ 内嵌 PNG），供 Wails appicon / NSIS MUI 使用。
 * 最长边不得超过 256（Windows ICO 目录项为单字节宽高）。
 * 用法: node scripts/png-to-ico.mjs <输入.png> <输出.ico>
 */
import fs from "node:fs";
import path from "node:path";

function readU32BE(buf, off) {
  return buf.readUInt32BE(off);
}

function pngDimensions(buf) {
  if (buf.length < 24 || buf.toString("ascii", 1, 4) !== "PNG") {
    throw new Error("不是有效的 PNG 文件");
  }
  const w = readU32BE(buf, 16);
  const h = readU32BE(buf, 20);
  if (!w || !h || w > 256 || h > 256) {
    throw new Error(`PNG 宽、高均须 ≤256（当前 ${w}×${h}）。请先缩放，例如: sips -Z 256 imgs/openocta_logo.png --out imgs/openocta_logo_wails.png`);
  }
  return { w, h };
}

function pngToIco(pngBuf) {
  const { w, h } = pngDimensions(pngBuf);
  const iconDir = Buffer.alloc(6);
  iconDir.writeUInt16LE(0, 0);
  iconDir.writeUInt16LE(1, 2);
  iconDir.writeUInt16LE(1, 4);

  const entry = Buffer.alloc(16);
  entry.writeUInt8(w === 256 ? 0 : w, 0);
  entry.writeUInt8(h === 256 ? 0 : h, 1);
  entry.writeUInt8(0, 2);
  entry.writeUInt8(0, 3);
  entry.writeUInt16LE(1, 4);
  entry.writeUInt16LE(32, 6);
  entry.writeUInt32LE(pngBuf.length, 8);
  entry.writeUInt32LE(6 + 16, 12);

  return Buffer.concat([iconDir, entry, pngBuf]);
}

const [, , inPath, outPath] = process.argv;
if (!inPath || !outPath) {
  console.error("用法: node scripts/png-to-ico.mjs <输入.png> <输出.ico>");
  process.exit(1);
}

const png = fs.readFileSync(inPath);
const ico = pngToIco(png);
fs.mkdirSync(path.dirname(path.resolve(outPath)), { recursive: true });
fs.writeFileSync(outPath, ico);

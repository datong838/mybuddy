import { describe, expect, it } from "vitest";
import { decodeFileText, extractFileBlocks, extractFileBlocksFromA2UIBlocks } from "./file-blocks.ts";
import { parseCsvTable, resolveTextFilePreviewKind } from "./file-preview-content.ts";

describe("extractFileBlocksFromA2UIBlocks", () => {
  it("parses @@OPENOCTA_ATTACHMENTS@@ from A2UI Text components", () => {
    const blocks = [
      {
        version: "v0.9",
        createSurface: { surfaceId: "main", catalogId: "basic" },
      },
      {
        version: "v0.9",
        updateComponents: {
          surfaceId: "main",
          components: [
            {
              id: "root",
              component: "Text",
              text: '已生成报告\n@@OPENOCTA_ATTACHMENTS@@\n[{"type":"file","filename":"report.html","mimeType":"text/html","data":"PGgxPm9rPC9oMT4="}]',
            },
          ],
        },
      },
    ];
    const files = extractFileBlocksFromA2UIBlocks(blocks);
    expect(files).toHaveLength(1);
    expect(files[0]?.filename).toBe("report.html");
    expect(files[0]?.mimeType).toBe("text/html");
    expect(files[0]?.isPreviewable).toBe(true);
  });

  it("extracts Image component URLs as previewable files", () => {
    const blocks = [
      {
        version: "v0.9",
        createSurface: { surfaceId: "main", catalogId: "basic" },
      },
      {
        version: "v0.9",
        updateComponents: {
          surfaceId: "main",
          components: [
            {
              id: "img",
              component: "Image",
              url: "data:image/png;base64,abcd",
              description: "chart",
            },
          ],
        },
      },
    ];
    const files = extractFileBlocksFromA2UIBlocks(blocks);
    expect(files).toHaveLength(1);
    expect(files[0]?.mimeType).toBe("image/png");
    expect(files[0]?.isPreviewable).toBe(true);
  });

  it("includes a2ui blocks when extracting from assistant messages", () => {
    const message = {
      role: "assistant",
      content: [
        {
          type: "a2ui",
          a2ui: {
            version: "v0.9",
            updateComponents: {
              surfaceId: "main",
              components: [
                {
                  id: "root",
                  component: "Text",
                  text: '@@OPENOCTA_ATTACHMENTS@@\n[{"type":"file","filename":"notes.txt","mimeType":"text/plain","data":"aGk="}]',
                },
              ],
            },
          },
        },
      ],
    };
    const files = extractFileBlocks(message);
    expect(files).toHaveLength(1);
    expect(files[0]?.filename).toBe("notes.txt");
  });
});

describe("decodeFileText", () => {
  it("decodes UTF-8 CSV with CJK characters from base64 data URLs", () => {
    const csv = "姓名,年龄\n张伟,28\n";
    const bytes = new TextEncoder().encode(csv);
    let binary = "";
    for (const byte of bytes) {
      binary += String.fromCharCode(byte);
    }
    const base64 = btoa(binary);
    const url = `data:text/plain; charset=utf-8;base64,${base64}`;
    expect(decodeFileText(url)).toBe(csv);
  });
});

describe("file preview content", () => {
  it("resolves preview kinds by extension", () => {
    expect(resolveTextFilePreviewKind("data.csv", "text/plain")).toBe("csv");
    expect(resolveTextFilePreviewKind("readme.md", "text/plain")).toBe("markdown");
    expect(resolveTextFilePreviewKind("config.json", "application/json")).toBe("json");
    expect(resolveTextFilePreviewKind("notes.txt", "text/plain")).toBe("text");
  });

  it("parses CSV with quoted commas", () => {
    const table = parseCsvTable('name,note\nAlice,"hello, world"\nBob,ok');
    expect(table.headers).toEqual(["name", "note"]);
    expect(table.rows).toEqual([
      ["Alice", "hello, world"],
      ["Bob", "ok"],
    ]);
  });
});

import { describe, expect, it } from "vitest";
import {
  extractReferencedAttachmentPaths,
  extractReferencedPathsFromGroup,
} from "./file-blocks.ts";

describe("extractReferencedAttachmentPaths", () => {
  it("matches basename paths in backticks", () => {
    expect(extractReferencedAttachmentPaths("文件 `sample_data.csv` 的内容")).toEqual([
      "sample_data.csv",
    ]);
  });

  it("matches absolute paths after Chinese colon", () => {
    const path = "/tmp/workspace/sample_data.csv";
    expect(
      extractReferencedAttachmentPaths(`把这个文件：${path} 返回给我`),
    ).toEqual([path]);
  });

  it("matches absolute paths in backticks inside A2UI markdown", () => {
    const path = "/Users/zhanbei/.openocta/workspace/sample_data.csv";
    const text = `这是文件 \`${path}\` 的内容`;
    expect(extractReferencedAttachmentPaths(text)).toEqual([path]);
  });
});

describe("extractReferencedPathsFromGroup", () => {
  it("extracts paths from A2UI updateDataModel in assistant messages", () => {
    const path = "/Users/zhanbei/.openocta/workspace/sample_data.csv";
    const group = [
      {
        message: {
          role: "user",
          content: [{ type: "text", text: `把这个文件：${path} 返回给我` }],
        },
      },
      {
        message: {
          role: "assistant",
          content: [
            {
              type: "a2ui",
              a2ui: {
                updateDataModel: {
                  path: "/content",
                  value: `这是文件 \`${path}\` 的内容`,
                },
              },
            },
          ],
        },
      },
    ];
    expect(extractReferencedPathsFromGroup(group)).toEqual([path]);
  });
});

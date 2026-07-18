package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeLocalResource(t *testing.T) {
	cases := map[string]bool{
		"attachments/report.html": true,
		"file:///tmp/report.html": true,
		"./out/report.html":       true,
		"https://example.com/a":   false,
	}
	for input, want := range cases {
		if got := looksLikeLocalResource(input); got != want {
			t.Fatalf("%q: got %v want %v", input, got, want)
		}
	}
}

func TestAttachmentBlocksFromWriteFileToolOutput(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "report.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := "Updated file report.csv"
	blocks := AttachmentBlocksFromWriteToolOutput("write_file", output, dir)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if fn, _ := blocks[0]["filename"].(string); fn != "report.csv" {
		t.Fatalf("filename: got %q want report.csv", fn)
	}
}

func TestAttachmentBlocksFromReferencedPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attachments", "report.html")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := "报告已保存到 `attachments/report.html`，请点击 [预览](attachments/report.html)"
	blocks := AttachmentBlocksFromReferencedPaths(text, dir)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestAttachmentBlocksFromReferencedMarkdownPath(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := "已将 README.md 返回给前端展示预览。"
	blocks := AttachmentBlocksFromReferencedPaths(text, dir)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if fn, _ := blocks[0]["filename"].(string); fn != "README.md" {
		t.Fatalf("filename: got %q want README.md", fn)
	}
}

func TestAttachmentBlocksFromReferencedAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "sample_data.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	text := "把这个文件：" + csvPath + " 返回给我"
	blocks := AttachmentBlocksFromReferencedPaths(text, dir)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if fn, _ := blocks[0]["filename"].(string); fn != "sample_data.csv" {
		t.Fatalf("filename: got %q want sample_data.csv", fn)
	}
}

func TestMergeDeliverableAttachmentBlocksFromA2UI(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "sample_data.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := []map[string]interface{}{
		{
			"type": "a2ui",
			"a2ui": map[string]interface{}{
				"updateDataModel": map[string]interface{}{
					"path":  "/content",
					"value": "文件 `" + csvPath + "` 已返回",
				},
			},
		},
	}
	merged := MergeDeliverableAttachmentBlocks(content, dir)
	if len(merged) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(merged))
	}
	if fn, _ := merged[1]["filename"].(string); fn != "sample_data.csv" {
		t.Fatalf("filename: got %q want sample_data.csv", fn)
	}
}

func TestAttachmentBlocksFromImagePathsInGlobOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photos", "cat.jpg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{0xff, 0xd8, 0xff, 0xe0}, 0o644); err != nil {
		t.Fatal(err)
	}
	output := "photos/cat.jpg\nphotos/dog.png\n"
	blocks := AttachmentBlocksFromDeliverableToolOutput("glob", output, dir)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 image block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "image" {
		t.Fatalf("expected image block, got %#v", blocks[0]["type"])
	}
}

func TestWebFetchLocalHTMLAttachesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attachments", "report.html")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	html := "<html><body>ok</body></html>"
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	output := "Local file: attachments/report.html\n"
	blocks := AttachmentBlocksFromDeliverableToolOutput("web_fetch", output, dir)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 attachment block, got %d", len(blocks))
	}
	if blocks[0]["type"] != "file" {
		t.Fatalf("expected file block, got %#v", blocks[0]["type"])
	}
}

package localagents

import "testing"

func TestParseMessageSingle(t *testing.T) {
	segs := ParseMessage("@cursor 实现登录功能")
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if segs[0].AgentID != "cursor" || segs[0].Task != "实现登录功能" {
		t.Fatalf("unexpected segment: %+v", segs[0])
	}
}

func TestParseMessageMultiple(t *testing.T) {
	segs := ParseMessage("@cursor 实现登录 @opencode 写产品说明")
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].AgentID != "cursor" || segs[1].AgentID != "opencode" {
		t.Fatalf("unexpected ids: %+v", segs)
	}
	if segs[1].Task != "写产品说明" {
		t.Fatalf("unexpected task2: %q", segs[1].Task)
	}
}

func TestParseMessageColon(t *testing.T) {
	segs := ParseMessage("@codex：生成测试用例")
	if len(segs) != 1 || segs[0].Task != "生成测试用例" {
		t.Fatalf("unexpected: %+v", segs)
	}
}

func TestParseMessageUnknownAgent(t *testing.T) {
	if segs := ParseMessage("@unknown do something"); segs != nil {
		t.Fatalf("expected nil for unknown agent, got %+v", segs)
	}
}

func TestParseMessageNoMention(t *testing.T) {
	if segs := ParseMessage("hello world"); segs != nil {
		t.Fatalf("expected nil, got %+v", segs)
	}
}

func TestValidateSegments(t *testing.T) {
	installed := map[string]AgentProbeResult{
		"cursor": {ID: "cursor", Installed: true},
	}
	missing, ok := ValidateSegments([]TaskSegment{{AgentID: "cursor", Task: "x"}}, installed)
	if !ok || len(missing) != 0 {
		t.Fatalf("expected ok, missing=%v", missing)
	}
	missing, ok = ValidateSegments([]TaskSegment{{AgentID: "trae", Task: "x"}}, installed)
	if ok || len(missing) != 1 {
		t.Fatalf("expected missing trae, ok=%v missing=%v", ok, missing)
	}
}

func TestFormatCombinedOutput(t *testing.T) {
	out := FormatCombinedOutput([]SegmentResult{
		{Label: "Cursor", Output: "done"},
		{Label: "OpenCode", Error: "failed"},
	})
	if out == "" {
		t.Fatal("empty output")
	}
}

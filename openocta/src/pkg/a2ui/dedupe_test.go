package a2ui

import "testing"

func TestCoalesceAssistantA2UIBlocksKeepsLatestUpdateDataModel(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "a2ui", "a2ui": map[string]interface{}{
			"updateDataModel": map[string]interface{}{
				"surfaceId": "main",
				"path":      "/content",
				"value":     "line1",
			},
		}},
		{"type": "a2ui", "a2ui": map[string]interface{}{
			"updateDataModel": map[string]interface{}{
				"surfaceId": "main",
				"path":      "/content",
				"value":     "line1\n\n- a\n- b",
			},
		}},
	}
	out := CoalesceAssistantA2UIBlocks(blocks)
	if len(out) != 1 {
		t.Fatalf("expected 1 block, got %d", len(out))
	}
	a2ui, _ := out[0]["a2ui"].(map[string]interface{})
	udm, _ := a2ui["updateDataModel"].(map[string]interface{})
	if udm["value"] != "line1\n\n- a\n- b" {
		t.Fatalf("unexpected value %#v", udm["value"])
	}
}

func TestAppendAssistantA2UIBlockReplacesSamePath(t *testing.T) {
	blocks := AppendAssistantA2UIBlock(nil, map[string]interface{}{
		"updateDataModel": map[string]interface{}{
			"surfaceId": "main",
			"path":      "/content",
			"value":     "a",
		},
	})
	blocks = AppendAssistantA2UIBlock(blocks, map[string]interface{}{
		"updateDataModel": map[string]interface{}{
			"surfaceId": "main",
			"path":      "/content",
			"value":     "b",
		},
	})
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

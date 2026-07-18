package a2ui

import "strings"

func normalizeDataModelPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	return path
}

func surfaceIDFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	sid, _ := payload["surfaceId"].(string)
	return strings.TrimSpace(sid)
}

// CoalesceAssistantA2UIBlocks keeps the latest updateDataModel per surface/path,
// the latest updateComponents per surface, and a single createSurface per surface.
func CoalesceAssistantA2UIBlocks(blocks []map[string]interface{}) []map[string]interface{} {
	if len(blocks) <= 1 {
		return blocks
	}
	out := make([]map[string]interface{}, 0, len(blocks))
	updateDataModelIdx := map[string]int{}
	updateComponentsIdx := map[string]int{}
	createSurfaceSeen := map[string]struct{}{}

	for _, block := range blocks {
		typ, _ := block["type"].(string)
		if !strings.EqualFold(strings.TrimSpace(typ), "a2ui") {
			out = append(out, block)
			continue
		}
		parsed, _ := block["a2ui"].(map[string]interface{})
		if parsed == nil {
			out = append(out, block)
			continue
		}

		if udm, ok := parsed["updateDataModel"].(map[string]interface{}); ok {
			key := surfaceIDFromPayload(udm) + "\x00" + normalizeDataModelPath(asString(udm["path"]))
			if idx, exists := updateDataModelIdx[key]; exists {
				out[idx] = block
				continue
			}
			updateDataModelIdx[key] = len(out)
			out = append(out, block)
			continue
		}

		if uc, ok := parsed["updateComponents"].(map[string]interface{}); ok {
			key := surfaceIDFromPayload(uc)
			if idx, exists := updateComponentsIdx[key]; exists {
				out[idx] = block
				continue
			}
			updateComponentsIdx[key] = len(out)
			out = append(out, block)
			continue
		}

		if cs, ok := parsed["createSurface"].(map[string]interface{}); ok {
			key := surfaceIDFromPayload(cs)
			if key != "" {
				if _, exists := createSurfaceSeen[key]; exists {
					continue
				}
				createSurfaceSeen[key] = struct{}{}
			}
			out = append(out, block)
			continue
		}

		out = append(out, block)
	}
	return out
}

func AppendAssistantA2UIBlock(blocks []map[string]interface{}, parsed map[string]interface{}) []map[string]interface{} {
	block := map[string]interface{}{"type": "a2ui", "a2ui": parsed}
	if udm, ok := parsed["updateDataModel"].(map[string]interface{}); ok {
		key := surfaceIDFromPayload(udm) + "\x00" + normalizeDataModelPath(asString(udm["path"]))
		for i := len(blocks) - 1; i >= 0; i-- {
			typ, _ := blocks[i]["type"].(string)
			if !strings.EqualFold(strings.TrimSpace(typ), "a2ui") {
				continue
			}
			inner, _ := blocks[i]["a2ui"].(map[string]interface{})
			if inner == nil {
				continue
			}
			prev, ok := inner["updateDataModel"].(map[string]interface{})
			if !ok {
				continue
			}
			prevKey := surfaceIDFromPayload(prev) + "\x00" + normalizeDataModelPath(asString(prev["path"]))
			if prevKey == key {
				blocks[i] = block
				return blocks
			}
		}
	}
	return append(blocks, block)
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/openocta/openocta/pkg/agent/knowledge"
)

type memorySearchInput struct {
	Query string `json:"query" jsonschema:"description=Search query for vault notes"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=Max results (default 8)"`
}

func KnowledgeTools(opts *KnowledgeOptions) []einotool.BaseTool {
	if opts == nil || !opts.Enabled {
		return nil
	}
	indexDir := opts.IndexDir
	vaultDir := opts.VaultDir
	tool, err := utils.InferTool("memory_search", "Search Obsidian-compatible vault notes indexed for this agent.", func(ctx context.Context, in memorySearchInput) (string, error) {
		q := strings.TrimSpace(in.Query)
		if q == "" {
			return "", fmt.Errorf("query is required")
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 8
		}
		eng, release, err := knowledge.AcquireEngine(indexDir, nil)
		if err != nil {
			return "", err
		}
		defer release()
		kinds := map[string]struct{}{knowledge.KindDocument: {}}
		hits, err := eng.SearchIndex(ctx, q, kinds, limit)
		if err != nil {
			return "", err
		}
		if len(hits) == 0 {
			return fmt.Sprintf("No vault hits for %q in %s", q, vaultDir), nil
		}
		b, _ := json.MarshalIndent(hits, "", "  ")
		return string(b), nil
	})
	if err != nil {
		return nil
	}
	return []einotool.BaseTool{tool}
}

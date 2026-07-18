// Package handlers provides a unified file read API for the Gateway.
// files.read allows reading file contents under allowed roots (state dir, workspace, bundled skills).
package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/paths"
)

// FilesReadParams holds parameters for files.read.
type FilesReadParams struct {
	Path string
}

func parseFilesReadParams(params map[string]interface{}) (*FilesReadParams, error) {
	p := &FilesReadParams{}
	path, ok := params["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("path is required")
	}
	p.Path = strings.TrimSpace(path)
	return p, nil
}

// allowedReadRoots returns absolute directory paths under which file reads are allowed
// (state dir and default agent workspace). Covers managed skills and workspace files.
func allowedReadRoots(cfg *config.OpenOctaConfig, env func(string) string) []string {
	var roots []string
	stateDir := paths.ResolveStateDir(env)
	if stateDir != "" {
		abs, _ := filepath.Abs(stateDir)
		if abs != "" {
			roots = append(roots, abs)
		}
	}
	workspaceDir := resolveAgentWorkspaceDir(cfg, resolveDefaultAgentID(cfg), env)
	if workspaceDir != "" {
		abs, _ := filepath.Abs(workspaceDir)
		if abs != "" {
			roots = append(roots, abs)
		}
	}
	return roots
}

// isPathUnderRoot returns true if cleanAbs is under root (or equal).
func isPathUnderRoot(cleanAbs, root string) bool {
	rel, err := filepath.Rel(root, cleanAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// FilesReadHandler handles "files.read". Reads file content for paths under allowed roots
// (state dir, agent workspace, managed skills, bundled skills). For use by UI and other features.
func FilesReadHandler(opts HandlerOpts) error {
	params, err := parseFilesReadParams(opts.Params)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: fmt.Sprintf("invalid files.read params: %v", err),
		}, nil)
		return nil
	}

	env := func(k string) string { return os.Getenv(k) }
	var cfg *config.OpenOctaConfig
	if opts.Context != nil && opts.Context.Config != nil {
		cfg = opts.Context.Config
	} else {
		loaded, loadErr := config.Load(env)
		if loadErr != nil {
			opts.Respond(false, nil, &protocol.ErrorShape{
				Code:    protocol.ErrCodeInternal,
				Message: "failed to load config: " + loadErr.Error(),
			}, nil)
			return nil
		}
		cfg = loaded
	}

	abs, err := filepath.Abs(params.Path)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "invalid path: " + err.Error(),
		}, nil)
		return nil
	}
	clean := filepath.Clean(abs)

	roots := allowedReadRoots(cfg, env)
	allowed := false
	for _, root := range roots {
		if root == "" {
			continue
		}
		if isPathUnderRoot(clean, root) {
			allowed = true
			break
		}
	}
	if !allowed {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "path is not under an allowed root",
		}, nil)
		return nil
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		if os.IsNotExist(err) {
			opts.Respond(false, nil, &protocol.ErrorShape{
				Code:    protocol.ErrCodeInvalidRequest,
				Message: "file not found",
			}, nil)
			return nil
		}
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "failed to read file: " + err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"content": string(data),
	}, nil, nil)
	return nil
}

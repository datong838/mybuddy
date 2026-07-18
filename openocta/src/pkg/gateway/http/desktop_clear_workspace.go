package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/openocta/openocta/pkg/paths"
)

type clearWorkspaceResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// handleDesktopClearWorkspaceOptions handles CORS preflight for POST /api/desktop/clear-workspace.
func (s *Server) handleDesktopClearWorkspaceOptions(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// handleDesktopClearWorkspace removes all files and subdirectories under the default agent workspace
// directory: <stateDir>/workspace (e.g. ~/.openocta/workspace, or %APPDATA%\openocta\workspace on Windows).
// Requires gateway token. Allowed when OPENOCTA_RUN_MODE=desktop or OPENOCTA_ALLOW_UNINSTALL=1.
func (s *Server) handleDesktopClearWorkspace(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{OK: false, Message: "仅支持 POST"})
		return
	}

	env := func(k string) string { return os.Getenv(k) }
	stateDir := filepath.Clean(paths.ResolveStateDir(env))
	workspaceDir := filepath.Join(stateDir, "workspace")
	workspaceDir = filepath.Clean(workspaceDir)

	if stateDir == "" || stateDir == "." || stateDir == "/" {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{OK: false, Message: "无法解析状态目录"})
		return
	}

	st, err := os.Stat(workspaceDir)
	if err != nil {
		if os.IsNotExist(err) {
			_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{
				OK:      true,
				Message: "默认工作区目录不存在，无需清理。",
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{OK: false, Message: "无法访问工作区目录", Detail: err.Error()})
		return
	}
	if !st.IsDir() {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{OK: false, Message: "工作区路径不是目录", Detail: workspaceDir})
		return
	}

	if err := removeDirContents(workspaceDir); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{OK: false, Message: "清理失败", Detail: err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(clearWorkspaceResponse{
		OK:      true,
		Message: "已清空默认工作区内的所有文件与文件夹（" + workspaceDir + "）。",
	})
}

// removeDirContents deletes everything inside dir; dir itself remains.
func removeDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

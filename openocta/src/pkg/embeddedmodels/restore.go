package embeddedmodels

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/openocta/openocta/pkg/logging"
)

type restoreCandidate struct {
	modelID string
	port    int
}

// RestoreRuntimeOnStartup reloads embedded models marked as running in manifest.json.
func RestoreRuntimeOnStartup(env func(string) string) {
	if env == nil {
		env = os.Getenv
	}
	RefreshSideloadCatalog(env)
	candidates := findModelsToRestore(env)
	if len(candidates) == 0 {
		return
	}

	for _, item := range candidates {
		modelID := item.modelID
		preferredPort := item.port

		entry, found := ResolveCatalogEntry(modelID)
		if !found {
			_ = setRuntimeState(env, modelID, false, preferredPort, "未知模型，无法自动恢复")
			continue
		}
		if !IsInstalled(env, modelID) {
			_ = setRuntimeState(env, modelID, false, preferredPort, "模型文件缺失，无法自动恢复")
			continue
		}
		if err := normalizeCatalogFiles(env, entry); err != nil {
			logging.Warn("embedded models: normalize files before restore model=%s err=%v", modelID, err)
		}

		logging.Info("embedded models: restoring runtime model=%s port=%d", modelID, preferredPort)
		port, err := startModel(env, modelID, preferredPort)
		if err != nil {
			_ = setRuntimeState(env, modelID, false, preferredPort, fmt.Sprintf("自动恢复失败: %v", err))
			logging.Warn("embedded models: restore failed model=%s err=%v", modelID, err)
			continue
		}
		logging.Info("embedded models: restored runtime model=%s port=%d endpoint=http://127.0.0.1:%d/v1", modelID, port, port)
	}

	if err := PersistMergedProviderConfig(env, 0); err != nil {
		logging.Warn("embedded models: update merged provider config err=%v", err)
	}
}

func findModelsToRestore(env func(string) string) []restoreCandidate {
	manifestMu.Lock()
	m, err := loadManifest(env)
	manifestMu.Unlock()
	if err != nil {
		logging.Warn("embedded models: load manifest for restore err=%v", err)
		return nil
	}

	out := make([]restoreCandidate, 0)
	for id, st := range m.Models {
		if !st.Running {
			continue
		}
		out = append(out, restoreCandidate{modelID: id, port: st.Port})
	}
	return out
}

func normalizeCatalogFiles(env func(string) string, entry CatalogEntry) error {
	dir := ModelDir(env, entry.ID)
	for _, f := range entry.Files {
		dest := filepath.Join(dir, f.Name)
		if err := normalizeDownloadDest(dest); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
	}
	return nil
}

// PersistProviderConfig writes embedded model provider settings into openocta.json.
func PersistProviderConfig(env func(string) string, port int, modelID string) error {
	_ = port
	_ = modelID
	return PersistMergedProviderConfig(env, 0)
}

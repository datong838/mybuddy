package embeddedmodels

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

var (
	sideloadMu   sync.RWMutex
	sideloadByID map[string]CatalogEntry
)

func validSideloadModelID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || id == manifestName {
		return false
	}
	for i, r := range id {
		if i == 0 {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return false
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func listGGUFFileNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".gguf") {
			out = append(out, name)
		}
	}
	return out
}

func splitPrimaryAndMMProj(names []string) (primary string, mmprojs []string) {
	for _, name := range names {
		if isMMProj(name) {
			mmprojs = append(mmprojs, name)
			continue
		}
		if primary == "" {
			primary = name
			continue
		}
		// Prefer the largest non-mmproj file name as primary when multiple weights exist.
		if len(name) > len(primary) {
			primary = name
		}
	}
	return primary, mmprojs
}

func inferSideloadKind(modelID, primaryGGUF string) ModelKind {
	combined := strings.ToLower(modelID + " " + primaryGGUF)
	if strings.Contains(combined, "embed") {
		return ModelKindEmbedding
	}
	return ModelKindChat
}

func sideloadDisplayName(modelID, primaryGGUF string) string {
	if primaryGGUF != "" {
		name := strings.TrimSuffix(primaryGGUF, filepath.Ext(primaryGGUF))
		if name != "" {
			return name
		}
	}
	return modelID
}

func fileSizeInDir(dir, name string) int64 {
	path, err := resolveCatalogFilePath(dir, name)
	if err != nil {
		return 0
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return 0
	}
	return info.Size()
}

func scanSideloadedModels(env func(string) string) map[string]CatalogEntry {
	root := ResolveModelsDir(env)
	entries, err := os.ReadDir(root)
	if err != nil {
		return map[string]CatalogEntry{}
	}
	out := make(map[string]CatalogEntry)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		id := e.Name()
		if !validSideloadModelID(id) {
			continue
		}
		if _, inCatalog := FindCatalogEntry(id); inCatalog {
			continue
		}
		dir := filepath.Join(root, id)
		ggufs := listGGUFFileNames(dir)
		if len(ggufs) == 0 {
			continue
		}
		primary, mmprojs := splitPrimaryAndMMProj(ggufs)
		if primary == "" {
			continue
		}
		files := []ModelFile{{Name: primary, Size: fileSizeInDir(dir, primary)}}
		for _, mm := range mmprojs {
			files = append(files, ModelFile{Name: mm, Size: fileSizeInDir(dir, mm)})
		}
		kind := inferSideloadKind(id, primary)
		out[id] = CatalogEntry{
			ID:            id,
			Kind:          kind,
			Name:          sideloadDisplayName(id, primary),
			Description:   "手动导入的本地 GGUF 模型。将权重放入 ~/.openocta/embedded-models/" + id + "/ 后点击「刷新」即可识别。",
			Tags:          []string{"手动导入", "本地"},
			Sideloaded:    true,
			Downloadable:  false,
			Multimodal:    len(mmprojs) > 0,
			Files:         files,
			ContextLength: defaultContextSize,
		}
	}
	return out
}

// RefreshSideloadCatalog rescans ~/.openocta/embedded-models for user-placed GGUF directories.
func RefreshSideloadCatalog(env func(string) string) {
	if env == nil {
		env = os.Getenv
	}
	scanned := scanSideloadedModels(env)
	sideloadMu.Lock()
	sideloadByID = scanned
	sideloadMu.Unlock()

	for id, entry := range scanned {
		if !entryFilesOnDisk(env, entry) {
			continue
		}
		names := make([]string, 0, len(entry.Files))
		for _, f := range entry.Files {
			if f.Name != "" {
				names = append(names, f.Name)
			}
		}
		_ = upsertInstalled(env, id, names)
	}
}

func sideloadSnapshot() map[string]CatalogEntry {
	sideloadMu.RLock()
	defer sideloadMu.RUnlock()
	if len(sideloadByID) == 0 {
		return nil
	}
	out := make(map[string]CatalogEntry, len(sideloadByID))
	for id, entry := range sideloadByID {
		out[id] = entry
	}
	return out
}

func entryFilesOnDisk(env func(string) string, entry CatalogEntry) bool {
	dir := ModelDir(env, entry.ID)
	files := effectiveModelFiles(env, entry)
	if len(files) == 0 {
		return false
	}
	for _, f := range files {
		if _, err := resolveCatalogFilePath(dir, f.Name); err != nil {
			return false
		}
	}
	return true
}

// ResolveCatalogEntry returns a plaza catalog entry or a sideloaded scan entry by id.
// Call RefreshSideloadCatalog before relying on sideloaded entries.
func ResolveCatalogEntry(id string) (CatalogEntry, bool) {
	if entry, ok := FindCatalogEntry(id); ok {
		return entry, true
	}
	sideloadMu.RLock()
	entry, ok := sideloadByID[id]
	sideloadMu.RUnlock()
	return entry, ok
}

package embeddedmodels

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InstalledState tracks download/runtime state for one model.
type InstalledState struct {
	ID          string    `json:"id"`
	InstalledAt time.Time `json:"installedAt"`
	Files       []string  `json:"files"`
	Running     bool      `json:"running"`
	Port        int       `json:"port,omitempty"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	LastError   string    `json:"lastError,omitempty"`
}

// Manifest persists installed embedded models under ~/.openocta/embedded-models/.
type Manifest struct {
	Models map[string]InstalledState `json:"models"`
}

var manifestMu sync.Mutex

func loadManifest(env func(string) string) (*Manifest, error) {
	path := ResolveManifestPath(env)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Models: map[string]InstalledState{}}, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Models == nil {
		m.Models = map[string]InstalledState{}
	}
	return &m, nil
}

func saveManifest(env func(string) string, m *Manifest) error {
	dir := ResolveModelsDir(env)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, ".manifest.tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, ResolveManifestPath(env))
}

// ListInstalled returns all installed models merged with catalog metadata.
func ListInstalled(env func(string) string) ([]map[string]interface{}, error) {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	m, err := loadManifest(env)
	if err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, 0, len(m.Models))
	for id, st := range m.Models {
		entry, ok := FindCatalogEntry(id)
		item := map[string]interface{}{
			"id":          id,
			"installed":   true,
			"running":     st.Running,
			"port":        st.Port,
			"files":       st.Files,
			"installedAt": st.InstalledAt,
			"lastError":   st.LastError,
		}
		if ok {
			item["name"] = entry.Name
			item["kind"] = entry.Kind
			item["kindLabel"] = KindLabel(entry.Kind)
			item["description"] = entry.Description
			item["tags"] = entry.Tags
			item["builtin"] = entry.Builtin
			item["multimodal"] = entry.Multimodal
		}
		out = append(out, item)
	}
	return out, nil
}

// effectiveModelFiles returns catalog files, or filenames recorded in manifest after download.
func effectiveModelFiles(env func(string) string, entry CatalogEntry) []ModelFile {
	if len(entry.Files) > 0 {
		return entry.Files
	}
	st, ok := getInstalledState(env, entry.ID)
	if !ok || len(st.Files) == 0 {
		return nil
	}
	out := make([]ModelFile, len(st.Files))
	for i, name := range st.Files {
		out[i] = ModelFile{Name: name}
	}
	return out
}

// IsInstalled checks whether all expected weight files exist on disk.
func IsInstalled(env func(string) string, id string) bool {
	entry, ok := ResolveCatalogEntry(id)
	if !ok {
		return false
	}
	files := effectiveModelFiles(env, entry)
	if len(files) == 0 {
		return false
	}
	dir := ModelDir(env, id)
	for _, f := range files {
		if _, err := resolveCatalogFilePath(dir, f.Name); err != nil {
			return false
		}
	}
	return true
}

func upsertInstalled(env func(string) string, id string, files []string) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	m, err := loadManifest(env)
	if err != nil {
		return err
	}
	st := m.Models[id]
	st.ID = id
	st.Files = files
	st.InstalledAt = time.Now().UTC()
	st.LastError = ""
	m.Models[id] = st
	return saveManifest(env, m)
}

func removeInstalled(env func(string) string, id string) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	m, err := loadManifest(env)
	if err != nil {
		return err
	}
	delete(m.Models, id)
	return saveManifest(env, m)
}

func setRuntimeState(env func(string) string, id string, running bool, port int, lastErr string) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	m, err := loadManifest(env)
	if err != nil {
		return err
	}
	st, ok := m.Models[id]
	if !ok {
		st = InstalledState{ID: id}
	}
	st.Running = running
	st.Port = port
	if running {
		st.StartedAt = time.Now().UTC()
		st.LastError = ""
	} else if lastErr != "" {
		st.LastError = lastErr
	}
	m.Models[id] = st
	return saveManifest(env, m)
}

// markOnlyRunning sets running=true for one model and clears running on all others.
func markOnlyRunning(env func(string) string, id string, port int) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	m, err := loadManifest(env)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for mid, st := range m.Models {
		if mid == id {
			st.Running = true
			st.Port = port
			st.StartedAt = now
			st.LastError = ""
		} else if st.Running {
			st.Running = false
		}
		m.Models[mid] = st
	}
	if st, ok := m.Models[id]; ok {
		st.Running = true
		st.Port = port
		st.StartedAt = now
		st.LastError = ""
		m.Models[id] = st
	} else {
		m.Models[id] = InstalledState{
			ID:          id,
			Running:     true,
			Port:        port,
			StartedAt:   now,
			InstalledAt: now,
		}
	}
	return saveManifest(env, m)
}

func getInstalledState(env func(string) string, id string) (InstalledState, bool) {
	manifestMu.Lock()
	defer manifestMu.Unlock()

	m, err := loadManifest(env)
	if err != nil {
		return InstalledState{}, false
	}
	st, ok := m.Models[id]
	return st, ok
}

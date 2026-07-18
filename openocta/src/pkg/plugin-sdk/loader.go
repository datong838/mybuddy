package pluginsdk

import (
	"os"
	"path/filepath"
	"plugin"
)

// Loader loads Go plugins from a directory.
type Loader struct {
	pluginDir string
}

// NewLoader creates a loader for the given plugin directory.
func NewLoader(pluginDir string) *Loader {
	return &Loader{pluginDir: pluginDir}
}

// List returns paths of .so files in the plugin directory.
func (l *Loader) List() ([]string, error) {
	entries, err := os.ReadDir(l.pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".so" {
			out = append(out, filepath.Join(l.pluginDir, e.Name()))
		}
	}
	return out, nil
}

// Open opens a plugin by path.
func (l *Loader) Open(path string) (*plugin.Plugin, error) {
	return plugin.Open(path)
}

// LookupSymbol looks up a symbol in a plugin.
func LookupSymbol(p *plugin.Plugin, name string) (interface{}, error) {
	return p.Lookup(name)
}

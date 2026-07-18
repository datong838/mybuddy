package embeddedmodels

import (
	"os"
	"path/filepath"

	"github.com/openocta/openocta/pkg/paths"
)

const (
	modelsSubdir = "embedded-models"
	libSubdir    = "yzma-lib"
	manifestName = "manifest.json"
)

// ResolveModelsDir returns ~/.openocta/embedded-models for GGUF weights.
func ResolveModelsDir(env func(string) string) string {
	if env == nil {
		env = os.Getenv
	}
	return filepath.Join(paths.ResolveStateDir(env), modelsSubdir)
}

// ResolveLibDir returns ~/.openocta/yzma-lib for llama.cpp prebuilt libraries.
func ResolveLibDir(env func(string) string) string {
	if env == nil {
		env = os.Getenv
	}
	return filepath.Join(paths.ResolveStateDir(env), libSubdir)
}

// ResolveManifestPath returns ~/.openocta/embedded-models/manifest.json.
func ResolveManifestPath(env func(string) string) string {
	return filepath.Join(ResolveModelsDir(env), manifestName)
}

// ModelDir returns the directory for a specific catalog model id.
func ModelDir(env func(string) string, modelID string) string {
	return filepath.Join(ResolveModelsDir(env), modelID)
}

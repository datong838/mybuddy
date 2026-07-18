package embeddedmodels

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolvePrimaryWeightPath(env func(string) string, entry CatalogEntry) (string, error) {
	files := effectiveModelFiles(env, entry)
	if len(files) == 0 {
		return "", fmt.Errorf("找不到模型权重文件")
	}
	dir := ModelDir(env, entry.ID)
	for _, f := range files {
		if f.Name == "" || isMMProj(f.Name) {
			continue
		}
		path, err := resolveCatalogFilePath(dir, f.Name)
		if err != nil {
			return "", fmt.Errorf("找不到模型权重文件: %w", err)
		}
		return path, nil
	}
	return "", fmt.Errorf("找不到模型权重文件")
}

// resolveCatalogFilePath locates a catalog file under the model directory.
// Handles legacy layouts where go-getter created a nested folder named after the file.
func resolveCatalogFilePath(modelDir, fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("empty file name")
	}
	direct := filepath.Join(modelDir, fileName)
	info, err := os.Stat(direct)
	if err == nil {
		if info.Mode().IsRegular() {
			return direct, nil
		}
		if info.IsDir() {
			if nested, ok := findGGUFFile(direct, fileName); ok {
				return nested, nil
			}
		}
	}
	return "", fmt.Errorf("missing model file: %s", fileName)
}

func findGGUFFile(dir, preferredName string) (string, bool) {
	if preferredName != "" {
		nested := filepath.Join(dir, preferredName)
		if info, err := os.Stat(nested); err == nil && info.Mode().IsRegular() {
			return nested, true
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
			return filepath.Join(dir, e.Name()), true
		}
	}
	return "", false
}

// normalizeDownloadDest flattens go-getter output when dest became a directory containing the file.
func normalizeDownloadDest(dest string) error {
	info, err := os.Stat(dest)
	if err != nil {
		return err
	}
	if info.Mode().IsRegular() {
		return nil
	}
	if !info.IsDir() {
		return nil
	}

	baseName := filepath.Base(dest)
	src, ok := findGGUFFile(dest, baseName)
	if !ok {
		return fmt.Errorf("download directory is empty: %s", dest)
	}

	tmp := dest + ".flattening"
	if err := os.Rename(src, tmp); err != nil {
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		_ = os.Rename(tmp, src)
		return err
	}
	return os.Rename(tmp, dest)
}

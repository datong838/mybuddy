// Package config: load .env from current working directory into os environment.
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFromFile reads path and sets environment variables for each KEY=VALUE line.
// Skips empty lines and lines starting with #. Does not override existing env vars.
// Returns nil if the file does not exist.
func LoadEnvFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

// LoadEnvFromCurrentDir loads .env from the current working directory.
// Called at startup so config load and the rest of the process see these vars.
func LoadEnvFromCurrentDir() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return LoadEnvFromFile(filepath.Join(dir, ".env"))
}

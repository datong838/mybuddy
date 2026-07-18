package appupdate

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/paths"
)

func envGetter() func(string) string {
	return os.Getenv
}

func configPath() string {
	env := envGetter()
	stateDir := paths.ResolveStateDir(env)
	return paths.ResolveConfigPath(env, stateDir)
}

func loadConfigMap() (map[string]interface{}, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]interface{}{}, nil
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root == nil {
		root = map[string]interface{}{}
	}
	return root, nil
}

func saveConfigMap(root map[string]interface{}) error {
	path := configPath()
	if err := os.MkdirAll(paths.ResolveStateDir(envGetter()), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func readUpdateSection(root map[string]interface{}) map[string]interface{} {
	updateRaw, ok := root["update"]
	if !ok || updateRaw == nil {
		return map[string]interface{}{}
	}
	update, ok := updateRaw.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return update
}

func skippedVersionsFromMap(update map[string]interface{}) []string {
	raw, ok := update["skippedVersions"]
	if !ok || raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, s := range v {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func lastCheckAtFromMap(update map[string]interface{}) (time.Time, bool) {
	raw, ok := update["lastCheckAt"]
	if !ok || raw == nil {
		return time.Time{}, false
	}
	s, ok := raw.(string)
	if !ok {
		return time.Time{}, false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func loadUpdateSettings() (skipped []string, lastCheckAt *time.Time, err error) {
	root, err := loadConfigMap()
	if err != nil {
		return nil, nil, err
	}
	update := readUpdateSection(root)
	skipped = skippedVersionsFromMap(update)
	if t, ok := lastCheckAtFromMap(update); ok {
		lastCheckAt = &t
	}
	return skipped, lastCheckAt, nil
}

func isVersionSkipped(version string, skipped []string) bool {
	version = strings.TrimSpace(version)
	for _, s := range skipped {
		if strings.EqualFold(strings.TrimSpace(s), version) {
			return true
		}
	}
	return false
}

func shouldRunDailyCheck(lastCheckAt *time.Time, now time.Time) bool {
	if lastCheckAt == nil {
		return true
	}
	return now.Sub(*lastCheckAt) >= 24*time.Hour
}

func recordLastCheckAt(now time.Time) error {
	root, err := loadConfigMap()
	if err != nil {
		return err
	}
	update := readUpdateSection(root)
	update["lastCheckAt"] = now.UTC().Format(time.RFC3339)
	root["update"] = update
	return saveConfigMap(root)
}

func appendSkippedVersion(version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil
	}
	root, err := loadConfigMap()
	if err != nil {
		return err
	}
	update := readUpdateSection(root)
	skipped := skippedVersionsFromMap(update)
	for _, s := range skipped {
		if strings.EqualFold(strings.TrimSpace(s), version) {
			return nil
		}
	}
	skipped = append(skipped, version)
	update["skippedVersions"] = skipped
	root["update"] = update
	return saveConfigMap(root)
}

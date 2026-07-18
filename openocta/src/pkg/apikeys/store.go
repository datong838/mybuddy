package apikeys

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type storeFile struct {
	Version int      `json:"version"`
	Keys    []Record `json:"keys"`
}

// LoadStore reads keys from disk.
func LoadStore(storePath string) (*storeFile, error) {
	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &storeFile{Version: 1, Keys: []Record{}}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return &storeFile{Version: 1, Keys: []Record{}}, nil
	}
	var store storeFile
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Keys == nil {
		store.Keys = []Record{}
	}
	if store.Version == 0 {
		store.Version = 1
	}
	return &store, nil
}

// SaveStore writes keys to disk.
func SaveStore(storePath string, store *storeFile) error {
	if err := os.MkdirAll(filepath.Dir(storePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storePath, data, 0644)
}

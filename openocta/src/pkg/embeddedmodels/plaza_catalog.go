package embeddedmodels

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed plaza-catalog.json
var plazaCatalogJSON []byte

type plazaCatalogItem struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Provider      string   `json:"provider"`
	ParamsB       float64  `json:"paramsB"`
	ActiveParamsB float64  `json:"activeParamsB"`
	ContextLength int      `json:"contextLength"`
	Capabilities  []string `json:"capabilities"`
	License       string   `json:"license"`
	ReleasedAt    string   `json:"releasedAt"`
	Architecture  string   `json:"architecture"`
	Quantization  string   `json:"quantization"`
	ToolCalling   bool     `json:"toolCalling"`
	Multimodal    bool     `json:"multimodal"`
	OllamaName    string   `json:"ollamaName"`
	HfURL         string   `json:"hfUrl"`
	Featured      bool     `json:"featured"`
	Builtin       bool     `json:"builtin"`
	Downloadable  bool     `json:"downloadable"`
}

var (
	plazaOnce    sync.Once
	plazaItems   []plazaCatalogItem
	plazaByID    map[string]plazaCatalogItem
	plazaLoadErr error
)

func loadPlazaCatalog() {
	plazaOnce.Do(func() {
		plazaByID = map[string]plazaCatalogItem{}
		if len(plazaCatalogJSON) == 0 {
			return
		}
		if err := json.Unmarshal(plazaCatalogJSON, &plazaItems); err != nil {
			plazaLoadErr = err
			return
		}
		for _, item := range plazaItems {
			plazaByID[item.ID] = item
		}
	})
}

func builtinOverrides() map[string]CatalogEntry {
	out := map[string]CatalogEntry{}
	for _, entry := range legacyBuiltinCatalog() {
		entry.Downloadable = true
		if entry.HfURL == "" && len(entry.Files) > 0 {
			entry.HfURL = entry.Files[0].URL
		}
		out[entry.ID] = entry
	}
	return out
}

func plazaItemToEntry(item plazaCatalogItem, override *CatalogEntry) CatalogEntry {
	kind := ModelKindChat
	if item.Kind == "embedding" {
		kind = ModelKindEmbedding
	}
	entry := CatalogEntry{
		ID:            item.ID,
		Kind:          kind,
		Name:          item.Name,
		Description:   item.Description,
		Builtin:       item.Builtin,
		Multimodal:    item.Multimodal,
		ParamsB:       item.ParamsB,
		ActiveParamsB: item.ActiveParamsB,
		ContextLength: item.ContextLength,
		Capabilities:  item.Capabilities,
		Provider:      item.Provider,
		License:       item.License,
		ReleasedAt:    item.ReleasedAt,
		Architecture:  item.Architecture,
		Quantization:  item.Quantization,
		ToolCalling:   item.ToolCalling,
		OllamaName:    item.OllamaName,
		HfURL:         strings.TrimSpace(item.HfURL),
		Downloadable:  item.Downloadable,
	}
	if override != nil {
		entry = *override
		entry.ID = item.ID
		if entry.Name == "" {
			entry.Name = item.Name
		}
		if entry.Description == "" {
			entry.Description = item.Description
		}
		if entry.HfURL == "" {
			entry.HfURL = strings.TrimSpace(item.HfURL)
		}
		entry.Downloadable = true
	}
	if entry.Quantization == "" {
		entry.Quantization = "Q4_K_M"
	}
	return entry
}

// AllCatalogEntries returns the merged CanIRun.ai plaza catalog with builtin overrides.
func AllCatalogEntries() []CatalogEntry {
	loadPlazaCatalog()
	overrides := builtinOverrides()
	if plazaLoadErr != nil || len(plazaItems) == 0 {
		out := make([]CatalogEntry, 0, len(overrides))
		for _, entry := range legacyBuiltinCatalog() {
			entry.Downloadable = true
			out = append(out, entry)
		}
		return out
	}
	out := make([]CatalogEntry, 0, len(plazaItems))
	for _, item := range plazaItems {
		var override *CatalogEntry
		if o, ok := overrides[item.ID]; ok {
			copy := o
			override = &copy
		}
		out = append(out, plazaItemToEntry(item, override))
	}
	return out
}

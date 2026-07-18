package embeddedmodels

// ListCatalog returns catalog entries merged with install/runtime status.
func ListCatalog(env func(string) string) []map[string]interface{} {
	RefreshSideloadCatalog(env)

	catalogIDs := map[string]struct{}{}
	out := make([]map[string]interface{}, 0, len(AllCatalogEntries())+len(sideloadSnapshot()))
	for _, entry := range AllCatalogEntries() {
		catalogIDs[entry.ID] = struct{}{}
		out = append(out, catalogEntryStatus(env, entry))
	}

	for id, entry := range sideloadSnapshot() {
		if _, ok := catalogIDs[id]; ok {
			continue
		}
		out = append(out, catalogEntryStatus(env, entry))
	}

	active := map[string]runtimeSnapshot{}
	for _, snap := range listRuntimeSnapshots() {
		active[snap.ModelID] = snap
	}
	for i := range out {
		id, _ := out[i]["id"].(string)
		if snap, ok := active[id]; ok {
			out[i]["running"] = true
			out[i]["port"] = snap.Port
			out[i]["endpoint"] = snap.Endpoint
		} else if running, _ := out[i]["running"].(bool); running {
			out[i]["running"] = false
		}
	}
	return out
}

func catalogEntryStatus(env func(string) string, entry CatalogEntry) map[string]interface{} {
	item := map[string]interface{}{
		"id":            entry.ID,
		"kind":          entry.Kind,
		"kindLabel":     KindLabel(entry.Kind),
		"name":          entry.Name,
		"description":   entry.Description,
		"tags":          entry.Tags,
		"builtin":       entry.Builtin,
		"multimodal":    entry.Multimodal,
		"files":         entry.Files,
		"installed":     IsInstalled(env, entry.ID),
		"paramsB":       entry.ParamsB,
		"activeParamsB": entry.ActiveParamsB,
		"contextLength": entry.ContextLength,
		"capabilities":  entry.Capabilities,
		"provider":      entry.Provider,
		"license":       entry.License,
		"releasedAt":    entry.ReleasedAt,
		"architecture":  entry.Architecture,
		"quantization":  entry.Quantization,
		"toolCalling":   entry.ToolCalling,
		"ollamaName":    entry.OllamaName,
		"hfUrl":         entry.HfURL,
		"downloadable":  entry.Downloadable,
		"sideloaded":    entry.Sideloaded,
	}
	st, ok := getInstalledState(env, entry.ID)
	if ok {
		item["running"] = st.Running
		item["port"] = st.Port
		item["lastError"] = st.LastError
		if !st.InstalledAt.IsZero() {
			item["installedAt"] = st.InstalledAt
		}
	} else {
		item["running"] = false
	}
	return item
}

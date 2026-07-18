package handlers

// ModelsListResult is the models.list response format.
type ModelsListResult struct {
	Models []ModelEntry `json:"models"`
}

// ModelEntry is a single model entry (compatible with protocol).
type ModelEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ModelsListHandler handles "models.list".
func ModelsListHandler(opts HandlerOpts) error {
	ctx := opts.Context
	var models []ModelEntry
	if ctx != nil && ctx.LoadModelCatalog != nil {
		models = ctx.LoadModelCatalog()
	}
	if models == nil {
		models = []ModelEntry{}
	}
	opts.Respond(true, &ModelsListResult{Models: models}, nil, nil)
	return nil
}

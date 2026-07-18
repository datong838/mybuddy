package embeddedmodels

// ModelKind distinguishes chat (generative) models from embedding (vector) models.
type ModelKind string

const (
	ModelKindChat      ModelKind = "chat"
	ModelKindEmbedding ModelKind = "embedding"
)

// ModelFile is a downloadable artifact (GGUF weight or mmproj).
type ModelFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size,omitempty"`
}

// CatalogEntry describes a model available in the model plaza.
type CatalogEntry struct {
	ID            string      `json:"id"`
	Kind          ModelKind   `json:"kind"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Tags          []string    `json:"tags"`
	Builtin       bool        `json:"builtin"`
	Multimodal    bool        `json:"multimodal"`
	Files         []ModelFile `json:"files"`
	ParamsB       float64     `json:"paramsB,omitempty"`
	ActiveParamsB float64     `json:"activeParamsB,omitempty"`
	ContextLength int         `json:"contextLength,omitempty"`
	Capabilities  []string    `json:"capabilities,omitempty"`
	Provider      string      `json:"provider,omitempty"`
	License       string      `json:"license,omitempty"`
	ReleasedAt    string      `json:"releasedAt,omitempty"`
	Architecture  string      `json:"architecture,omitempty"`
	Quantization  string      `json:"quantization,omitempty"`
	ToolCalling   bool        `json:"toolCalling,omitempty"`
	OllamaName    string      `json:"ollamaName,omitempty"`
	HfURL         string      `json:"hfUrl,omitempty"`
	Downloadable  bool        `json:"downloadable,omitempty"`
	Sideloaded    bool        `json:"sideloaded,omitempty"`
}

// legacyBuiltinCatalog returns hard-coded entries with explicit GGUF URLs (incl. mmproj).
func legacyBuiltinCatalog() []CatalogEntry {
	return []CatalogEntry{
		{
			ID:            "qwen3-0.6b",
			Kind:          ModelKindChat,
			Name:          "Qwen3-0.6B",
			Description:   "阿里通义千问 3 系列 0.6B 对话模型，中文理解优秀，适合本地轻量对话与工具调用。",
			Tags:          []string{"Chat", "中文", "文本", "内置", "推荐"},
			Builtin:       true,
			Multimodal:    false,
			ParamsB:       0.6,
			ContextLength: 32768,
			Capabilities:  []string{"chat", "code", "reasoning"},
			Provider:      "Alibaba",
			License:       "Apache 2.0",
			ReleasedAt:    "2025-04",
			Architecture:  "Dense",
			Quantization:  "Q4_K_M",
			ToolCalling:   true,
			OllamaName:    "qwen3:0.6b",
			Files: []ModelFile{
				{
					Name: "Qwen3-0.6B-Q4_K_M.gguf",
					URL:  "https://huggingface.co/lmstudio-community/Qwen3-0.6B-GGUF/resolve/main/Qwen3-0.6B-Q4_K_M.gguf",
					Size: 484 * 1024 * 1024,
				},
			},
		},
		{
			ID:            "qwen3-embedding-0.6b",
			Kind:          ModelKindEmbedding,
			Name:          "Qwen3-Embedding-0.6B",
			Description:   "通义千问 3 系列 0.6B 向量模型，支持 100+ 语言，适用于知识库检索、语义搜索与文本聚类。",
			Tags:          []string{"Embedding", "中文", "向量", "内置", "推荐"},
			Builtin:       true,
			Multimodal:    false,
			ParamsB:       0.6,
			ContextLength: 8192,
			Capabilities:  []string{"embedding", "rag"},
			Provider:      "Alibaba",
			License:       "Apache 2.0",
			ReleasedAt:    "2025-04",
			Architecture:  "Dense",
			Quantization:  "Q8_0",
			OllamaName:    "qwen3-embedding:0.6b",
			Files: []ModelFile{
				{
					Name: "Qwen3-Embedding-0.6B-Q8_0.gguf",
					URL:  "https://huggingface.co/Qwen/Qwen3-Embedding-0.6B-GGUF/resolve/main/Qwen3-Embedding-0.6B-Q8_0.gguf",
					Size: 639 * 1024 * 1024,
				},
			},
		},
		{
			ID:            "qwen2.5-vl-3b",
			Kind:          ModelKindChat,
			Name:          "Qwen2.5-VL-3B",
			Description:   "通义千问 2.5 视觉语言对话模型，支持图像理解与多模态对话，适合 OCR、识图等场景。",
			Tags:          []string{"Chat", "中文", "多模态", "视觉"},
			Builtin:       false,
			Multimodal:    true,
			ParamsB:       3,
			ContextLength: 32768,
			Capabilities:  []string{"chat", "vision", "code"},
			Provider:      "Alibaba",
			License:       "Apache 2.0",
			ReleasedAt:    "2024-11",
			Architecture:  "Dense",
			Quantization:  "Q4_K_M",
			ToolCalling:   true,
			OllamaName:    "qwen2.5vl:3b",
			Files: []ModelFile{
				{
					Name: "Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf",
					URL:  "https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF/resolve/main/Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf",
					Size: 2100 * 1024 * 1024,
				},
				{
					Name: "mmproj-Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf",
					URL:  "https://huggingface.co/ggml-org/Qwen2.5-VL-3B-Instruct-GGUF/resolve/main/mmproj-Qwen2.5-VL-3B-Instruct-Q4_K_M.gguf",
					Size: 400 * 1024 * 1024,
				},
			},
		},
		{
			ID:            "smollm2-135m",
			Kind:          ModelKindChat,
			Name:          "SmolLM2-135M",
			Description:   "超轻量英文对话模型，体积极小，适合快速验证内嵌 Chat 推理链路。",
			Tags:          []string{"Chat", "英文", "文本", "极小"},
			Builtin:       false,
			Multimodal:    false,
			ParamsB:       0.135,
			ContextLength: 8192,
			Capabilities:  []string{"chat", "edge"},
			Provider:      "HuggingFace",
			License:       "Apache 2.0",
			ReleasedAt:    "2024-09",
			Architecture:  "Dense",
			Quantization:  "Q4_K_M",
			OllamaName:    "smollm2:135m",
			Files: []ModelFile{
				{
					Name: "SmolLM2-135M.Q4_K_M.gguf",
					URL:  "https://huggingface.co/QuantFactory/SmolLM2-135M-GGUF/resolve/main/SmolLM2-135M.Q4_K_M.gguf",
					Size: 90 * 1024 * 1024,
				},
			},
		},
	}
}

// FindCatalogEntry returns a catalog entry by id.
func FindCatalogEntry(id string) (CatalogEntry, bool) {
	for _, e := range AllCatalogEntries() {
		if e.ID == id {
			return e, true
		}
	}
	return CatalogEntry{}, false
}

// CatalogCount returns the number of models in the plaza catalog.
func CatalogCount() int {
	return len(AllCatalogEntries())
}

// KindLabel returns a human-readable label for a model kind.
func KindLabel(kind ModelKind) string {
	switch kind {
	case ModelKindEmbedding:
		return "Embedding 向量"
	default:
		return "Chat 对话"
	}
}

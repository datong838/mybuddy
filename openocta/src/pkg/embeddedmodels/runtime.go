package embeddedmodels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/message"
	"github.com/hybridgroup/yzma/pkg/template"
)

const (
	defaultBasePort    = 18902
	defaultContextSize = 8192  // fallback when catalog has no contextLength
	defaultMaxTokens   = 32000 // fallback max output when context is unknown
	defaultBatchSize   = 2048
)

// catalogLimits returns context window and max output tokens from catalog metadata.
func catalogLimits(entry CatalogEntry) (contextWindow, maxOutput int) {
	contextWindow = entry.ContextLength
	if contextWindow <= 0 {
		contextWindow = defaultContextSize
	}
	maxOutput = contextWindow / 2
	if maxOutput <= 0 {
		maxOutput = defaultMaxTokens
	}
	return contextWindow, maxOutput
}

type runtimeInstance struct {
	mu        sync.Mutex
	modelID   string
	kind      ModelKind
	port      int
	server    *http.Server
	cancel    context.CancelFunc
	model     llama.Model
	lctx      llama.Context
	sampler   llama.Sampler
	vocab     llama.Vocab
	embedDim  int
	modelPath string
	libLoaded bool
}

// RuntimeStatus returns embedded model runtime state (supports multiple concurrent models).
func RuntimeStatus(env func(string) string) map[string]interface{} {
	snaps := listRuntimeSnapshots()
	models := make([]map[string]interface{}, 0, len(snaps))
	for _, s := range snaps {
		models = append(models, map[string]interface{}{
			"modelId":  s.ModelID,
			"kind":     s.Kind,
			"port":     s.Port,
			"endpoint": s.Endpoint,
		})
	}
	out := map[string]interface{}{
		"ok":      true,
		"running": len(models) > 0,
		"count":   len(models),
		"models":  models,
	}
	if len(models) > 0 {
		last := models[len(models)-1]
		out["modelId"] = last["modelId"]
		out["kind"] = last["kind"]
		out["port"] = last["port"]
		out["endpoint"] = last["endpoint"]
	}
	installed, _ := ListInstalled(env)
	out["installed"] = installed
	return out
}

// StartModel loads a GGUF model and serves a minimal OpenAI-compatible chat API.
func StartModel(env func(string) string, modelID string) (port int, err error) {
	return startModel(env, modelID, 0)
}

func startModel(env func(string) string, modelID string, preferredPort int) (port int, err error) {
	RefreshSideloadCatalog(env)
	if !IsInstalled(env, modelID) {
		return 0, fmt.Errorf("模型未安装，请先下载")
	}
	entry, ok := ResolveCatalogEntry(modelID)
	if !ok {
		return 0, fmt.Errorf("未知模型: %s", modelID)
	}

	if inst, ok := getRuntime(modelID); ok {
		inst.mu.Lock()
		running := inst.server != nil
		port := inst.port
		inst.mu.Unlock()
		if running {
			return port, nil
		}
	}

	if err := EnsureLibraries(env); err != nil {
		return 0, fmt.Errorf("推理引擎未就绪: %w", err)
	}

	modelPath, err := resolvePrimaryWeightPath(env, entry)
	if err != nil {
		return 0, err
	}

	libDir := ResolveLibDir(env)
	inst := &runtimeInstance{modelID: modelID, kind: entry.Kind, modelPath: modelPath}

	if err := acquireLlamaLib(libDir); err != nil {
		return 0, fmt.Errorf("加载推理库失败: %w", err)
	}
	inst.libLoaded = true

	mParams := llama.ModelDefaultParams()
	model, err := llama.ModelLoadFromFile(modelPath, mParams)
	if err != nil || model == 0 {
		inst.cleanup()
		return 0, fmt.Errorf("加载模型失败: %w", err)
	}
	inst.model = model
	inst.vocab = llama.ModelGetVocab(model)

	ctxParams := llama.ContextDefaultParams()
	contextSize, _ := catalogLimits(entry)
	ctxParams.NCtx = uint32(contextSize)
	ctxParams.NBatch = defaultBatchSize
	ctxParams.NUbatch = defaultBatchSize
	if entry.Kind == ModelKindEmbedding {
		ctxParams.Embeddings = 1
		ctxParams.PoolingType = llama.PoolingTypeLast
	}
	lctx, err := llama.InitFromModel(model, ctxParams)
	if err != nil {
		inst.cleanup()
		return 0, fmt.Errorf("初始化推理上下文失败: %w", err)
	}
	inst.lctx = lctx
	inst.embedDim = int(llama.ModelNEmbd(model))

	if entry.Kind == ModelKindChat {
		sp := llama.DefaultSamplerParams()
		sp.Temp = 0.7
		sp.TopK = 40
		sp.TopP = 0.9
		samplers := []llama.SamplerType{llama.SamplerTypeTopK, llama.SamplerTypeTopP, llama.SamplerTypeTemperature}
		inst.sampler = llama.NewSampler(model, samplers, sp)
	}

	p, ln, err := listenPortExcluding(preferredPort, defaultBasePort, usedRuntimePorts())
	if err != nil {
		inst.cleanup()
		return 0, err
	}
	inst.port = p

	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"data": []map[string]string{
				{"id": entry.ID, "object": "model"},
			},
		})
	})
	mux.HandleFunc("POST /v1/chat/completions", inst.handleChat)
	mux.HandleFunc("POST /v1/embeddings", inst.handleEmbeddings)

	srvCtx, cancel := context.WithCancel(context.Background())
	inst.cancel = cancel
	inst.server = &http.Server{
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context {
			return srvCtx
		},
	}

	go func() {
		_ = inst.server.Serve(ln)
	}()

	registerRuntime(inst)

	_ = setRuntimeState(env, modelID, true, p, "")
	return p, nil
}

// StopModel stops one or all embedded runtimes. Pass empty modelID to stop all.
func StopModel(env func(string) string, modelID string) error {
	if modelID == "" {
		snaps := listRuntimeSnapshots()
		for _, snap := range snaps {
			if err := stopOneRuntime(env, snap.ModelID); err != nil {
				return err
			}
		}
		return nil
	}
	return stopOneRuntime(env, modelID)
}

func stopOneRuntime(env func(string) string, modelID string) error {
	activeRuntimeMu.Lock()
	inst, ok := activeRuntimes[modelID]
	if !ok {
		activeRuntimeMu.Unlock()
		return fmt.Errorf("该模型未在运行")
	}
	delete(activeRuntimes, modelID)
	activeRuntimeMu.Unlock()

	port := inst.port
	id := inst.modelID
	inst.shutdown()
	_ = setRuntimeState(env, id, false, port, "")
	return nil
}

func (inst *runtimeInstance) shutdown() {
	if inst.cancel != nil {
		inst.cancel()
	}
	if inst.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = inst.server.Shutdown(ctx)
		cancel()
	}
	inst.cleanup()
}

func (inst *runtimeInstance) cleanup() {
	if inst.sampler != 0 {
		llama.SamplerFree(inst.sampler)
		inst.sampler = 0
	}
	if inst.lctx != 0 {
		llama.Free(inst.lctx)
		inst.lctx = 0
	}
	if inst.model != 0 {
		llama.ModelFree(inst.model)
		inst.model = 0
	}
	if inst.libLoaded {
		releaseLlamaLib()
		inst.libLoaded = false
	}
}

func isMMProj(name string) bool {
	return len(name) >= 6 && name[:6] == "mmproj"
}

func chatRequestWantsStream(stream *bool) bool {
	return stream != nil && *stream
}

func (inst *runtimeInstance) handleChat(w http.ResponseWriter, r *http.Request) {
	if inst.kind == ModelKindEmbedding {
		http.Error(w, "当前运行的是 Embedding 模型，请使用 /v1/embeddings", http.StatusBadRequest)
		return
	}
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = inst.defaultMaxOutputTokens()
	}
	enableThinking := chatRequestEnableThinking(req.Thinking)
	includeTools := chatRequestWantsTools(req)

	inst.mu.Lock()
	gen, err := inst.generateChatContent(req.Messages, req.Tools, includeTools, maxTokens, enableThinking)
	inst.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	modelName := strings.TrimSpace(req.Model)
	if modelName == "" {
		modelName = inst.modelID
	}
	created := time.Now().Unix()

	if chatRequestWantsStream(req.Stream) {
		inst.writeChatStream(w, modelName, created, gen)
		return
	}
	inst.writeChatJSON(w, modelName, created, gen)
}

func decodePromptTokens(lctx llama.Context, tokens []llama.Token) error {
	if len(tokens) == 0 {
		return nil
	}
	nBatch := int(llama.NBatch(lctx))
	if nBatch <= 0 {
		nBatch = int(defaultBatchSize)
	}
	for offset := 0; offset < len(tokens); {
		end := offset + nBatch
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := llama.BatchGetOne(tokens[offset:end])
		if _, err := llama.Decode(lctx, batch); err != nil {
			return err
		}
		offset = end
	}
	return nil
}

func (inst *runtimeInstance) resetChatState() {
	if mem, err := llama.GetMemory(inst.lctx); err == nil {
		_ = llama.MemoryClear(mem, true)
	}
	if inst.sampler != 0 {
		llama.SamplerReset(inst.sampler)
	}
}

func (inst *runtimeInstance) defaultMaxOutputTokens() int {
	if entry, ok := ResolveCatalogEntry(inst.modelID); ok {
		_, maxOutput := catalogLimits(entry)
		return maxOutput
	}
	return defaultMaxTokens
}

func chatRequestEnableThinking(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	var asBool bool
	if err := json.Unmarshal(raw, &asBool); err == nil {
		return asBool
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		s := strings.ToLower(strings.TrimSpace(asString))
		if s == "" || s == "off" || s == "false" || s == "disabled" || s == "none" {
			return false
		}
		return true
	}
	return true
}

func (inst *runtimeInstance) generateChatContent(
	messages []chatMessage,
	tools []openAIChatTool,
	includeTools bool,
	maxTokens int,
	enableThinking bool,
) (chatGenerationResult, error) {
	inst.resetChatState()

	chatMsgs, err := chatMessagesToYzma(messages, tools, includeTools)
	if err != nil {
		return chatGenerationResult{}, err
	}
	tmpl := llama.ModelChatTemplate(inst.model, "")
	if tmpl == "" {
		var ok bool
		tmpl, ok = template.BuiltinTemplate("chatml")
		if !ok {
			tmpl = "chatml"
		}
	}
	prompt, err := template.ApplyWithOptions(tmpl, chatMsgs, true, template.Options{
		EnableThinking: enableThinking,
	})
	if err != nil {
		return chatGenerationResult{}, fmt.Errorf("构建对话模板失败: %w", err)
	}

	tokens := llama.Tokenize(inst.vocab, prompt, true, true)
	promptTokens := len(tokens)
	if promptTokens == 0 {
		return chatGenerationResult{}, fmt.Errorf("无法编码输入")
	}
	if nCtx := int(llama.NCtx(inst.lctx)); promptTokens > nCtx {
		return chatGenerationResult{}, fmt.Errorf("输入过长（%d tokens），超过上下文限制（%d）", promptTokens, nCtx)
	}
	if err := decodePromptTokens(inst.lctx, tokens); err != nil {
		return chatGenerationResult{}, fmt.Errorf("推理失败: %w", err)
	}

	var content strings.Builder
	finishReason := "stop"
	completionTokens := 0
	for i := 0; i < maxTokens; i++ {
		token := llama.SamplerSample(inst.sampler, inst.lctx, -1)
		if llama.VocabIsEOG(inst.vocab, token) {
			break
		}

		pieceBuf := make([]byte, 256)
		pieceLen := llama.TokenToPiece(inst.vocab, token, pieceBuf, 0, true)
		if pieceLen > 0 {
			content.WriteString(string(pieceBuf[:pieceLen]))
		}
		completionTokens++
		if _, err := llama.Decode(inst.lctx, llama.BatchGetOne([]llama.Token{token})); err != nil {
			break
		}
		if i == maxTokens-1 {
			finishReason = "length"
		}
	}

	raw := content.String()
	parsedCalls := message.ParseToolCalls(raw)
	result := chatGenerationResult{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		FinishReason:     finishReason,
	}
	if len(parsedCalls) > 0 {
		result.ToolCalls = yzmaToolCallsToOpenAI(parsedCalls)
		result.Content = strings.TrimSpace(message.StripMarkup(raw))
		result.FinishReason = "tool_calls"
		return result, nil
	}
	result.Content = raw
	return result, nil
}

func (inst *runtimeInstance) writeChatJSON(w http.ResponseWriter, modelName string, created int64, gen chatGenerationResult) {
	if gen.FinishReason == "" {
		gen.FinishReason = "stop"
	}
	msg := map[string]interface{}{
		"role": "assistant",
	}
	if gen.Content != "" {
		msg["content"] = gen.Content
	} else if len(gen.ToolCalls) > 0 {
		msg["content"] = nil
	} else {
		msg["content"] = ""
	}
	if len(gen.ToolCalls) > 0 {
		msg["tool_calls"] = gen.ToolCalls
	}
	usage := chatUsage{
		PromptTokens:     gen.PromptTokens,
		CompletionTokens: gen.CompletionTokens,
		TotalTokens:      gen.PromptTokens + gen.CompletionTokens,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      "embedded",
		"object":  "chat.completion",
		"created": created,
		"model":   modelName,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"message":       msg,
				"finish_reason": gen.FinishReason,
			},
		},
		"usage": usage,
	})
}

func (inst *runtimeInstance) writeChatStream(w http.ResponseWriter, modelName string, created int64, gen chatGenerationResult) {
	if gen.FinishReason == "" {
		gen.FinishReason = "stop"
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Emit one chunk with full content for minimal SSE compatibility.
	deltaMsg := map[string]interface{}{}
	if gen.Content != "" {
		deltaMsg["content"] = gen.Content
	}
	if len(gen.ToolCalls) > 0 {
		deltaMsg["tool_calls"] = gen.ToolCalls
	}
	if len(deltaMsg) > 0 {
		delta := map[string]interface{}{
			"id":      "embedded",
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   modelName,
			"choices": []map[string]interface{}{
				{"index": 0, "delta": deltaMsg, "finish_reason": nil},
			},
		}
		b, _ := json.Marshal(delta)
		_, _ = io.WriteString(w, "data: "+string(b)+"\n\n")
		flusher.Flush()
	}

	done := map[string]interface{}{
		"id":      "embedded",
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   modelName,
		"choices": []map[string]interface{}{
			{"index": 0, "delta": map[string]string{}, "finish_reason": gen.FinishReason},
		},
	}
	b, _ := json.Marshal(done)
	_, _ = io.WriteString(w, "data: "+string(b)+"\n\n")
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	flusher.Flush()
}

type embeddingRequest struct {
	Model string      `json:"model"`
	Input interface{} `json:"input"`
}

func (inst *runtimeInstance) handleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if inst.kind != ModelKindEmbedding {
		http.Error(w, "当前运行的是 Chat 模型，请使用 /v1/chat/completions", http.StatusBadRequest)
		return
	}
	var req embeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	inputs := flattenEmbeddingInputs(req.Input)
	if len(inputs) == 0 {
		http.Error(w, "input 不能为空", http.StatusBadRequest)
		return
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	data := make([]map[string]interface{}, 0, len(inputs))
	for i, text := range inputs {
		vec, err := inst.embedText(text)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data = append(data, map[string]interface{}{
			"object":    "embedding",
			"index":     i,
			"embedding": vec,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   data,
		"model":  inst.modelID,
		"usage": map[string]interface{}{
			"prompt_tokens": len(inputs),
			"total_tokens":  len(inputs),
		},
	})
}

func flattenEmbeddingInputs(input interface{}) []string {
	switch v := input.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func (inst *runtimeInstance) embedText(text string) ([]float32, error) {
	tokens := llama.Tokenize(inst.vocab, text, true, true)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("无法编码输入")
	}
	if nCtx := int(llama.NCtx(inst.lctx)); len(tokens) > nCtx {
		return nil, fmt.Errorf("输入过长（%d tokens），超过上下文限制（%d）", len(tokens), nCtx)
	}
	if err := decodePromptTokens(inst.lctx, tokens); err != nil {
		return nil, fmt.Errorf("推理失败: %w", err)
	}
	dim := inst.embedDim
	if dim <= 0 {
		dim = int(llama.ModelNEmbd(inst.model))
	}
	embs, err := llama.GetEmbeddings(inst.lctx, 1, dim)
	if err != nil || len(embs) == 0 {
		// fallback: last-token pooling
		embs, err = llama.GetEmbeddingsIth(inst.lctx, -1, int32(dim))
		if err != nil {
			return nil, fmt.Errorf("生成向量失败: %w", err)
		}
	}
	if len(embs) > dim {
		embs = embs[:dim]
	}
	return embs, nil
}

// ProviderConfig returns config patch fields for a running embedded model.
func ProviderConfig(port int, modelID string) map[string]interface{} {
	entry, ok := ResolveCatalogEntry(modelID)
	name := modelID
	kind := ModelKindChat
	contextWindow := defaultContextSize
	maxOutput := defaultMaxTokens
	if ok {
		name = entry.Name
		kind = entry.Kind
		contextWindow, maxOutput = catalogLimits(entry)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d/v1", port)
	providerKey := "openocta-embedded-chat"
	displayPrefix := "内嵌对话"
	capabilities := "chat"
	if kind == ModelKindEmbedding {
		providerKey = "openocta-embedded-embedding"
		displayPrefix = "内嵌向量"
		capabilities = "embedding"
	}
	modelEntry := map[string]interface{}{
		"id":            modelID,
		"name":          name,
		"contextWindow": contextWindow,
		"capabilities":  capabilities,
	}
	if kind == ModelKindChat {
		modelEntry["maxTokens"] = maxOutput
	}
	return map[string]interface{}{
		providerKey: map[string]interface{}{
			"baseUrl":     baseURL,
			"apiKey":      "local",
			"displayName": displayPrefix + " · " + name,
			"models":      []map[string]interface{}{modelEntry},
		},
	}
}

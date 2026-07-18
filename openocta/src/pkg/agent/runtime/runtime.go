// Package runtime wraps Eino DeepAgent for OPENOCTA agent execution.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tencent-connect/botgo/log"

	"github.com/openocta/openocta/pkg/agent/eino"
	"github.com/openocta/openocta/pkg/agent/evolution"
	"github.com/openocta/openocta/pkg/agent/stream"
	"github.com/openocta/openocta/pkg/agent/tool"
	agenttools "github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/agent/types"
	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/paths"
)

// Runtime wraps the Eino engine for OPENOCTA.
type Runtime struct {
	eng            *eino.Engine
	agentRunBudget time.Duration
	turnMu         sync.Mutex
	turn           *eino.TurnSession
	turnSessionID  string
}

// KnowledgeOptions configures vault indexing (memory_search).
type KnowledgeOptions = eino.KnowledgeOptions

// New creates a new Runtime with the given options.
func New(ctx context.Context, opts Options) (*Runtime, error) {
	if opts.ModelFactory == nil {
		opts.ModelFactory = DefaultModelFactory()
	}
	projectRoot := opts.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}

	var tools []tool.Tool
	if !opts.DisableTools {
		if shouldRegisterWebTools(opts) {
			for _, t := range agenttools.WebToolsFromConfig(opts.Config, projectRoot) {
				tools = append(tools, t)
			}
		}
		if shouldRegisterBrowserTools(opts) {
			for _, t := range agenttools.BrowserToolsFromConfig(opts.Config) {
				tools = append(tools, t)
			}
		}
		if len(opts.Tools) > 0 {
			extra := agenttools.FilterOutWebTools(opts.Tools)
			if shouldRegisterBrowserTools(opts) {
				extra = agenttools.FilterOutBrowserTools(extra)
			}
			tools = append(tools, extra...)
		}
	}

	var instruction string
	if opts.EnableSystemPrompt {
		if opts.SystemPromptOverrides != "" {
			instruction = opts.SystemPromptOverrides
		} else {
			env := opts.Env
			if env == nil {
				env = func(string) string { return "" }
			}
			prompt, err := BuildSystemPrompt(SystemPromptOptions{
				WorkspaceDir: "",
				ProjectRoot:  projectRoot,
			})
			if err == nil {
				instruction = prompt
			}
		}
		if skillsSection := BuildSystemPromptSkillsSection(projectRoot, opts); skillsSection != "" && !einoSkillMiddlewareActive(projectRoot, opts) {
			if instruction != "" {
				instruction = strings.TrimSpace(instruction) + "\n\n" + skillsSection
			} else {
				instruction = skillsSection
			}
		}
		if knowledgeSection := BuildSystemPromptKnowledgeSection(opts.Config, opts.AgentID, opts.Env); knowledgeSection != "" {
			if instruction != "" {
				instruction = strings.TrimSpace(instruction) + "\n\n" + knowledgeSection
			} else {
				instruction = knowledgeSection
			}
		}
		instruction = augmentEvolutionPrompt(projectRoot, opts, instruction)
	}

	var validateCommand func(string) error
	if opts.Config != nil && opts.Config.Security != nil {
		if policy := ResolveCommandPolicy(opts.Config.Security); policy != nil && policy.Enabled {
			validatorCfg := policy.ToLegacyValidator()
			if validatorCfg != nil {
				validateCommand = func(cmd string) error {
					return ValidateCommandWithConfig(cmd, validatorCfg)
				}
			}
		}
	}

	var knowledge *eino.KnowledgeOptions
	if opts.Knowledge != nil {
		knowledge = opts.Knowledge
	} else {
		knowledge = resolveKnowledgeOptions(opts.Config, opts.Env, opts.AgentID)
	}

	skillsDir := ""
	if opts.EnableSkills {
		skillsDir = eino.ResolveSkillsDir(projectRoot, opts.Config, opts.EmployeeID, opts.SkillFilter, opts.Env)
	}

	eng, err := eino.NewEngine(ctx, eino.BuildConfig{
		ModelFactory:    opts.ModelFactory,
		Tools:           tools,
		ProjectRoot:     projectRoot,
		Instruction:     instruction,
		DisableTools:    opts.DisableTools,
		SkillsDir:       skillsDir,
		Knowledge:       knowledge,
		MaxIteration:    50,
		ValidateCommand: validateCommand,
		EnableApproval:  opts.EnableApprovalQueue,
		Env:             opts.Env,
		Config:          opts.Config,
		TokenLimit:      opts.TokenLimit,
		AgentID:         opts.AgentID,
		TokenTracking:   opts.TokenTracking,
	})
	if err != nil {
		return nil, err
	}
	runBudget := resolveAgentRunTimeout(opts)
	eng.SetAgentRunBudget(runBudget)

	// Approval queue settings file (Gateway UI compatibility).
	if opts.EnableApprovalQueue && opts.Config != nil && opts.Config.Security != nil && opts.Config.Security.ApprovalQueue != nil {
		if opts.Config.Security.ApprovalQueue.Enabled != nil && *opts.Config.Security.ApprovalQueue.Enabled {
			var permsOverride *ApprovalQueuePermsOverride
			if policy := ResolveCommandPolicy(opts.Config.Security); policy != nil && policy.Enabled {
				allow, ask, deny := policy.ToApprovalQueuePerms()
				permsOverride = &ApprovalQueuePermsOverride{Allow: allow, Ask: ask, Deny: deny}
			}
			if err := writeApprovalQueueSettings(opts.Env, opts.Config.Security.ApprovalQueue, permsOverride); err != nil {
				log.Errorf("Warning: failed to write approval queue settings: %v", err)
			}
		}
	}

	return &Runtime{eng: eng, agentRunBudget: runBudget}, nil
}

func augmentEvolutionPrompt(projectRoot string, opts Options, base string) string {
	evoDir := projectRoot
	if opts.Config != nil && opts.Config.Agents != nil && opts.Config.Agents.Defaults != nil {
		if ws := strings.TrimSpace(opts.Config.Agents.Defaults.Workspace); ws != "" {
			evoDir = ws
		}
	}
	if opts.Env != nil {
		if ws := strings.TrimSpace(opts.Env("OPENOCTA_WORKSPACE")); ws != "" {
			evoDir = ws
		}
	}
	dir := filepath.Join(evoDir, ".agents", "evolution")
	store, err := evolution.Open(evolution.Config{Dir: dir})
	if err != nil {
		return base
	}
	snap := store.SnapshotForSession("")
	return evolution.AugmentSystemPrompt(base, snap)
}

// ApprovalQueueEnabled reports whether security.approvalQueue.enabled is explicitly true.
func ApprovalQueueEnabled(cfg *config.OpenOctaConfig) bool {
	if cfg == nil || cfg.Security == nil || cfg.Security.ApprovalQueue == nil {
		return false
	}
	return cfg.Security.ApprovalQueue.Enabled != nil && *cfg.Security.ApprovalQueue.Enabled
}

// Options configures the Runtime.
type Options struct {
	ModelFactory          eino.ChatModelFactory
	Tools                 []tool.Tool
	ProjectRoot           string
	Config                *config.OpenOctaConfig
	EnableSkills          bool
	EmployeeID            string
	EnableSubagents       bool
	EnableSandbox         bool
	EnableApprovalQueue   bool
	TokenTracking         bool
	AgentID               string
	Env                   func(string) string
	EnableSystemPrompt    bool
	SystemPromptOverrides string
	Knowledge             *KnowledgeOptions
	AgentRunTimeout       time.Duration
	MiddlewareTimeout     time.Duration
	HookTimeout           time.Duration
	TokenLimit            int
	SkillFilter           *[]string
	McpServerFilter       *[]string
	EnableWebTools        *bool
	DisallowedTools       []string
	DisableTools          bool
	BashToolTimeout       time.Duration
	ParallelToolCalls     *bool
}

// Request is an alias for types.Request.
type Request = types.Request

// Response is an alias for types.Response.
type Response = types.Response

// StreamEvent is an alias for stream.StreamEvent.
type StreamEvent = stream.StreamEvent

func DefaultModelFactory() eino.ChatModelFactory {
	return eino.DefaultModelFactory()
}

func (r *Runtime) Run(ctx context.Context, req Request) (*Response, error) {
	if r == nil || r.eng == nil {
		return nil, eino.ErrRuntimeClosed
	}
	return r.eng.Run(ctx, req)
}

func (r *Runtime) RunStream(ctx context.Context, req Request) (<-chan StreamEvent, error) {
	if r == nil || r.eng == nil {
		return nil, eino.ErrRuntimeClosed
	}
	return r.eng.RunStream(ctx, req)
}

// RunStreamTurn executes via Eino TurnLoop (supports preempt / abort / resume).
func (r *Runtime) RunStreamTurn(ctx context.Context, req Request, runID string, preempt bool) (<-chan StreamEvent, error) {
	if r == nil || r.eng == nil {
		return nil, eino.ErrRuntimeClosed
	}
	ts := r.ensureTurnSession(strings.TrimSpace(req.SessionID))
	return ts.PushMessage(ctx, req, runID, preempt)
}

// ResumeStream continues an interrupted agent run after user approval.
func (r *Runtime) ResumeStream(ctx context.Context, sessionID, runID string, approval eino.TurnApproval) (<-chan StreamEvent, error) {
	if r == nil || r.eng == nil {
		return nil, eino.ErrRuntimeClosed
	}
	ts := r.ensureTurnSession(strings.TrimSpace(sessionID))
	return ts.PushApproval(ctx, runID, approval)
}

// ResumeStreamWithTargets resumes using explicit Eino interrupt target map.
func (r *Runtime) ResumeStreamWithTargets(ctx context.Context, sessionID, runID string, targets map[string]any) (<-chan StreamEvent, error) {
	if r == nil || r.eng == nil {
		return nil, eino.ErrRuntimeClosed
	}
	return r.eng.ResumeStream(ctx, sessionID, runID, targets)
}

// ClearSessionCheckpoints wipes Runner/TurnLoop checkpoint files for a session.
func (r *Runtime) ClearSessionCheckpoints(sessionID string) {
	if r == nil || r.eng == nil {
		eino.ClearPersistedSessionCheckpoints(sessionID)
		return
	}
	r.eng.ClearAllSessionCheckpoints(sessionID)
}

// AbortSessionTurn stops the session TurnLoop immediately (chat.abort).
func (r *Runtime) AbortSessionTurn() {
	if r == nil {
		return
	}
	r.turnMu.Lock()
	ts := r.turn
	r.turn = nil
	r.turnSessionID = ""
	r.turnMu.Unlock()
	if ts != nil {
		ts.StopImmediate()
	}
}

func (r *Runtime) ensureTurnSession(sessionID string) *eino.TurnSession {
	r.turnMu.Lock()
	defer r.turnMu.Unlock()
	if r.turn != nil && r.turnSessionID == sessionID {
		return r.turn
	}
	if r.turn != nil {
		r.turn.StopImmediate()
	}
	r.turn = eino.NewTurnSession(r.eng, sessionID)
	r.turnSessionID = sessionID
	return r.turn
}

func (r *Runtime) Close() error {
	if r == nil || r.eng == nil {
		return nil
	}
	r.AbortSessionTurn()
	return r.eng.Close()
}

func ClearSessionHistory(projectRoot, sessionID string) {
	projectRoot = strings.TrimSpace(projectRoot)
	sessionID = strings.TrimSpace(sessionID)
	if projectRoot == "" || sessionID == "" {
		return
	}
	name := historySessionFileName(sessionID)
	if name == "" {
		return
	}
	path := filepath.Join(projectRoot, ".claude", "history", name+".json")
	_ = os.Remove(path)
}

func historySessionFileName(sessionID string) string {
	const fallback = "default"
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return fallback
	}
	var b strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		return fallback
	}
	return sanitized
}

func shouldRegisterWebTools(_ Options) bool {
	// web_search / web_fetch temporarily disabled; see tools.webToolsEnabled.
	return false
}

func shouldRegisterBrowserTools(opts Options) bool {
	if opts.Config != nil && opts.Config.Browser != nil && opts.Config.Browser.Enabled != nil {
		return *opts.Config.Browser.Enabled
	}
	return true
}

func einoSkillMiddlewareActive(projectRoot string, opts Options) bool {
	if !opts.EnableSkills {
		return false
	}
	return strings.TrimSpace(eino.ResolveSkillsDir(projectRoot, opts.Config, opts.EmployeeID, opts.SkillFilter, opts.Env)) != ""
}

type SandboxOpts struct {
	Root              string
	AllowedPaths      []string
	ExtraAllowedPaths []string
	NetworkAllow      []string
}

type ApprovalQueuePermsOverride struct {
	Allow, Ask, Deny []string
}

func getSettingsPath(env func(string) string) string {
	if env == nil {
		env = func(string) string { return "" }
	}
	stateDir := paths.ResolveStateDir(env)
	return filepath.Join(stateDir, "workspace", ".claude", "settings.json")
}

func writeApprovalQueueSettings(env func(string) string, cfg *config.SandboxApprovalQueue, permsOverride *ApprovalQueuePermsOverride) error {
	settingsPath := getSettingsPath(env)
	perms := struct {
		DefaultMode                  string   `json:"defaultMode,omitempty"`
		DisableBypassPermissionsMode string   `json:"disableBypassPermissionsMode,omitempty"`
		Allow                        []string `json:"allow,omitempty"`
		Ask                          []string `json:"ask,omitempty"`
		Deny                         []string `json:"deny,omitempty"`
	}{
		DefaultMode:                  "bypassPermissions",
		DisableBypassPermissionsMode: "enable",
	}
	var allow, ask, deny []string
	if permsOverride != nil {
		allow, ask, deny = permsOverride.Allow, permsOverride.Ask, permsOverride.Deny
	} else if cfg != nil {
		allow, ask, deny = cfg.Allow, cfg.Ask, cfg.Deny
	}
	for _, cmd := range allow {
		if cmd != "" {
			perms.Allow = append(perms.Allow, fmt.Sprintf("Bash(%s:*)", cmd))
		}
	}
	for _, cmd := range ask {
		if cmd != "" {
			perms.Ask = append(perms.Ask, fmt.Sprintf("Bash(%s:*)", cmd))
		}
	}
	for _, cmd := range deny {
		if cmd != "" {
			perms.Deny = append(perms.Deny, fmt.Sprintf("Bash(%s:*)", cmd))
		}
	}
	settings := struct {
		Permissions *struct {
			DefaultMode                  string   `json:"defaultMode,omitempty"`
			DisableBypassPermissionsMode string   `json:"disableBypassPermissionsMode,omitempty"`
			Allow                        []string `json:"allow,omitempty"`
			Ask                          []string `json:"ask,omitempty"`
			Deny                         []string `json:"deny,omitempty"`
		} `json:"permissions,omitempty"`
	}{Permissions: &perms}
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0644)
}

func removeApprovalQueueSettings(env func(string) string) error {
	settingsPath := getSettingsPath(env)
	if _, err := os.Stat(settingsPath); err != nil {
		return nil
	}
	return os.Remove(settingsPath)
}

func resolveAgentRunTimeout(opts Options) time.Duration {
	if opts.AgentRunTimeout > 0 {
		return opts.AgentRunTimeout
	}
	if v := strings.TrimSpace(os.Getenv("OPENOCTA_AGENT_RUN_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 10 * time.Minute
}

var _ = runtime.GOOS

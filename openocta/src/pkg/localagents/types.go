// Package localagents detects and invokes external CLI-based AI agents on the host.
package localagents

// AgentProbeResult describes whether a local agent CLI is available.
type AgentProbeResult struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Installed   bool     `json:"installed"`
	Path        string   `json:"path,omitempty"`
	Version     string   `json:"version,omitempty"`
	ProbeMethod string   `json:"probeMethod"`
	InvokeHint  string   `json:"invokeHint"`
	Aliases     []string `json:"aliases"`
}

// ProbeReport is returned by localAgents.probe / localAgents.status.
type ProbeReport struct {
	Agents   []AgentProbeResult `json:"agents"`
	ProbedAt int64              `json:"probedAt"`
}

// TaskSegment is one @agent task parsed from a user message.
type TaskSegment struct {
	AgentID string `json:"agentId"`
	Task    string `json:"task"`
}

// SegmentResult is the outcome of running one segment.
type SegmentResult struct {
	AgentID string `json:"agentId"`
	Label   string `json:"label"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

package handlers

import (
	"fmt"
	"strings"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/localagents"
)

// BuildLocalAgentSystemHint returns extra system instructions when the user @mentions local CLI agents.
// The main agent should analyze intent and delegate via the local_agent tool instead of direct CLI invocation.
func BuildLocalAgentSystemHint(message string, cfg *config.OpenOctaConfig) string {
	if cfg != nil && cfg.LocalAgents != nil && cfg.LocalAgents.Enabled != nil && !*cfg.LocalAgents.Enabled {
		return ""
	}
	segments := localagents.ParseMessage(message)
	if len(segments) == 0 {
		return ""
	}

	var mentions strings.Builder
	for i, seg := range segments {
		if i > 0 {
			mentions.WriteString("; ")
		}
		mentions.WriteString("@" + seg.AgentID)
		if strings.TrimSpace(seg.Task) != "" {
			mentions.WriteString(": ")
			mentions.WriteString(strings.TrimSpace(seg.Task))
		}
	}

	return fmt.Sprintf(`## Local agent delegation (mandatory workflow)

The user message mentions local CLI agents (%s).

Rules:
1. Do NOT invoke openclaw, hermes, cursor, codex, opencode, or trae CLI commands directly in shell/bash.
2. First analyze the user intent, conversation context, and constraints. Expand vague requests into a complete, self-contained task description (paths, tech stack, acceptance criteria when relevant).
3. Call the **local_agent** tool to delegate:
   - action=status — check which agents are installed (optional, before run)
   - action=run — agent=<id>, task=<full task text>
   - action=run_many — tasks=[{agent, task}, ...] for parallel work
4. After tool output returns, summarize results for the user in plain language.
5. If the user only @mentions an agent without a clear task, ask clarifying questions before calling local_agent.

Refer to the **local-agents-collab** skill for per-agent CLI semantics and task-writing guidelines.`, mentions.String())
}

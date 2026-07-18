package eino

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
)

const executeToolSingleLineHint = `
## execute tool (command parameter)
- The "command" argument MUST be a single line with no newline characters.
- Chain multiple steps with ";" or "&&" on the same line (e.g. "mkdir dir && cd dir && npm test").
- Keep inline command short (aim for < 800 characters). If JSON arguments may exceed model output limits, do NOT embed large scripts in command.
- For long PowerShell/Bash: use write_file to save a .ps1/.sh script, then execute with powershell -NoProfile -ExecutionPolicy Bypass -File "<absolute-path>" (or bash "<path>").
- On Windows process/desktop tasks, prefer the desktop-control skill scripts (e.g. process-manager.ps1) instead of rewriting inline PowerShell.
- On Windows, avoid mega inline powershell -Command "..." ; cmd.exe and JSON escaping eat $ variables easily. Prefer -File scripts.
- Do not split one logical command across multiple lines in the JSON argument.
- If tool result says arguments JSON was truncated (finish_reason=length), shorten the approach — never retry with a longer inline command.
`

type executeToolHintMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func newExecuteToolHintMiddleware() adk.ChatModelAgentMiddleware {
	return &executeToolHintMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
}

func (m *executeToolHintMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	if runCtx == nil {
		return ctx, runCtx, nil
	}
	if strings.Contains(runCtx.Instruction, "execute tool (command parameter)") {
		return ctx, runCtx, nil
	}
	nRunCtx := *runCtx
	if nRunCtx.Instruction != "" {
		nRunCtx.Instruction = strings.TrimSpace(nRunCtx.Instruction) + "\n" + executeToolSingleLineHint
	} else {
		nRunCtx.Instruction = strings.TrimSpace(executeToolSingleLineHint)
	}
	return ctx, &nRunCtx, nil
}
